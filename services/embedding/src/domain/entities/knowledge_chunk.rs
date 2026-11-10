//! Knowledge chunk domain entity.

use uuid::Uuid;

use crate::domain::value_objects::{
    ChunkIndex, Embedding, EmbeddingModelVersion, EntityId, KnowledgeChunkContent, StorageReference,
};

/// A persistable, embedded chunk extracted from a knowledge document.
#[derive(Debug, Clone, PartialEq)]
pub struct KnowledgeChunk {
    id: EntityId,
    knowledge_id: EntityId,
    store_id: EntityId,
    storage_reference: StorageReference,
    content: KnowledgeChunkContent,
    chunk_index: ChunkIndex,
    embedding: Embedding,
    embedding_model_version: EmbeddingModelVersion,
}

impl KnowledgeChunk {
    /// Creates a knowledge chunk from validated domain values.
    #[allow(clippy::too_many_arguments)]
    pub fn new(
        id: Uuid,
        knowledge_id: Uuid,
        store_id: Uuid,
        storage_reference: StorageReference,
        content: KnowledgeChunkContent,
        chunk_index: ChunkIndex,
        embedding: Embedding,
        embedding_model_version: EmbeddingModelVersion,
    ) -> Self {
        Self {
            id: EntityId::new(id),
            knowledge_id: EntityId::new(knowledge_id),
            store_id: EntityId::new(store_id),
            storage_reference,
            content,
            chunk_index,
            embedding,
            embedding_model_version,
        }
    }

    /// Returns the chunk identifier.
    pub fn id(&self) -> Uuid {
        self.id.value()
    }

    /// Returns the source knowledge identifier.
    pub fn knowledge_id(&self) -> Uuid {
        self.knowledge_id.value()
    }

    /// Returns the owning store identifier.
    pub fn store_id(&self) -> Uuid {
        self.store_id.value()
    }

    /// Returns the source storage reference.
    pub fn storage_reference(&self) -> &StorageReference {
        &self.storage_reference
    }

    /// Returns the chunk content.
    pub fn content(&self) -> &KnowledgeChunkContent {
        &self.content
    }

    /// Returns the zero-based position of this chunk.
    pub fn chunk_index(&self) -> ChunkIndex {
        self.chunk_index
    }

    /// Returns the embedding vector.
    pub fn embedding(&self) -> &Embedding {
        &self.embedding
    }

    /// Returns the model version used to generate the embedding.
    pub fn embedding_model_version(&self) -> &EmbeddingModelVersion {
        &self.embedding_model_version
    }
}
