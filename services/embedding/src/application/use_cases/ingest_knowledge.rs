//! Use case for ingesting and embedding a knowledge document.

use std::sync::Arc;

use uuid::Uuid;

use crate::application::{
    dto::{IngestKnowledgeInput, IngestKnowledgeOutput},
    errors::{ApplicationError, ApplicationResult},
    ports::{
        DocumentContentReader, EmbeddingModel, KnowledgeChunkRepository, Logger, Metrics,
        TextChunker,
    },
};
use crate::domain::{
    entities::KnowledgeChunk,
    value_objects::{ChunkIndex, KnowledgeChunkContent, StorageReference},
};

/// Reads, chunks, embeds, and persists a knowledge document.
pub struct IngestKnowledgeUseCase {
    document_reader: Arc<dyn DocumentContentReader>,
    embedding_model: Arc<dyn EmbeddingModel>,
    chunker: Arc<dyn TextChunker>,
    repository: Arc<dyn KnowledgeChunkRepository>,
    logger: Arc<dyn Logger>,
    metrics: Arc<dyn Metrics>,
}

impl IngestKnowledgeUseCase {
    /// Creates the use case with its required application dependencies.
    pub fn new(
        document_reader: Arc<dyn DocumentContentReader>,
        embedding_model: Arc<dyn EmbeddingModel>,
        chunker: Arc<dyn TextChunker>,
        repository: Arc<dyn KnowledgeChunkRepository>,
        logger: Arc<dyn Logger>,
        metrics: Arc<dyn Metrics>,
    ) -> Self {
        Self {
            document_reader,
            embedding_model,
            chunker,
            repository,
            logger,
            metrics,
        }
    }

    /// Ingests a document and replaces its previously stored chunks.
    pub fn execute(&self, input: IngestKnowledgeInput) -> ApplicationResult<IngestKnowledgeOutput> {
        let storage_reference = StorageReference::new(input.storage_reference.trim())?;

        self.logger.info(
            "embedding.knowledge.ingest.started",
            &[
                ("knowledge_id", input.knowledge_id.to_string()),
                ("store_id", input.store_id.to_string()),
            ],
        );

        // Read the source document before attempting to modify stored chunks.
        let document = self.document_reader.read(&storage_reference)?;

        if document.trim().is_empty() {
            return Err(ApplicationError::Validation(
                "knowledge document content cannot be empty".into(),
            ));
        }

        let chunks = self.chunker.chunk(&document)?;

        if chunks.is_empty() {
            return Err(ApplicationError::Validation(
                "knowledge document produced no chunks".into(),
            ));
        }

        let mut knowledge_chunks = Vec::with_capacity(chunks.len());
        let mut model_versions = Vec::with_capacity(chunks.len());

        // Generate and validate every chunk before replacing existing data.
        for (index, chunk) in chunks.into_iter().enumerate() {
            let content = chunk.trim().to_owned();

            if content.is_empty() {
                return Err(ApplicationError::Validation(format!(
                    "knowledge chunk at index {index} is empty"
                )));
            }

            let (embedding, model_version) = self.embedding_model.generate(&content)?;

            let index = i64::try_from(index).map_err(|_| {
                ApplicationError::Validation("knowledge document contains too many chunks".into())
            })?;

            let knowledge_chunk = KnowledgeChunk::new(
                Uuid::new_v4(),
                input.knowledge_id,
                input.store_id,
                storage_reference.clone(),
                KnowledgeChunkContent::new(content)?,
                ChunkIndex::new(index)?,
                embedding,
                model_version.clone(),
            );

            knowledge_chunks.push(knowledge_chunk);
            model_versions.push(model_version);
        }

        // Replace only after all chunks have been generated successfully.
        let chunk_count = knowledge_chunks.len();

        self.repository
            .replace_for_knowledge(input.knowledge_id, knowledge_chunks)?;

        self.metrics
            .increment("embedding.knowledge.documents.ingested", 1);
        self.metrics
            .observe("embedding.knowledge.chunks.ingested", chunk_count as f64);

        self.logger.info(
            "embedding.knowledge.ingest.succeeded",
            &[
                ("knowledge_id", input.knowledge_id.to_string()),
                ("store_id", input.store_id.to_string()),
                ("chunk_count", chunk_count.to_string()),
            ],
        );

        Ok(IngestKnowledgeOutput {
            knowledge_id: input.knowledge_id,
            store_id: input.store_id,
            chunk_count,
            embedding_model_versions: model_versions,
        })
    }
}
