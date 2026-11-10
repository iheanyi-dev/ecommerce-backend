//! Input contract for deleting indexed knowledge chunks.

use uuid::Uuid;

/// Identifies the knowledge document whose indexed chunks must be removed.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DeleteKnowledgeInput {
    pub knowledge_id: Uuid,
}
