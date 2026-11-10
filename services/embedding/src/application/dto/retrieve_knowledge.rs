//! Request and response contracts for knowledge retrieval.

use uuid::Uuid;

use super::ScoredKnowledgeChunk;

/// Describes a tenant-scoped semantic knowledge search request.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct RetrieveKnowledgeInput {
    pub store_id: Uuid,
    pub query: String,
    pub top_k: usize,
    pub offset: usize,
}

impl RetrieveKnowledgeInput {
    /// Creates a knowledge retrieval request with the Python defaults.
    pub fn new(store_id: Uuid, query: impl Into<String>) -> Self {
        Self {
            store_id,
            query: query.into(),
            top_k: 10,
            offset: 0,
        }
    }
}

/// Returns matching knowledge chunks without exposing repository internals.
#[derive(Debug, Clone, PartialEq)]
pub struct RetrieveKnowledgeOutput {
    pub store_id: Uuid,
    pub chunks: Vec<ScoredKnowledgeChunk>,
}
