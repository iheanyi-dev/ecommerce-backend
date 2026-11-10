//! Configuration identifying an embedding model.

/// Identifies a pinned embedding model and its output dimension.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct EmbeddingModelManifest {
    pub model_id: String,
    pub revision: String,
    pub embedding_dimension: usize,
}

/// Selected multilingual E5-small model.
pub const MULTILINGUAL_E5_SMALL_MODEL_ID: &str = "intfloat/multilingual-e5-small";
pub const MULTILINGUAL_E5_SMALL_REVISION: &str = "614241f622f53c4eeff9890bdc4f31cfecc418b3";
pub const MULTILINGUAL_E5_SMALL_DIMENSION: usize = 384;

impl EmbeddingModelManifest {
    pub fn new(
        model_id: impl Into<String>,
        revision: impl Into<String>,
        embedding_dimension: usize,
    ) -> Result<Self, String> {
        let model_id = model_id.into();
        let revision = revision.into();

        if model_id.trim().is_empty() {
            return Err("model_id must not be empty".into());
        }

        if revision.trim().is_empty() {
            return Err("revision must not be empty".into());
        }

        if embedding_dimension == 0 {
            return Err("embedding_dimension must be positive".into());
        }

        Ok(Self {
            model_id,
            revision,
            embedding_dimension,
        })
    }
}

/// Returns the pinned multilingual E5-small manifest.
pub fn multilingual_e5_small() -> EmbeddingModelManifest {
    EmbeddingModelManifest {
        model_id: MULTILINGUAL_E5_SMALL_MODEL_ID.into(),
        revision: MULTILINGUAL_E5_SMALL_REVISION.into(),
        embedding_dimension: MULTILINGUAL_E5_SMALL_DIMENSION,
    }
}
