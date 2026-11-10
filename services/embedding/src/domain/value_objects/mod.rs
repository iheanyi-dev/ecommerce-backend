//! Immutable domain value objects used by the Embedding service.

mod chunk_index;
mod embedding;
mod embedding_model_version;
mod entity_description;
mod entity_id;
mod entity_name;
mod knowledge_chunk_content;
mod storage_reference;

pub use chunk_index::ChunkIndex;
pub use embedding::Embedding;
pub use embedding_model_version::EmbeddingModelVersion;
pub use entity_description::EntityDescription;
pub use entity_id::EntityId;
pub use entity_name::EntityName;
pub use knowledge_chunk_content::KnowledgeChunkContent;
pub use storage_reference::StorageReference;
