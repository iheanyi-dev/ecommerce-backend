//! DTOs for ingesting a knowledge document.

use uuid::Uuid;

use crate::domain::value_objects::EmbeddingModelVersion;

/// Identifies a knowledge document and its stored source.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct IngestKnowledgeInput {
    pub knowledge_id: Uuid,
    pub store_id: Uuid,
    pub storage_reference: String,
}

/// Summarizes a successfully ingested knowledge document.
#[derive(Debug, Clone, PartialEq)]
pub struct IngestKnowledgeOutput {
    pub knowledge_id: Uuid,
    pub store_id: Uuid,
    pub chunk_count: usize,
    pub embedding_model_versions: Vec<EmbeddingModelVersion>,
}
