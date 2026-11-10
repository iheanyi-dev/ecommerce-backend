//! Value object representing an entity description.

use crate::domain::errors::DomainError;

/// A validated description belonging to a domain entity.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct EntityDescription {
    value: String,
}

impl EntityDescription {
    /// Creates a description associated with the specified entity type.
    pub fn new(
        value: impl Into<String>,
        entity_name: impl Into<String>,
    ) -> Result<Self, DomainError> {
        let value = value.into();
        let entity_name = entity_name.into();

        if value.trim().is_empty() {
            return Err(DomainError::EmptyEntityDescription { entity_name });
        }

        Ok(Self { value })
    }

    /// Returns the original description.
    pub fn value(&self) -> &str {
        &self.value
    }
}

impl std::fmt::Display for EntityDescription {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.value)
    }
}
