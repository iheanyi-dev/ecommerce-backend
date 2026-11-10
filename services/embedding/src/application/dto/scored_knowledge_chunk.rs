//! Application result carrying a chunk and its query-specific similarity score.

use crate::domain::entities::KnowledgeChunk;

/// Represents a retrieved knowledge chunk and its similarity score.
#[derive(Debug, Clone, PartialEq)]
pub struct ScoredKnowledgeChunk {
    pub chunk: KnowledgeChunk,
    pub score: f32,
}
