//! DTOs for semantic Product embedding retrieval.

use uuid::Uuid;

use crate::application::dto::ScoredProductEmbedding;

/// Describes a semantic Product embedding search request.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct RetrieveProductEmbeddingsInput {
    pub query: String,
    pub top_k: usize,
    pub offset: usize,
    pub store_id: Option<Uuid>,
}

impl RetrieveProductEmbeddingsInput {
    /// Creates a Product retrieval request with the Python defaults.
    pub fn new(query: impl Into<String>) -> Self {
        Self {
            query: query.into(),
            top_k: 10,
            offset: 0,
            store_id: None,
        }
    }
}

/// Contains the matching Product embeddings.
#[derive(Debug, Clone, PartialEq)]
pub struct RetrieveProductEmbeddingsOutput {
    pub results: Vec<ScoredProductEmbedding>,
}
