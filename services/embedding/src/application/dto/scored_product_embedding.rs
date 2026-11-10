//! Application result carrying a Product embedding and its similarity score.

use crate::domain::entities::ProductEmbedding;

/// Represents a Product embedding and its query-specific similarity score.
#[derive(Debug, Clone, PartialEq)]
pub struct ScoredProductEmbedding {
    pub embedding: ProductEmbedding,
    pub score: f32,
}
