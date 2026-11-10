use embedding::domain::{errors::DomainError, value_objects::StorageReference};

#[test]
fn creates_storage_reference() {
    let reference = StorageReference::new("storage/object/123").unwrap();

    assert_eq!(reference.value(), "storage/object/123");
}

#[test]
fn rejects_empty_storage_reference() {
    assert_eq!(
        StorageReference::new(""),
        Err(DomainError::EmptyStorageReference)
    );
}

#[test]
fn rejects_whitespace_storage_reference() {
    assert_eq!(
        StorageReference::new("   "),
        Err(DomainError::EmptyStorageReference)
    );
}

#[test]
fn displays_storage_reference() {
    let reference = StorageReference::new("storage/object/123").unwrap();

    assert_eq!(reference.to_string(), "storage/object/123");
}
