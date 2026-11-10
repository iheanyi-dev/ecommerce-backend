//! Use case for creating or replacing a product embedding.

use std::sync::Arc;

use crate::application::{
    dto::{CreateProductEmbeddingInput, CreateProductEmbeddingOutput},
    errors::{ApplicationError, ApplicationResult},
    ports::{EmbeddingModel, Logger, Metrics, ProductEmbeddingRepository},
};
use crate::domain::{
    entities::ProductEmbedding,
    value_objects::{EntityDescription, EntityName},
};

/// Creates and persists a product embedding.
pub struct CreateProductEmbeddingUseCase {
    embedding_model: Arc<dyn EmbeddingModel>,
    repository: Arc<dyn ProductEmbeddingRepository>,
    logger: Arc<dyn Logger>,
    metrics: Arc<dyn Metrics>,
}

impl CreateProductEmbeddingUseCase {
    /// Creates the use case with its required application dependencies.
    pub fn new(
        embedding_model: Arc<dyn EmbeddingModel>,
        repository: Arc<dyn ProductEmbeddingRepository>,
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

    /// Generates and persists the embedding for a product.
    pub fn execute(
        &self,
        input: CreateProductEmbeddingInput,
    ) -> ApplicationResult<CreateProductEmbeddingOutput> {
        let name = input.name.trim().to_owned();
        let description = input.description.trim().to_owned();

        if name.is_empty() {
            return Err(ApplicationError::Validation(
                "product name cannot be empty".into(),
            ));
        }

        if description.is_empty() {
            return Err(ApplicationError::Validation(
                "product description cannot be empty".into(),
            ));
        }

        self.logger.info(
            "embedding.product.create.started",
            &[
                ("product_id", input.product_id.to_string()),
                ("store_id", input.store_id.to_string()),
            ],
        );

        // Keep the text format consistent with the Python implementation.
        let text = format!("Product name: {name}\nProduct description: {description}");

        let (embedding, model_version) = self.embedding_model.generate(&text)?;

        let product_embedding = ProductEmbedding::new(
            input.product_id,
            input.store_id,
            EntityName::new(name, "product")?,
            EntityDescription::new(description, "product")?,
            embedding.clone(),
            model_version.clone(),
        );

        self.repository.upsert(product_embedding)?;

        self.metrics
            .increment("embedding.product_embeddings.created", 1);

        self.logger.info(
            "embedding.product.create.succeeded",
            &[("product_id", input.product_id.to_string())],
        );

        Ok(CreateProductEmbeddingOutput {
            product_id: input.product_id,
            embedding,
            embedding_model_version: model_version,
        })
    }
}
