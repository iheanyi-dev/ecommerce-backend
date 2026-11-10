use embedding::domain::entities::KnowledgeChunk;
use embedding::domain::value_objects::*;
use uuid::Uuid;

#[test]
fn creates_knowledge_chunk_with_valid_data() {
    let chunk = KnowledgeChunk::new(
        Uuid::new_v4(),
        Uuid::new_v4(),
        Uuid::new_v4(),
        StorageReference::new("knowledge/store-id/guide.pdf").unwrap(),
        KnowledgeChunkContent::new("A useful knowledge chunk.").unwrap(),
        ChunkIndex::new(0).unwrap(),
        Embedding::new(vec![0.1, 0.2, 0.3]).unwrap(),
        EmbeddingModelVersion::new("gte-v1").unwrap(),
    );

    assert_eq!(chunk.content().value(), "A useful knowledge chunk.");
    assert_eq!(chunk.chunk_index().value(), 0);
    assert_eq!(chunk.embedding().dimension(), 3);
}

#[test]
fn preserves_entity_identifiers() {
    let id = Uuid::new_v4();
    let knowledge_id = Uuid::new_v4();
    let store_id = Uuid::new_v4();

    let chunk = KnowledgeChunk::new(
        id,
        knowledge_id,
        store_id,
        StorageReference::new("knowledge/guide.pdf").unwrap(),
        KnowledgeChunkContent::new("Useful content.").unwrap(),
        ChunkIndex::new(0).unwrap(),
        Embedding::new(vec![0.1]).unwrap(),
        EmbeddingModelVersion::new("gte-v1").unwrap(),
    );

    assert_eq!(chunk.id(), id);
    assert_eq!(chunk.knowledge_id(), knowledge_id);
    assert_eq!(chunk.store_id(), store_id);
}
