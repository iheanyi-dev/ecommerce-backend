//! Translates Kafka event payloads into application use-case requests.
//!
//! This adapter is responsible for:
//! - Identifying supported Kafka topics.
//! - Parsing and validating JSON event payloads.
//! - Converting event fields into strongly typed application DTOs.
//! - Dispatching requests to the appropriate application use case.
//!
//! Application errors are deliberately propagated to the caller so that
//! the Kafka consumer can avoid committing unsuccessful events.

use std::sync::Arc;

use serde_json::{Map, Value};
use uuid::Uuid;

use crate::application::{
    dto::{
        CreateProductEmbeddingInput, CreateStoreEmbeddingInput, DeleteKnowledgeInput,
        DeleteProductEmbeddingInput, DeleteStoreEmbeddingInput, IngestKnowledgeInput,
    },
    errors::{ApplicationError, ApplicationResult},
    use_cases::{
        create_product_embedding::CreateProductEmbeddingUseCase,
        create_store_embedding::CreateStoreEmbeddingUseCase,
        delete_knowledge::DeleteKnowledgeUseCase,
        delete_product_embedding::DeleteProductEmbeddingUseCase,
        delete_store_embedding::DeleteStoreEmbeddingUseCase,
        ingest_knowledge::IngestKnowledgeUseCase,
    },
};

/// Kafka topic requesting creation of a store embedding.
pub const STORE_EMBEDDING_REQUESTED: &str = "store.embedding.requested";

/// Kafka topic requesting deletion of a store and its product embeddings.
pub const STORE_EMBEDDING_DELETION_REQUESTED: &str = "store.embedding.deletion.requested";

/// Kafka topic requesting ingestion of a knowledge document.
pub const KNOWLEDGE_INGESTION_REQUESTED: &str = "knowledge.ingestion.requested";

/// Kafka topic requesting deletion of a knowledge document's chunks.
pub const KNOWLEDGE_DELETION_REQUESTED: &str = "knowledge.deletion.requested";

/// Kafka topic requesting creation of a product embedding.
pub const PRODUCT_EMBEDDING_REQUESTED: &str = "product.embedding.requested";

/// Kafka topic requesting deletion of a product embedding.
pub const PRODUCT_EMBEDDING_DELETION_REQUESTED: &str = "product.embedding.deletion.requested";

/// All Kafka topics consumed by the Embedding service.
pub const EVENT_TOPICS: [&str; 6] = [
    STORE_EMBEDDING_REQUESTED,
    STORE_EMBEDDING_DELETION_REQUESTED,
    KNOWLEDGE_INGESTION_REQUESTED,
    KNOWLEDGE_DELETION_REQUESTED,
    PRODUCT_EMBEDDING_REQUESTED,
    PRODUCT_EMBEDDING_DELETION_REQUESTED,
];

/// Holds the application use cases responsible for processing Kafka events.
///
/// Each use case is shared through an `Arc`, allowing this handler to be
/// safely shared with infrastructure workers without owning duplicate
/// application dependencies.
pub struct EmbeddingEventHandlers {
    create_store_embedding: Arc<CreateStoreEmbeddingUseCase>,
    delete_store_embedding: Arc<DeleteStoreEmbeddingUseCase>,
    ingest_knowledge: Arc<IngestKnowledgeUseCase>,
    delete_knowledge: Arc<DeleteKnowledgeUseCase>,
    create_product_embedding: Arc<CreateProductEmbeddingUseCase>,
    delete_product_embedding: Arc<DeleteProductEmbeddingUseCase>,
}

impl EmbeddingEventHandlers {
    /// Creates the event handler with all required application use cases.
    ///
    /// Keeping these dependencies explicit makes the adapter easy to wire
    /// through the infrastructure composition root later.
    pub fn new(
        create_store_embedding: Arc<CreateStoreEmbeddingUseCase>,
        delete_store_embedding: Arc<DeleteStoreEmbeddingUseCase>,
        ingest_knowledge: Arc<IngestKnowledgeUseCase>,
        delete_knowledge: Arc<DeleteKnowledgeUseCase>,
        create_product_embedding: Arc<CreateProductEmbeddingUseCase>,
        delete_product_embedding: Arc<DeleteProductEmbeddingUseCase>,
    ) -> Self {
        Self {
            create_store_embedding,
            delete_store_embedding,
            ingest_knowledge,
            delete_knowledge,
            create_product_embedding,
            delete_product_embedding,
        }
    }

