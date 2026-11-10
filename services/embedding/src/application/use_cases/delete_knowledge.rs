//! Use case for deleting all chunks belonging to a knowledge document.

use std::sync::Arc;

use crate::application::{
    dto::DeleteKnowledgeInput,
    errors::ApplicationResult,
    ports::{KnowledgeChunkRepository, Logger, Metrics},
};

/// Deletes a knowledge document's stored chunks.
pub struct DeleteKnowledgeUseCase {
    repository: Arc<dyn KnowledgeChunkRepository>,
    logger: Arc<dyn Logger>,
    metrics: Arc<dyn Metrics>,
}

impl DeleteKnowledgeUseCase {
    /// Creates the use case with its required dependencies.
    pub fn new(
        repository: Arc<dyn KnowledgeChunkRepository>,
        logger: Arc<dyn Logger>,
        metrics: Arc<dyn Metrics>,
    ) -> Self {
        Self {
            repository,
            logger,
            metrics,
        }
    }

    /// Deletes all chunks associated with the supplied knowledge ID.
    pub fn execute(&self, input: DeleteKnowledgeInput) -> ApplicationResult<()> {
        self.logger.info(
            "embedding.knowledge.delete.started",
            &[("knowledge_id", input.knowledge_id.to_string())],
        );

        self.repository.delete_by_knowledge_id(input.knowledge_id)?;

        self.metrics.increment("embedding.knowledge.deleted", 1);

        self.logger.info(
            "embedding.knowledge.delete.succeeded",
            &[("knowledge_id", input.knowledge_id.to_string())],
        );

        Ok(())
    }
}
