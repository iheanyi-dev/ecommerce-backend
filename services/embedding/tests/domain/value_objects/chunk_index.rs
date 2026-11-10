use embedding::domain::{errors::DomainError, value_objects::ChunkIndex};

#[test]
fn creates_chunk_index_from_non_negative_value() {
    let index = ChunkIndex::new(5).unwrap();

    assert_eq!(index.value(), 5);
}

#[test]
fn accepts_zero_as_first_chunk_index() {
    let index = ChunkIndex::new(0).unwrap();

    assert_eq!(index.value(), 0);
}

#[test]
fn rejects_negative_chunk_index() {
    assert_eq!(ChunkIndex::new(-1), Err(DomainError::InvalidChunkIndex));
}
