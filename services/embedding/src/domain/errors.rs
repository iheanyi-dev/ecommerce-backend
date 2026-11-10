//! Domain-specific errors raised when business invariants are violated.

use std::error::Error;
use std::fmt::{Display, Formatter};

/// Represents an error originating from the Embedding domain.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum DomainError {
    /// An entity identifier could not be interpreted as a UUID.
    InvalidEntityId { entity_name: String },

    /// An entity name is empty or contains only whitespace.
    EmptyEntityName { entity_name: String },

    /// An entity description is empty or contains only whitespace.
    EmptyEntityDescription { entity_name: String },

    /// An embedding vector contains no values.
    EmptyEmbedding,

    /// An embedding vector contains NaN or an infinite value.
    NonFiniteEmbeddingValue,

    /// An embedding model version is empty or contains only whitespace.
    EmptyEmbeddingModelVersion,

    /// Knowledge chunk content is empty or contains only whitespace.
    EmptyKnowledgeChunkContent,

    /// A storage reference is empty or contains only whitespace.
    EmptyStorageReference,

    /// A chunk index is negative.
    InvalidChunkIndex,
}

impl Display for DomainError {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::InvalidEntityId { entity_name } => {
                write!(f, "{entity_name} id must be a UUID")
            }
            Self::EmptyEntityName { entity_name } => {
                write!(f, "{entity_name} name cannot be empty")
            }
            Self::EmptyEntityDescription { entity_name } => {
                write!(f, "{entity_name} description cannot be empty")
            }
            Self::EmptyEmbedding => write!(f, "embedding vector cannot be empty"),
            Self::NonFiniteEmbeddingValue => {
                write!(f, "embedding vector must contain only finite values")
            }
            Self::EmptyEmbeddingModelVersion => {
                write!(f, "embedding model version cannot be empty")
            }
            Self::EmptyKnowledgeChunkContent => {
                write!(f, "knowledge chunk content cannot be empty")
            }
            Self::EmptyStorageReference => {
                write!(f, "storage reference cannot be empty")
            }
            Self::InvalidChunkIndex => {
                write!(f, "chunk index must be a non-negative integer")
            }
        }
    }
}

impl Error for DomainError {}
