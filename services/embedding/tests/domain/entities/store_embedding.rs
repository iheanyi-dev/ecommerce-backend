use embedding::domain::entities::StoreEmbedding;
use embedding::domain::value_objects::*;
use uuid::Uuid;

#[test]
fn creates_store_embedding_with_valid_data() {
    let store = StoreEmbedding::new(
        Uuid::new_v4(),
        EntityName::new("Test Store", "store").unwrap(),
        EntityDescription::new("A test store.", "store").unwrap(),
        Embedding::new(vec![0.1, 0.2, 0.3]).unwrap(),
        EmbeddingModelVersion::new("gte-v1").unwrap(),
    );

    assert_eq!(store.name().value(), "Test Store");
    assert_eq!(store.description().value(), "A test store.");
    assert_eq!(store.embedding().dimension(), 3);
}

#[test]
fn preserves_store_identifier() {
    let id = Uuid::new_v4();

    let store = StoreEmbedding::new(
        id,
        EntityName::new("Test Store", "store").unwrap(),
        EntityDescription::new("A test store.", "store").unwrap(),
        Embedding::new(vec![0.1]).unwrap(),
        EmbeddingModelVersion::new("gte-v1").unwrap(),
    );

    assert_eq!(store.id(), id);
}
