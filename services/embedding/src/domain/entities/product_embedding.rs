//! Product embedding domain entity.

use uuid::Uuid;

use crate::domain::value_objects::{
    Embedding, EmbeddingModelVersion, EntityDescription, EntityId, EntityName,
};

/// A persistable vector representation of a product.
#[derive(Debug, Clone, PartialEq)]
pub struct ProductEmbedding {
    id: EntityId,
    store_id: EntityId,
    name: EntityName,
    description: EntityDescription,
    embedding: Embedding,
    embedding_model_version: EmbeddingModelVersion,
}

impl ProductEmbedding {
    /// Creates a product embedding from validated domain values.
    pub fn new(
        id: Uuid,
        store_id: Uuid,
        name: EntityName,
        description: EntityDescription,
        embedding: Embedding,
        embedding_model_version: EmbeddingModelVersion,
    ) -> Self {
        Self {
            id: EntityId::new(id),
            store_id: EntityId::new(store_id),
            name,
            description,
            embedding,
            embedding_model_version,
        }
    }

    /// Returns the product identifier.
    pub fn id(&self) -> Uuid {
        self.id.value()
    }

    /// Returns the owning store identifier.
    pub fn store_id(&self) -> Uuid {
        self.store_id.value()
    }

    /// Returns the product name.
    pub fn name(&self) -> &EntityName {
        &self.name
    }

    /// Returns the product description.
    pub fn description(&self) -> &EntityDescription {
        &self.description
    }

    /// Returns the product embedding vector.
    pub fn embedding(&self) -> &Embedding {
        &self.embedding
    }

    /// Returns the model version used to generate the embedding.
    pub fn embedding_model_version(&self) -> &EmbeddingModelVersion {
        &self.embedding_model_version
    }
}
