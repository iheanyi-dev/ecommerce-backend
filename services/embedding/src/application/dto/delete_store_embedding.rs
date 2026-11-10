//! DTO for deleting a Store embedding.

use uuid::Uuid;

/// Identifies the Store whose embedding and associated Product embeddings must be deleted.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DeleteStoreEmbeddingInput {
    pub store_id: Uuid,
}
