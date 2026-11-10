use embedding::domain::{errors::DomainError, value_objects::EntityDescription};

#[test]
fn creates_entity_description() {
    let description = EntityDescription::new("A useful product description", "Product").unwrap();

    assert_eq!(description.value(), "A useful product description");
}

#[test]
fn rejects_empty_description() {
    assert_eq!(
        EntityDescription::new("", "Product"),
        Err(DomainError::EmptyEntityDescription {
            entity_name: "Product".to_owned()
        })
    );
}

#[test]
fn rejects_whitespace_description() {
    assert_eq!(
        EntityDescription::new("   ", "Store"),
        Err(DomainError::EmptyEntityDescription {
            entity_name: "Store".to_owned()
        })
    );
}

#[test]
fn displays_description() {
    let description = EntityDescription::new("Description", "Product").unwrap();

    assert_eq!(description.to_string(), "Description");
}
