//! Value object identifying the model version that generated an embedding.

use crate::domain::errors::DomainError;

/// Identifies a particular embedding model version.
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub struct EmbeddingModelVersion(String);

impl EmbeddingModelVersion {
    /// Creates a model version after validating that it is not blank.
    pub fn new(value: impl Into<String>) -> Result<Self, DomainError> {
        let value = value.into();

        if value.trim().is_empty() {
            return Err(DomainError::EmptyEmbeddingModelVersion);
        }

        Ok(Self(value))
    }

    /// Returns the original model version string.
    pub fn value(&self) -> &str {
        &self.0
    }
}

impl std::fmt::Display for EmbeddingModelVersion {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.0)
    }
}
