use embedding::domain::{errors::DomainError, value_objects::EntityName};

#[test]
fn creates_entity_name() {
    let name = EntityName::new("Wireless Headphones", "Product").unwrap();

    assert_eq!(name.value(), "Wireless Headphones");
}

#[test]
fn rejects_empty_entity_name() {
    assert_eq!(
        EntityName::new("", "Product"),
        Err(DomainError::EmptyEntityName {
            entity_name: "Product".to_owned()
        })
    );
}

#[test]
fn rejects_whitespace_entity_name() {
    assert_eq!(
        EntityName::new("   ", "Store"),
        Err(DomainError::EmptyEntityName {
            entity_name: "Store".to_owned()
        })
    );
}

#[test]
fn displays_entity_name() {
    let name = EntityName::new("Example Store", "Store").unwrap();

    assert_eq!(name.to_string(), "Example Store");
}
