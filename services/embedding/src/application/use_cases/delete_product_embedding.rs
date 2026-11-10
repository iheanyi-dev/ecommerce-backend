//! Use case for deleting a product embedding.

use std::sync::Arc;

use crate::application::{
    dto::DeleteProductEmbeddingInput,
    errors::ApplicationResult,
    ports::{Logger, Metrics, ProductEmbeddingRepository},
};

/// Deletes a product's stored embedding.
pub struct DeleteProductEmbeddingUseCase {
    repository: Arc<dyn ProductEmbeddingRepository>,
    logger: Arc<dyn Logger>,
    metrics: Arc<dyn Metrics>,
}

impl DeleteProductEmbeddingUseCase {
    /// Creates the use case with its required dependencies.
    pub fn new(
        repository: Arc<dyn ProductEmbeddingRepository>,
        logger: Arc<dyn Logger>,
        metrics: Arc<dyn Metrics>,
    ) -> Self {
        Self {
            repository,
            logger,
            metrics,
        }
    }

    /// Deletes the embedding associated with the supplied product ID.
    pub fn execute(&self, input: DeleteProductEmbeddingInput) -> ApplicationResult<()> {
        self.logger.info(
            "embedding.product.delete.started",
            &[("product_id", input.product_id.to_string())],
        );

        self.repository.delete(input.product_id)?;

        self.metrics
            .increment("embedding.product_embeddings.deleted", 1);

        self.logger.info(
            "embedding.product.delete.succeeded",
            &[("product_id", input.product_id.to_string())],
        );

        Ok(())
    }
}
