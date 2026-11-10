//! Application result carrying a Store embedding and its similarity score.

use crate::domain::entities::StoreEmbedding;

/// Represents a Store embedding and its query-specific similarity score.
#[derive(Debug, Clone, PartialEq)]
pub struct ScoredStoreEmbedding {
    pub embedding: StoreEmbedding,
    pub score: f32,
}
