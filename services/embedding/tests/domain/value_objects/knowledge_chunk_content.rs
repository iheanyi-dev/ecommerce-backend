use embedding::domain::{errors::DomainError, value_objects::KnowledgeChunkContent};

#[test]
fn creates_knowledge_chunk_content() {
    let content = KnowledgeChunkContent::new("Embedding domain knowledge").unwrap();

    assert_eq!(content.value(), "Embedding domain knowledge");
}

#[test]
fn rejects_empty_content() {
    assert_eq!(
        KnowledgeChunkContent::new(""),
        Err(DomainError::EmptyKnowledgeChunkContent)
    );
}

#[test]
fn rejects_whitespace_content() {
    assert_eq!(
        KnowledgeChunkContent::new("   "),
        Err(DomainError::EmptyKnowledgeChunkContent)
    );
}

#[test]
fn displays_content() {
    let content = KnowledgeChunkContent::new("Some content").unwrap();

    assert_eq!(content.to_string(), "Some content");
}
