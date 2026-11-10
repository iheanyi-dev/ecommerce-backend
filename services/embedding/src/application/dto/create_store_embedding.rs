//! DTOs for creating or replacing a Store embedding.

use uuid::Uuid;

use crate::domain::value_objects::{Embedding, EmbeddingModelVersion};

/// Store embedding request received from an external boundary.
#[derive(Debug, Clone, PartialEq)]
pub struct CreateStoreEmbeddingInput {
    pub store_id: Uuid,
    pub name: String,
    pub description: String,
}

/// Result returned after persisting a Store embedding.
#[derive(Debug, Clone, PartialEq)]
pub struct CreateStoreEmbeddingOutput {
    pub store_id: Uuid,
    pub embedding: Embedding,
    pub embedding_model_version: EmbeddingModelVersion,
}
