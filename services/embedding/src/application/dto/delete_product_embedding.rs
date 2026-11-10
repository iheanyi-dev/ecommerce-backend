//! DTO for deleting a Product embedding.

use uuid::Uuid;

/// Identifies the Product whose embedding must be deleted.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DeleteProductEmbeddingInput {
    pub product_id: Uuid,
}
