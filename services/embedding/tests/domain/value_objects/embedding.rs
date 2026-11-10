use embedding::domain::{errors::DomainError, value_objects::Embedding};

#[test]
fn creates_embedding_with_finite_values() {
    let embedding = Embedding::new(vec![0.1, 0.2, 0.3]).unwrap();

    assert_eq!(embedding.vector(), &[0.1, 0.2, 0.3]);
    assert_eq!(embedding.dimension(), 3);
}

#[test]
fn rejects_empty_embedding() {
    assert_eq!(Embedding::new(vec![]), Err(DomainError::EmptyEmbedding));
}

#[test]
fn rejects_nan_values() {
    assert_eq!(
        Embedding::new(vec![0.1, f32::NAN]),
        Err(DomainError::NonFiniteEmbeddingValue)
    );
}

#[test]
fn rejects_infinite_values() {
    assert_eq!(
        Embedding::new(vec![f32::INFINITY]),
        Err(DomainError::NonFiniteEmbeddingValue)
    );
}

#[test]
fn rejects_negative_infinity() {
    assert_eq!(
        Embedding::new(vec![f32::NEG_INFINITY]),
        Err(DomainError::NonFiniteEmbeddingValue)
    );
}