    /// Validates and dispatches one Kafka event to its application use case.
    ///
    /// The raw payload must contain a UTF-8 JSON object. Validation errors
    /// and application failures are returned to the consumer unchanged.
    /// The consumer is responsible for committing the offset only after
    /// this method returns successfully.
    pub fn handle(&self, topic: &str, raw: &[u8]) -> ApplicationResult<()> {
        // Parse the payload before dispatching, matching the Python adapter's
        // behavior: malformed payloads are rejected even for unknown topics.
        let event = parse_payload(raw)?;

        match topic {
            STORE_EMBEDDING_REQUESTED => {
                let request = CreateStoreEmbeddingInput {
                    store_id: required_uuid(&event, "store_id")?,
                    name: required_string(&event, "name")?,
                    description: required_string(&event, "description")?,
                };

                self.create_store_embedding.execute(request)?;
            }

            STORE_EMBEDDING_DELETION_REQUESTED => {
                let request = DeleteStoreEmbeddingInput {
                    store_id: required_uuid(&event, "store_id")?,
                };

                self.delete_store_embedding.execute(request)?;
            }

            KNOWLEDGE_INGESTION_REQUESTED => {
                let request = IngestKnowledgeInput {
                    knowledge_id: required_uuid(&event, "knowledge_id")?,
                    store_id: required_uuid(&event, "store_id")?,
                    storage_reference: required_string(&event, "storage_reference")?,
                };

                self.ingest_knowledge.execute(request)?;
            }

            KNOWLEDGE_DELETION_REQUESTED => {
                // Store ID is part of the published event contract and must
                // remain validated, even though deletion currently needs
                // only the knowledge ID.
                required_uuid(&event, "store_id")?;

                let request = DeleteKnowledgeInput {
                    knowledge_id: required_uuid(&event, "knowledge_id")?,
                };

                self.delete_knowledge.execute(request)?;
            }

            PRODUCT_EMBEDDING_REQUESTED => {
                let request = CreateProductEmbeddingInput {
                    product_id: required_uuid(&event, "product_id")?,
                    store_id: required_uuid(&event, "store_id")?,
                    name: required_string(&event, "name")?,
                    description: required_string(&event, "description")?,
                };

                self.create_product_embedding.execute(request)?;
            }

            PRODUCT_EMBEDDING_DELETION_REQUESTED => {
                // Store ID remains a required part of the event contract,
                // even though deleting a product embedding needs only its ID.
                required_uuid(&event, "store_id")?;

                let request = DeleteProductEmbeddingInput {
                    product_id: required_uuid(&event, "product_id")?,
                };

                self.delete_product_embedding.execute(request)?;
            }

            _ => {
                return Err(ApplicationError::Validation(format!(
                    "unsupported Kafka topic: {topic}"
                )));
            }
        }

        Ok(())
    }
}

/// Decodes a Kafka payload into a JSON object.
///
/// Both UTF-8 decoding and JSON parsing failures become application
/// validation errors. Arrays, strings, numbers, booleans and null are
/// rejected because the event contract requires a JSON object.
fn parse_payload(raw: &[u8]) -> ApplicationResult<Map<String, Value>> {
    let decoded = std::str::from_utf8(raw).map_err(|_| {
        ApplicationError::Validation("Kafka event must contain valid UTF-8 JSON.".into())
    })?;

    let value: Value = serde_json::from_str(decoded).map_err(|_| {
        ApplicationError::Validation("Kafka event must contain valid UTF-8 JSON.".into())
    })?;

    value.as_object().cloned().ok_or_else(|| {
        ApplicationError::Validation("Kafka event payload must be a JSON object.".into())
    })
}

/// Extracts a required non-empty string from an event payload.
///
/// Whitespace-only strings are rejected. The original string is returned
/// without trimming, leaving normalization to the application use case.
fn required_string(event: &Map<String, Value>, field: &str) -> ApplicationResult<String> {
    match event.get(field).and_then(Value::as_str) {
        Some(value) if !value.trim().is_empty() => Ok(value.to_owned()),
        _ => Err(ApplicationError::Validation(format!(
            "Kafka event field '{field}' must be a non-empty string."
        ))),
    }
}

