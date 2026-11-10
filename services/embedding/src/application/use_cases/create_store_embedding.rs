//! Use case for creating or replacing a store embedding.

use std::sync::Arc;

use crate::application::{
    dto::{CreateStoreEmbeddingInput, CreateStoreEmbeddingOutput},
    errors::{ApplicationError, ApplicationResult},
    ports::{EmbeddingModel, Logger, Metrics, StoreEmbeddingRepository},
};
use crate::domain::{
    entities::StoreEmbedding,
    value_objects::{EntityDescription, EntityName},
};

/// Creates and persists a store embedding.
pub struct CreateStoreEmbeddingUseCase {
    embedding_model: Arc<dyn EmbeddingModel>,
    repository: Arc<dyn StoreEmbeddingRepository>,
    logger: Arc<dyn Logger>,
    metrics: Arc<dyn Metrics>,
}

impl CreateStoreEmbeddingUseCase {
    /// Creates the use case with its required application dependencies.
    pub fn new(
        embedding_model: Arc<dyn EmbeddingModel>,
        repository: Arc<dyn StoreEmbeddingRepository>,
        logger: Arc<dyn Logger>,
        metrics: Arc<dyn Metrics>,
    ) -> Self {
        Self {
            embedding_model,
            repository,
            logger,
            metrics,
        }
    }

    /// Generates and persists the embedding for a store.
    pub fn execute(
        &self,
        input: CreateStoreEmbeddingInput,
    ) -> ApplicationResult<CreateStoreEmbeddingOutput> {
        let name = input.name.trim().to_owned();
        let description = input.description.trim().to_owned();

        if name.is_empty() {
            return Err(ApplicationError::Validation(
                "store name cannot be empty".into(),
            ));
        }

        if description.is_empty() {
            return Err(ApplicationError::Validation(
                "store description cannot be empty".into(),
            ));
        }

        self.logger.info(
            "embedding.store.create.started",
            &[("store_id", input.store_id.to_string())],
        );

        // Keep the text format consistent with the Python implementation.
        let text = format!("Store name: {name}\nStore description: {description}");

        let (embedding, model_version) = self.embedding_model.generate(&text)?;

        let store_embedding = StoreEmbedding::new(
            input.store_id,
            EntityName::new(name, "store")?,
            EntityDescription::new(description, "store")?,
            embedding.clone(),
            model_version.clone(),
        );

        self.repository.upsert(store_embedding)?;

        self.metrics
            .increment("embedding.store_embeddings.created", 1);

        self.logger.info(
            "embedding.store.create.succeeded",
            &[("store_id", input.store_id.to_string())],
        );

        Ok(CreateStoreEmbeddingOutput {
            store_id: input.store_id,
            embedding,
            embedding_model_version: model_version,
        })
    }
}
