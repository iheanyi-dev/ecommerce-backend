//! Application-layer ports.
//!
//! These traits define the external capabilities required by the
//! Embedding application use cases.

use std::error::Error;
use uuid::Uuid;

use crate::application::dto::{ScoredKnowledgeChunk, ScoredProductEmbedding, ScoredStoreEmbedding};
use crate::domain::{
    entities::{KnowledgeChunk, ProductEmbedding, StoreEmbedding},
    value_objects::{Embedding, EmbeddingModelVersion, StorageReference},
};

/// Error returned by infrastructure adapters.
pub type PortError = Box<dyn Error + Send + Sync>;

/// Result returned by application ports.
pub type PortResult<T> = Result<T, PortError>;

/// Reads document content from an external storage system.
pub trait DocumentContentReader: Send + Sync {
    /// Reads the content referenced by the supplied storage reference.
    fn read(&self, reference: &StorageReference) -> PortResult<String>;
}

/// Generates vector embeddings for supplied text.
pub trait EmbeddingModel: Send + Sync {
    /// Generates an embedding and returns the model version used.
    fn generate(&self, text: &str) -> PortResult<(Embedding, EmbeddingModelVersion)>;
}

/// Splits document text into smaller chunks.
pub trait TextChunker: Send + Sync {
    /// Splits the supplied text into chunks.
    fn chunk(&self, text: &str) -> PortResult<Vec<String>>;
}

/// Persists and searches knowledge chunks.
pub trait KnowledgeChunkRepository: Send + Sync {
    /// Replaces all chunks belonging to a knowledge document.
    fn replace_for_knowledge(
        &self,
        knowledge_id: Uuid,
        chunks: Vec<KnowledgeChunk>,
    ) -> PortResult<()>;

    /// Deletes all chunks belonging to a knowledge document.
    fn delete_by_knowledge_id(&self, knowledge_id: Uuid) -> PortResult<()>;

    /// Searches knowledge chunks using vector similarity.
    fn search(
        &self,
        query_embedding: &Embedding,
        top_k: usize,
        offset: usize,
        store_id: Uuid,
    ) -> PortResult<Vec<ScoredKnowledgeChunk>>;
}

/// Persists and searches product embeddings.
pub trait ProductEmbeddingRepository: Send + Sync {
    /// Creates or updates a product embedding.
    fn upsert(&self, embedding: ProductEmbedding) -> PortResult<()>;

    /// Deletes the embedding belonging to a product.
    fn delete(&self, product_id: Uuid) -> PortResult<()>;

    /// Deletes all product embeddings belonging to a store.
    fn delete_by_store_id(&self, store_id: Uuid) -> PortResult<()>;

    /// Searches product embeddings, optionally scoped to a store.
    fn search(
        &self,
        query_embedding: &Embedding,
        top_k: usize,
        offset: usize,
        store_id: Option<Uuid>,
    ) -> PortResult<Vec<ScoredProductEmbedding>>;
}

/// Persists and searches store embeddings.
pub trait StoreEmbeddingRepository: Send + Sync {
    /// Creates or updates a store embedding.
    fn upsert(&self, embedding: StoreEmbedding) -> PortResult<()>;

    /// Deletes the embedding belonging to a store.
    fn delete(&self, store_id: Uuid) -> PortResult<()>;

    /// Searches store embeddings.
    fn search(
        &self,
        query_embedding: &Embedding,
        top_k: usize,
        offset: usize,
    ) -> PortResult<Vec<ScoredStoreEmbedding>>;
}

/// Application logging port.
pub trait Logger: Send + Sync {
    /// Records an informational event.
    fn info(&self, event: &str, fields: &[(&str, String)]);

    /// Records a warning event.
    fn warning(&self, event: &str, fields: &[(&str, String)]);

    /// Records an error event.
    fn error(&self, event: &str, fields: &[(&str, String)]);
}

/// Application metrics port.
pub trait Metrics: Send + Sync {
    /// Increments a counter by the supplied amount.
    fn increment(&self, name: &str, value: u64);

    /// Records an observed numeric value.
    fn observe(&self, name: &str, value: f64);
}
