//! Kafka consumer with sequential processing and explicit offset commits.
//!
//! Each event is handled before the next event is read. Its offset is
//! committed only after the application handler succeeds. Handler failures
//! therefore escape the consumer loop and leave the event available for
//! redelivery when the worker restarts.

use std::{error::Error, io, sync::Arc};

use futures_util::StreamExt;
use rdkafka::{
    ClientConfig,
    consumer::{CommitMode, Consumer, StreamConsumer},
    message::Message,
};

use crate::application::ports::Logger;

use super::event_handlers::EmbeddingEventHandlers;

/// Maximum time Kafka allows between consumer poll operations: 30 minutes.
const MAX_POLL_INTERVAL_MS: &str = "1800000";

/// Errors returned by the Kafka consumer.
type ConsumerResult<T> = Result<T, Box<dyn Error + Send + Sync>>;

/// Sequential Kafka event consumer.
///
/// The application handlers and logger are shared through `Arc`, allowing
/// the consumer to run as an infrastructure worker without owning duplicate
/// application dependencies.
pub struct KafkaEventConsumer {
    brokers: String,
    group_id: String,
    topics: Vec<String>,
    client_id: String,
    handlers: Arc<EmbeddingEventHandlers>,
    logger: Arc<dyn Logger>,
}

impl KafkaEventConsumer {
    /// Creates a consumer after validating its required configuration.
    ///
    /// The client ID defaults to `embedding-service`, matching the Python
    /// implementation.
    pub fn new(
        brokers: String,
        group_id: String,
        topics: Vec<String>,
        handlers: Arc<EmbeddingEventHandlers>,
        logger: Arc<dyn Logger>,
    ) -> ConsumerResult<Self> {
        validate_configuration(&brokers, &group_id, &topics)?;

        Ok(Self {
            brokers,
            group_id,
            topics,
            client_id: "embedding-service".to_owned(),
            handlers,
            logger,
        })
    }

    /// Starts consuming events until Kafka ends the stream or an error occurs.
    ///
    /// Automatic commits are disabled. The consumer is unsubscribed and the
    /// stopped event is logged after the processing loop exits, including
    /// when event handling or offset committing fails.
    pub async fn run(&self) -> ConsumerResult<()> {
        let consumer: StreamConsumer = ClientConfig::new()
            .set("bootstrap.servers", &self.brokers)
            .set("group.id", &self.group_id)
            .set("client.id", &self.client_id)
            .set("enable.auto.commit", "false")
            .set("auto.offset.reset", "earliest")
            .set("max.poll.interval.ms", MAX_POLL_INTERVAL_MS)
            .create()?;

        let topics: Vec<&str> = self.topics.iter().map(String::as_str).collect();
        consumer.subscribe(&topics)?;

        self.logger.info(
            "embedding.kafka_consumer.started",
            &[
                ("group_id", self.group_id.clone()),
                ("topics", self.topics.join(",")),
            ],
        );

        // Keep cleanup outside the processing loop so it runs for both
        // successful stream completion and returned processing errors.
        let result = self.consume(&consumer).await;

        consumer.unsubscribe();

        self.logger.info("embedding.kafka_consumer.stopped", &[]);

        result
    }

    /// Processes Kafka records sequentially and commits each successful one.
    async fn consume(&self, consumer: &StreamConsumer) -> ConsumerResult<()> {
        let mut stream = consumer.stream();

        while let Some(record_result) = stream.next().await {
            let record = record_result?;

            // Copy message data before moving work to the blocking thread.
            let topic = record.topic().to_owned();
            let partition = record.partition();
            let offset = record.offset();

            let payload = record.payload().ok_or_else(|| {
                io::Error::new(
                    io::ErrorKind::InvalidData,
                    "Kafka event payload must not be null.",
                )
            })?;
            let payload = payload.to_vec();

            // Application handlers are synchronous. Run them on Tokio's
            // blocking pool so model and database work does not block the
            // asynchronous Kafka runtime thread.
            let handlers = Arc::clone(&self.handlers);
            let handler_topic = topic.clone();

            let handler_result =
                tokio::task::spawn_blocking(move || handlers.handle(&handler_topic, &payload))
                    .await?;

            // Propagate application failures. In particular, do not commit
            // the record when validation or use-case execution fails.
            handler_result?;

            // Commit synchronously only after successful event processing.
            // This ensures the next record is not processed before the
            // current record's offset has been committed.
            consumer.commit_message(&record, CommitMode::Sync)?;

            self.logger.info(
                "embedding.kafka_event.processed",
                &[
                    ("topic", topic),
                    ("partition", partition.to_string()),
                    ("offset", offset.to_string()),
                ],
            );
        }

        Ok(())
    }
}

/// Validates the consumer configuration before connecting to Kafka.
///
/// This follows the Python consumer's required configuration checks.
fn validate_configuration(brokers: &str, group_id: &str, topics: &[String]) -> ConsumerResult<()> {
    if brokers.trim().is_empty() {
        return Err(io::Error::new(
            io::ErrorKind::InvalidInput,
            "Kafka bootstrap servers must not be empty.",
        )
        .into());
    }

    if group_id.trim().is_empty() {
        return Err(io::Error::new(
            io::ErrorKind::InvalidInput,
            "Kafka consumer group ID must not be empty.",
        )
        .into());
    }

    if topics.is_empty() {
        return Err(io::Error::new(
            io::ErrorKind::InvalidInput,
            "At least one Kafka topic is required.",
        )
        .into());
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    /// Ensures a complete consumer configuration is accepted.
    #[test]
    fn validate_configuration_accepts_valid_values() {
        let topics = vec!["store.embedding.requested".to_owned()];

        assert!(validate_configuration("localhost:9092", "embedding", &topics).is_ok());
    }

    /// Ensures empty bootstrap servers are rejected.
    #[test]
    fn validate_configuration_rejects_empty_brokers() {
        let topics = vec!["store.embedding.requested".to_owned()];

        let error = validate_configuration("  ", "embedding", &topics).unwrap_err();

        assert_eq!(
            error.to_string(),
            "Kafka bootstrap servers must not be empty."
        );
    }

    /// Ensures an empty consumer group is rejected.
    #[test]
    fn validate_configuration_rejects_empty_group_id() {
        let topics = vec!["store.embedding.requested".to_owned()];

        let error = validate_configuration("localhost:9092", " ", &topics).unwrap_err();

        assert_eq!(
            error.to_string(),
            "Kafka consumer group ID must not be empty."
        );
    }

    /// Ensures at least one subscribed topic is required.
    #[test]
    fn validate_configuration_rejects_empty_topics() {
        let error = validate_configuration("localhost:9092", "embedding", &[]).unwrap_err();

        assert_eq!(error.to_string(), "At least one Kafka topic is required.");
    }
}
