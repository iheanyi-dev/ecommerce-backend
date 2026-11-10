//! Value object representing a knowledge chunk's position.

use crate::domain::errors::DomainError;

/// Represents a zero-based position within a source knowledge document.
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct ChunkIndex(usize);

impl ChunkIndex {
    /// Creates a chunk index, rejecting negative values.
    pub fn new(value: i64) -> Result<Self, DomainError> {
        let value = usize::try_from(value).map_err(|_| DomainError::InvalidChunkIndex)?;

        Ok(Self(value))
    }

    /// Returns the zero-based chunk position.
    pub fn value(&self) -> usize {
        self.0
    }
}
