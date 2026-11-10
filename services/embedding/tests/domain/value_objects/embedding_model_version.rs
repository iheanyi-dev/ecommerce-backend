use embedding::domain::{errors::DomainError, value_objects::EmbeddingModelVersion};

#[test]
fn creates_model_version() {
    let version = EmbeddingModelVersion::new("text-embedding-v1").unwrap();

    assert_eq!(version.value(), "text-embedding-v1");
}

#[test]
fn rejects_empty_model_version() {
    assert_eq!(
        EmbeddingModelVersion::new(""),
        Err(DomainError::EmptyEmbeddingModelVersion)
    );
}

#[test]
fn rejects_whitespace_model_version() {
    assert_eq!(
        EmbeddingModelVersion::new("   "),
        Err(DomainError::EmptyEmbeddingModelVersion)
    );
}

#[test]
fn displays_model_version() {
    let version = EmbeddingModelVersion::new("model-v1").unwrap();

    assert_eq!(version.to_string(), "model-v1");
}
