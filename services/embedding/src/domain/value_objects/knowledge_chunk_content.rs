//! Value object representing knowledge chunk content.

use crate::domain::errors::DomainError;

/// Validated text belonging to a knowledge chunk.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct KnowledgeChunkContent(String);

impl KnowledgeChunkContent {
    /// Creates knowledge chunk content, rejecting empty or whitespace-only text.
    pub fn new(value: impl Into<String>) -> Result<Self, DomainError> {
        let value = value.into();

        if value.trim().is_empty() {
            return Err(DomainError::EmptyKnowledgeChunkContent);
        }

        Ok(Self(value))
    }

    /// Returns the original knowledge chunk content.
    pub fn value(&self) -> &str {
        &self.0
    }
}

impl std::fmt::Display for KnowledgeChunkContent {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.0)
    }
}
