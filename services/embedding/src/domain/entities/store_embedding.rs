//! Store embedding domain entity.

use uuid::Uuid;

use crate::domain::value_objects::{
    Embedding, EmbeddingModelVersion, EntityDescription, EntityId, EntityName,
};

/// A persistable vector representation of a store.
#[derive(Debug, Clone, PartialEq)]
pub struct StoreEmbedding {
    id: EntityId,
    name: EntityName,
    description: EntityDescription,
    embedding: Embedding,
    embedding_model_version: EmbeddingModelVersion,
}

impl StoreEmbedding {
    /// Creates a store embedding from validated domain values.
    pub fn new(
        id: Uuid,
        name: EntityName,
        description: EntityDescription,
        embedding: Embedding,
        embedding_model_version: EmbeddingModelVersion,
    ) -> Self {
        Self {
            id: EntityId::new(id),
            name,
            description,
            embedding,
            embedding_model_version,
        }
    }

    /// Returns the store identifier.
    pub fn id(&self) -> Uuid {
        self.id.value()
    }

    /// Returns the store name.
    pub fn name(&self) -> &EntityName {
        &self.name
    }

    /// Returns the store description.
    pub fn description(&self) -> &EntityDescription {
        &self.description
    }

    /// Returns the store embedding vector.
    pub fn embedding(&self) -> &Embedding {
        &self.embedding
    }

    /// Returns the model version used to generate the embedding.
    pub fn embedding_model_version(&self) -> &EmbeddingModelVersion {
        &self.embedding_model_version
    }
}
