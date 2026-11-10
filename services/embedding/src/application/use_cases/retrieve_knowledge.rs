//! Use case for tenant-scoped semantic knowledge retrieval.

use std::sync::Arc;

use crate::application::{
    dto::{RetrieveKnowledgeInput, RetrieveKnowledgeOutput},
    errors::{ApplicationError, ApplicationResult},
    ports::{EmbeddingModel, KnowledgeChunkRepository, Logger, Metrics},
};

/// Embeds a query and retrieves matching knowledge chunks for one store.
pub struct RetrieveKnowledgeUseCase {
    repository: Arc<dyn KnowledgeChunkRepository>,
    embedding_model: Arc<dyn EmbeddingModel>,
    logger: Arc<dyn Logger>,
    metrics: Arc<dyn Metrics>,
}

impl RetrieveKnowledgeUseCase {
    /// Creates the use case with its required dependencies.
    pub fn new(
        repository: Arc<dyn KnowledgeChunkRepository>,
        embedding_model: Arc<dyn EmbeddingModel>,
        logger: Arc<dyn Logger>,
        metrics: Arc<dyn Metrics>,
    ) -> Self {
        Self {
            repository,
            embedding_model,
            logger,
            metrics,
        }
    }

    /// Validates, embeds, and retrieves knowledge within the requested store.
    pub fn execute(
        &self,
        input: RetrieveKnowledgeInput,
    ) -> ApplicationResult<RetrieveKnowledgeOutput> {
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
            "knowledge.retrieval.started",
            &[
                ("store_id", input.store_id.to_string()),
                ("top_k", input.top_k.to_string()),
                ("offset", input.offset.to_string()),
            ],
        );

        let (query_embedding, _) = self.embedding_model.generate(query)?;

        // The repository must enforce the store ID as a tenant filter.
        let chunks =
            self.repository
                .search(&query_embedding, input.top_k, input.offset, input.store_id)?;

        self.metrics.increment("knowledge.retrieval.succeeded", 1);
        self.metrics
            .observe("knowledge.retrieval.result_count", chunks.len() as f64);

        self.logger.info(
            "knowledge.retrieval.succeeded",
            &[
                ("store_id", input.store_id.to_string()),
                ("result_count", chunks.len().to_string()),
            ],
        );

        Ok(RetrieveKnowledgeOutput {
            store_id: input.store_id,
            chunks,
        })
    }
}
