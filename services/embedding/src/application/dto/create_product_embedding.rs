//! DTOs for creating or replacing a Product embedding.

use uuid::Uuid;

use crate::domain::value_objects::{Embedding, EmbeddingModelVersion};

/// Product embedding request received from an external boundary.
#[derive(Debug, Clone, PartialEq)]
pub struct CreateProductEmbeddingInput {
    pub product_id: Uuid,
    pub store_id: Uuid,
    pub name: String,
    pub description: String,
}

/// Result returned after persisting a Product embedding.
#[derive(Debug, Clone, PartialEq)]
pub struct CreateProductEmbeddingOutput {
    pub product_id: Uuid,
    pub embedding: Embedding,
    pub embedding_model_version: EmbeddingModelVersion,
}
