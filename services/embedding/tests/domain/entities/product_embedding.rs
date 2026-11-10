use embedding::domain::entities::ProductEmbedding;
use embedding::domain::value_objects::*;
use uuid::Uuid;

#[test]
fn creates_product_embedding_with_valid_data() {
    let product = ProductEmbedding::new(
        Uuid::new_v4(),
        Uuid::new_v4(),
        EntityName::new("Test Product", "product").unwrap(),
        EntityDescription::new("A test product.", "product").unwrap(),
        Embedding::new(vec![0.1, 0.2, 0.3]).unwrap(),
        EmbeddingModelVersion::new("gte-v1").unwrap(),
    );

    assert_eq!(product.name().value(), "Test Product");
    assert_eq!(product.description().value(), "A test product.");
    assert_eq!(product.embedding().dimension(), 3);
}

#[test]
fn preserves_product_and_store_identifiers() {
    let id = Uuid::new_v4();
    let store_id = Uuid::new_v4();

    let product = ProductEmbedding::new(
        id,
        store_id,
        EntityName::new("Test Product", "product").unwrap(),
        EntityDescription::new("A test product.", "product").unwrap(),
        Embedding::new(vec![0.1]).unwrap(),
        EmbeddingModelVersion::new("gte-v1").unwrap(),
    );

    assert_eq!(product.id(), id);
    assert_eq!(product.store_id(), store_id);
}
