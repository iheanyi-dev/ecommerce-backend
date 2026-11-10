//! Candle-backed multilingual E5 embedding model.

use std::{io, path::PathBuf, sync::Mutex};

use candle_core::{DType, Device, Tensor};
use candle_nn::VarBuilder;
use candle_transformers::models::bert::{BertModel, Config};
use hf_hub::{Repo, RepoType, api::sync::ApiBuilder};
use tokenizers::{Tokenizer, TruncationParams};

use crate::{
    application::ports::{EmbeddingModel, PortResult},
    domain::value_objects::{Embedding, EmbeddingModelVersion},
};

use super::model_manifest::EmbeddingModelManifest;

struct LoadedModel {
    tokenizer: Tokenizer,
    model: BertModel,
    device: Device,
}

pub struct CandleEmbeddingModel {
    cache_dir: PathBuf,
    manifest: EmbeddingModelManifest,
    loaded: Mutex<Option<LoadedModel>>,
}

impl CandleEmbeddingModel {
    pub fn new(cache_dir: impl Into<PathBuf>, manifest: EmbeddingModelManifest) -> Self {
        Self {
            cache_dir: cache_dir.into(),
            manifest,
            loaded: Mutex::new(None),
        }
    }

    fn load_model(&self) -> PortResult<LoadedModel> {
        let api = ApiBuilder::new()
            .with_cache_dir(self.cache_dir.clone())
            .build()?;

        let repo = api.repo(Repo::with_revision(
            self.manifest.model_id.clone(),
            RepoType::Model,
            self.manifest.revision.clone(),
        ));

        let config_path = repo.get("config.json")?;
        let tokenizer_path = repo.get("tokenizer.json")?;
        let weights_path = repo.get("model.safetensors")?;

        let config: Config = serde_json::from_slice(&std::fs::read(config_path)?)?;
        let tokenizer = Tokenizer::from_file(tokenizer_path).map_err(io::Error::other)?;

        let mut tokenizer = tokenizer;
        tokenizer
            .with_truncation(Some(TruncationParams {
                max_length: 512,
                ..Default::default()
            }))
            .map_err(io::Error::other)?;

        let device = Device::Cpu;
        let vb =
            unsafe { VarBuilder::from_mmaped_safetensors(&[weights_path], DType::F32, &device)? };

        let model = BertModel::load(vb, &config)?;

        Ok(LoadedModel {
            tokenizer,
            model,
            device,
        })
    }
}

impl EmbeddingModel for CandleEmbeddingModel {
    fn generate(&self, text: &str) -> PortResult<(Embedding, EmbeddingModelVersion)> {
        if text.trim().is_empty() {
            return Err(io::Error::other("Embedding text must not be empty").into());
        }

        let mut guard = self
            .loaded
            .lock()
            .map_err(|_| io::Error::other("Model lock poisoned"))?;

        if guard.is_none() {
            *guard = Some(self.load_model()?);
        }

        let loaded = guard.as_ref().expect("model was just loaded");

        let encoding = loaded
            .tokenizer
            .encode(text, true)
            .map_err(io::Error::other)?;

        let ids = encoding.get_ids();
        let masks = encoding.get_attention_mask();
        let type_ids = encoding.get_type_ids();
        let sequence_length = ids.len();

        let input_ids = Tensor::from_vec(ids.to_vec(), (1, sequence_length), &loaded.device)?;

        let attention_mask =
            Tensor::from_vec(masks.to_vec(), (1, sequence_length), &loaded.device)?;

        let token_type_ids =
            Tensor::from_vec(type_ids.to_vec(), (1, sequence_length), &loaded.device)?;

        let hidden_states =
            loaded
                .model
                .forward(&input_ids, &token_type_ids, Some(&attention_mask))?;

        let mask = attention_mask.to_dtype(DType::F32)?.unsqueeze(2)?;

        let pooled = hidden_states
            .broadcast_mul(&mask)?
            .sum(1)?
            .broadcast_div(&mask.sum(1)?)?;

        let mut values = pooled.to_vec2::<f32>()?.remove(0);

        let norm = values.iter().map(|value| value * value).sum::<f32>().sqrt();

        if !norm.is_finite() || norm == 0.0 {
            return Err(io::Error::other("Generated embedding has an invalid norm").into());
        }

        for value in &mut values {
            *value /= norm;
        }

        let embedding = Embedding::new(values)?;

        if embedding.dimension() != self.manifest.embedding_dimension {
            return Err(io::Error::other(format!(
                "Expected {} dimensions, got {}",
                self.manifest.embedding_dimension,
                embedding.dimension(),
            ))
            .into());
        }

        let version = EmbeddingModelVersion::new(format!(
            "{}@{}",
            self.manifest.model_id, self.manifest.revision,
        ))?;

        Ok((embedding, version))
    }
}
