//! Value object representing an entity name.

use crate::domain::errors::DomainError;

/// A validated name belonging to a domain entity.
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub struct EntityName(String);

impl EntityName {
    /// Creates an entity name associated with the specified entity type.
    pub fn new(
        value: impl Into<String>,
        entity_name: impl Into<String>,
    ) -> Result<Self, DomainError> {
        let value = value.into();
        let entity_name = entity_name.into();

        if value.trim().is_empty() {
            return Err(DomainError::EmptyEntityName { entity_name });
        }

        Ok(Self(value))
    }

    /// Returns the original entity name.
    pub fn value(&self) -> &str {
        &self.0
    }
}

impl std::fmt::Display for EntityName {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.0)
    }
}
