//! Use case for deleting a store embedding and its product embeddings.

use std::sync::Arc;

use crate::application::{
    dto::DeleteStoreEmbeddingInput,
    errors::ApplicationResult,
    ports::{Logger, Metrics, ProductEmbeddingRepository, StoreEmbeddingRepository},
};

/// Deletes all product embeddings and the store embedding belonging to a store.
pub struct DeleteStoreEmbeddingUseCase {
    product_repository: Arc<dyn ProductEmbeddingRepository>,
    store_repository: Arc<dyn StoreEmbeddingRepository>,
    logger: Arc<dyn Logger>,
    metrics: Arc<dyn Metrics>,
}

impl DeleteStoreEmbeddingUseCase {
    /// Creates the use case with its required dependencies.
    pub fn new(
        product_repository: Arc<dyn ProductEmbeddingRepository>,
        store_repository: Arc<dyn StoreEmbeddingRepository>,
        logger: Arc<dyn Logger>,
        metrics: Arc<dyn Metrics>,
    ) -> Self {
        Self {
            product_repository,
            store_repository,
            logger,
            metrics,
        }
    }

    /// Deletes product embeddings first, followed by the store embedding.
    pub fn execute(&self, input: DeleteStoreEmbeddingInput) -> ApplicationResult<()> {
        self.logger.info(
            "embedding.store.delete.started",
            &[("store_id", input.store_id.to_string())],
        );

        // Do not delete the store embedding if deleting its products fails.
        self.product_repository.delete_by_store_id(input.store_id)?;
        self.store_repository.delete(input.store_id)?;

        self.metrics
            .increment("embedding.store_embeddings.deleted", 1);

        self.logger.info(
            "embedding.store.delete.succeeded",
            &[("store_id", input.store_id.to_string())],
        );

        Ok(())
    }
}
