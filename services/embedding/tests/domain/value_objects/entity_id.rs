use embedding::domain::{errors::DomainError, value_objects::EntityId};
use uuid::Uuid;

#[test]
fn creates_entity_id_from_uuid() {
    let uuid = Uuid::new_v4();
    let entity_id = EntityId::new(uuid);

    assert_eq!(entity_id.value(), uuid);
}

#[test]
fn parses_valid_uuid_string() {
    let uuid = Uuid::new_v4();
    let entity_id = EntityId::parse(&uuid.to_string(), "Product").unwrap();

    assert_eq!(entity_id.value(), uuid);
}

#[test]
fn rejects_invalid_uuid_string() {
    assert_eq!(
        EntityId::parse("not-a-uuid", "Product"),
        Err(DomainError::InvalidEntityId {
            entity_name: "Product".to_owned()
        })
    );
}

#[test]
fn displays_uuid() {
    let uuid = Uuid::new_v4();
    let entity_id = EntityId::new(uuid);

    assert_eq!(entity_id.to_string(), uuid.to_string());
}
