//! Value object representing a stored knowledge object's reference.

use crate::domain::errors::DomainError;

/// Identifies the source object in the Storage service.
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub struct StorageReference(String);

impl StorageReference {
    /// Creates a storage reference, rejecting empty or whitespace-only values.
    pub fn new(value: impl Into<String>) -> Result<Self, DomainError> {
        let value = value.into();

        if value.trim().is_empty() {
            return Err(DomainError::EmptyStorageReference);
        }

        Ok(Self(value))
    }

    /// Returns the original storage reference.
    pub fn value(&self) -> &str {
        &self.0
    }
}

impl std::fmt::Display for StorageReference {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.0)
    }
}
