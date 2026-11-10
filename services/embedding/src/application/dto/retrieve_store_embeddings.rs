//! DTOs for semantic Store embedding retrieval.

use crate::application::dto::ScoredStoreEmbedding;

/// Describes a semantic Store embedding search request.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct RetrieveStoreEmbeddingsInput {
    pub query: String,
    pub top_k: usize,
    pub offset: usize,
}

impl RetrieveStoreEmbeddingsInput {
    /// Creates a Store retrieval request with the Python defaults.
    pub fn new(query: impl Into<String>) -> Self {
        Self {
            query: query.into(),
            top_k: 10,
            offset: 0,
        }
    }
}

/// Contains the matching Store embeddings.
#[derive(Debug, Clone, PartialEq)]
pub struct RetrieveStoreEmbeddingsOutput {
    pub results: Vec<ScoredStoreEmbedding>,
}
