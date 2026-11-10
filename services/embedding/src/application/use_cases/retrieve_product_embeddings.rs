//! Use case for semantically retrieving Product embeddings.

use std::sync::Arc;

use crate::application::{
    dto::{RetrieveProductEmbeddingsInput, RetrieveProductEmbeddingsOutput},
    errors::{ApplicationError, ApplicationResult},
    ports::{EmbeddingModel, Logger, Metrics, ProductEmbeddingRepository},
};

/// Embeds a query and retrieves matching Product embeddings.
pub struct RetrieveProductEmbeddingsUseCase {
    embedding_model: Arc<dyn EmbeddingModel>,
    repository: Arc<dyn ProductEmbeddingRepository>,
    logger: Arc<dyn Logger>,
    metrics: Arc<dyn Metrics>,
}

impl RetrieveProductEmbeddingsUseCase {
    /// Creates the use case with its required dependencies.
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

    /// Validates the request and retrieves matching Product embeddings.
    pub fn execute(
        &self,
        input: RetrieveProductEmbeddingsInput,
    ) -> ApplicationResult<RetrieveProductEmbeddingsOutput> {
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
            "embedding.product_embedding.retrieve_started",
            &[
                ("top_k", input.top_k.to_string()),
                ("offset", input.offset.to_string()),
                (
                    "store_id",
                    input
                        .store_id
                        .map(|id| id.to_string())
                        .unwrap_or_else(|| "none".into()),
                ),
            ],
        );

        let (query_embedding, _) = self.embedding_model.generate(query)?;

        let results =
            self.repository
                .search(&query_embedding, input.top_k, input.offset, input.store_id)?;

        self.metrics
            .increment("embedding.product_embeddings.retrieved", 1);
        self.metrics.observe(
            "embedding.product_embeddings.retrieval_results",
            results.len() as f64,
        );

        self.logger.info(
            "embedding.product_embedding.retrieved",
            &[
                ("result_count", results.len().to_string()),
                ("top_k", input.top_k.to_string()),
                ("offset", input.offset.to_string()),
            ],
        );

        Ok(RetrieveProductEmbeddingsOutput { results })
    }
}
