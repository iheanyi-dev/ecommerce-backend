//! Behavioural tests for Embedding infrastructure adapters.

use std::io::{self, Write};
use std::sync::{Arc, Mutex};

use embedding::application::ports::{Logger, Metrics, TextChunker};
use embedding::infrastructure::{
    observability::{logger::StructuredLogger, metrics::InMemoryMetrics},
    text_processing::character_text_chunker::{CharacterTextChunker, ChunkerConfigurationError},
};

#[derive(Clone)]
struct SharedBuffer(Arc<Mutex<Vec<u8>>>);

impl Write for SharedBuffer {
    fn write(&mut self, bytes: &[u8]) -> io::Result<usize> {
        self.0.lock().unwrap().extend_from_slice(bytes);
        Ok(bytes.len())
    }

    fn flush(&mut self) -> io::Result<()> {
        Ok(())
    }
}

#[test]
fn logger_writes_structured_json_with_sorted_keys() {
    let buffer = Arc::new(Mutex::new(Vec::new()));
    let logger = StructuredLogger::new(SharedBuffer(buffer.clone()));

    logger.info("embedding.created", &[("store_id", "store-1".to_owned())]);

    let output = String::from_utf8(buffer.lock().unwrap().clone()).unwrap();

    assert_eq!(
        output,
        "{\"event\":\"embedding.created\",\"store_id\":\"store-1\"}\n"
    );
}

#[test]
fn logger_writes_warning_and_error_events() {
    let buffer = Arc::new(Mutex::new(Vec::new()));
    let logger = StructuredLogger::new(SharedBuffer(buffer.clone()));

    logger.warning("warning.event", &[]);
    logger.error("error.event", &[]);

    let output = String::from_utf8(buffer.lock().unwrap().clone()).unwrap();

    assert!(output.contains("\"event\":\"warning.event\""));
    assert!(output.contains("\"event\":\"error.event\""));
}

#[test]
fn metrics_accumulate_counter_values() {
    let metrics = InMemoryMetrics::new();

    metrics.increment("embedding.created", 1);
    metrics.increment("embedding.created", 3);

    assert_eq!(metrics.counter_value("embedding.created"), 4.0);
}

#[test]
fn metrics_return_observations_without_exposing_internal_storage() {
    let metrics = InMemoryMetrics::new();

    metrics.observe("embedding.duration", 1.5);
    metrics.observe("embedding.duration", 2.5);

    let mut observations = metrics.observation_values("embedding.duration");
    observations.push(99.0);

    assert_eq!(
        metrics.observation_values("embedding.duration"),
        vec![1.5, 2.5]
    );
}

#[test]
#[should_panic(expected = "metric name cannot be empty")]
fn metrics_reject_empty_counter_names() {
    InMemoryMetrics::new().increment("  ", 1);
}

#[test]
#[should_panic(expected = "metric name cannot be empty")]
fn metrics_reject_empty_observation_names() {
    InMemoryMetrics::new().observe("", 1.0);
}

#[test]
fn chunker_returns_short_text_unchanged() {
    let chunker = CharacterTextChunker::new(10, 2).unwrap();

    assert_eq!(chunker.chunk("hello").unwrap(), vec!["hello"]);
}

#[test]
fn chunker_splits_long_text_with_overlap() {
    let chunker = CharacterTextChunker::new(6, 2).unwrap();

    assert_eq!(
        chunker.chunk("abcdefghijkl").unwrap(),
        vec!["abcdef", "efghij", "ijkl"]
    );
}

#[test]
fn chunker_preserves_overlap_characters() {
    let chunker = CharacterTextChunker::new(5, 2).unwrap();

    let chunks = chunker.chunk("abcdefgh").unwrap();

    assert_eq!(chunks, vec!["abcde", "defgh"]);
    assert_eq!(&chunks[0][3..], &chunks[1][..2]);
}

#[test]
fn chunker_counts_unicode_characters_not_utf8_bytes() {
    let chunker = CharacterTextChunker::new(3, 1).unwrap();

    assert_eq!(chunker.chunk("猫犬鳥魚").unwrap(), vec!["猫犬鳥", "鳥魚"]);
}

#[test]
fn chunker_returns_no_chunks_for_empty_or_whitespace_text() {
    let chunker = CharacterTextChunker::new(10, 2).unwrap();

    assert!(chunker.chunk("").unwrap().is_empty());
    assert!(chunker.chunk("   \n\t ").unwrap().is_empty());
}

#[test]
fn chunker_preserves_whitespace_inside_document() {
    let chunker = CharacterTextChunker::new(4, 1).unwrap();

    assert_eq!(
        chunker.chunk("ab cd ef").unwrap(),
        vec!["ab c", "cd e", "ef"]
    );
}

#[test]
fn chunker_rejects_invalid_configuration() {
    assert_eq!(
        CharacterTextChunker::new(0, 0).err(),
        Some(ChunkerConfigurationError::InvalidChunkSize)
    );
    assert_eq!(
        CharacterTextChunker::new(5, 5).err(),
        Some(ChunkerConfigurationError::InvalidOverlap)
    );
    assert_eq!(
        CharacterTextChunker::new(5, 6).err(),
        Some(ChunkerConfigurationError::InvalidOverlap)
    );
}
