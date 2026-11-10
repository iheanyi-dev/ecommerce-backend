//! Domain entities representing embedded products, stores, and knowledge chunks.

mod knowledge_chunk;
mod product_embedding;
mod store_embedding;

pub use knowledge_chunk::KnowledgeChunk;
pub use product_embedding::ProductEmbedding;
pub use store_embedding::StoreEmbedding;
