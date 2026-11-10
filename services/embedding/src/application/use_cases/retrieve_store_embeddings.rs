//! Use case for semantically retrieving Store embeddings.

use std::sync::Arc;

use crate::application::{
    dto::{RetrieveStoreEmbeddingsInput, RetrieveStoreEmbeddingsOutput},
    errors::{ApplicationError, ApplicationResult},
    ports::{EmbeddingModel, Logger, Metrics, StoreEmbeddingRepository},
};

/// Embeds a query and retrieves matching Store embeddings.
pub struct RetrieveStoreEmbeddingsUseCase {
    embedding_model: Arc<dyn EmbeddingModel>,
    repository: Arc<dyn StoreEmbeddingRepository>,
    logger: Arc<dyn Logger>,
    metrics: Arc<dyn Metrics>,
}

impl RetrieveStoreEmbeddingsUseCase {
    /// Creates the use case with its required dependencies.
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

    /// Validates the request and retrieves matching Store embeddings.
    pub fn execute(
        &self,
        input: RetrieveStoreEmbeddingsInput,
    ) -> ApplicationResult<RetrieveStoreEmbeddingsOutput> {
        let query = input.query.trim();

        if query.is_empty() {
            return Err(ApplicationError::Validation(
                "query must not be empty".into(),
            ));
        }

        if input.top_k == 0 {
            return Err(ApplicationError::Validation(
                "top_k must be a positive integer".into(),
            ));
        }

        self.logger.info(
            "embedding.store_embedding.retrieve_started",
            &[
                ("top_k", input.top_k.to_string()),
                ("offset", input.offset.to_string()),
            ],
        );

        let (query_embedding, _) = self.embedding_model.generate(query)?;

        let results = self
            .repository
            .search(&query_embedding, input.top_k, input.offset)?;

        self.metrics
            .increment("embedding.store_embeddings.retrieved", 1);
        self.metrics.observe(
            "embedding.store_embeddings.retrieval_results",
            results.len() as f64,
        );

        self.logger.info(
            "embedding.store_embedding.retrieved",
            &[
                ("result_count", results.len().to_string()),
                ("top_k", input.top_k.to_string()),
                ("offset", input.offset.to_string()),
            ],
        );

        Ok(RetrieveStoreEmbeddingsOutput { results })
    }
}
