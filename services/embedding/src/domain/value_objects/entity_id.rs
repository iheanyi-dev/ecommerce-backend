//! Value object representing a domain entity identifier.

use std::str::FromStr;
use uuid::Uuid;

use crate::domain::errors::DomainError;

/// A validated UUID identifying a domain entity.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub struct EntityId(Uuid);

impl EntityId {
    /// Creates an entity identifier from an existing UUID.
    pub fn new(value: Uuid) -> Self {
        Self(value)
    }

    /// Parses a UUID string and associates parsing failures with an entity type.
    pub fn parse(value: &str, entity_name: impl Into<String>) -> Result<Self, DomainError> {
        Uuid::from_str(value)
            .map(Self)
            .map_err(|_| DomainError::InvalidEntityId {
                entity_name: entity_name.into(),
            })
    }

    /// Returns the underlying UUID.
    pub fn value(&self) -> Uuid {
        self.0
    }
}

impl std::fmt::Display for EntityId {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.0)
    }
}