/// Extracts and parses a required UUID from an event payload.
///
/// UUID values must first satisfy the required-string contract and then
/// successfully parse into the domain's UUID representation.
fn required_uuid(event: &Map<String, Value>, field: &str) -> ApplicationResult<Uuid> {
    let value = required_string(event, field)?;

    Uuid::parse_str(&value).map_err(|_| {
        ApplicationError::Validation(format!("Kafka event field '{field}' must be a valid UUID."))
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    /// Ensures every Python event topic is represented in the Rust adapter.
    #[test]
    fn event_topics_contains_all_six_supported_topics() {
        assert_eq!(EVENT_TOPICS.len(), 6);

        assert!(EVENT_TOPICS.contains(&STORE_EMBEDDING_REQUESTED));
        assert!(EVENT_TOPICS.contains(&STORE_EMBEDDING_DELETION_REQUESTED));
        assert!(EVENT_TOPICS.contains(&KNOWLEDGE_INGESTION_REQUESTED));
        assert!(EVENT_TOPICS.contains(&KNOWLEDGE_DELETION_REQUESTED));
        assert!(EVENT_TOPICS.contains(&PRODUCT_EMBEDDING_REQUESTED));
        assert!(EVENT_TOPICS.contains(&PRODUCT_EMBEDDING_DELETION_REQUESTED));
    }

    /// Ensures valid JSON objects are accepted.
    #[test]
    fn parse_payload_accepts_json_objects() {
        let event = parse_payload(br#"{"name":"Example"}"#).unwrap();

        assert_eq!(event.get("name").and_then(Value::as_str), Some("Example"));
    }

    /// Ensures malformed JSON is rejected.
    #[test]
    fn parse_payload_rejects_invalid_json() {
        let result = parse_payload(b"{invalid json}");

        assert!(matches!(
            result,
            Err(ApplicationError::Validation(message))
                if message == "Kafka event must contain valid UTF-8 JSON."
        ));
    }

    /// Ensures invalid UTF-8 payloads are rejected.
    #[test]
    fn parse_payload_rejects_invalid_utf8() {
        let result = parse_payload(&[0xff, 0xfe, 0xfd]);

        assert!(matches!(
            result,
            Err(ApplicationError::Validation(message))
                if message == "Kafka event must contain valid UTF-8 JSON."
        ));
    }

    /// Ensures non-object JSON values are rejected.
    #[test]
    fn parse_payload_rejects_non_object_json() {
        for payload in [
            br#"["value"]"#.as_slice(),
            br#""value""#.as_slice(),
            b"null".as_slice(),
            b"123".as_slice(),
        ] {
            assert!(matches!(
                parse_payload(payload),
                Err(ApplicationError::Validation(message))
                    if message == "Kafka event payload must be a JSON object."
            ));
        }
    }

    /// Ensures required non-empty string values are accepted unchanged.
    #[test]
    fn required_string_accepts_valid_string() {
        let event = parse_payload(br#"{"name":" Example "}"#).unwrap();

        assert_eq!(required_string(&event, "name").unwrap(), " Example ");
    }

    /// Ensures missing, non-string and whitespace-only fields are rejected.
    #[test]
    fn required_string_rejects_invalid_values() {
        for payload in [
            br#"{}"#.as_slice(),
            br#"{"name":null}"#.as_slice(),
            br#"{"name":123}"#.as_slice(),
            br#"{"name":""}"#.as_slice(),
            br#"{"name":"   "}"#.as_slice(),
        ] {
            let event = parse_payload(payload).unwrap();

            assert!(matches!(
                required_string(&event, "name"),
                Err(ApplicationError::Validation(_))
            ));
        }
    }

    /// Ensures valid UUID strings are converted into UUID values.
    #[test]
    fn required_uuid_accepts_valid_uuid() {
        let expected = Uuid::new_v4();
        let payload = format!(r#"{{"store_id":"{expected}"}}"#);
        let event = parse_payload(payload.as_bytes()).unwrap();

        assert_eq!(required_uuid(&event, "store_id").unwrap(), expected);
    }

    /// Ensures invalid and missing UUID values are rejected.
    #[test]
    fn required_uuid_rejects_invalid_values() {
        for payload in [
            br#"{}"#.as_slice(),
            br#"{"store_id":null}"#.as_slice(),
            br#"{"store_id":""}"#.as_slice(),
            br#"{"store_id":"not-a-uuid"}"#.as_slice(),
        ] {
            let event = parse_payload(payload).unwrap();

            assert!(matches!(
                required_uuid(&event, "store_id"),
                Err(ApplicationError::Validation(_))
            ));
        }
    }
}
