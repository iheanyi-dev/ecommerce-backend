//! JSON structured logger implementation.

use std::io::{self, Write};
use std::sync::Mutex;

use serde_json::{Map, Value};

use crate::application::ports::Logger;

/// Writes one JSON object per application event.
pub struct StructuredLogger {
    writer: Mutex<Box<dyn Write + Send>>,
}

impl StructuredLogger {
    /// Creates a logger writing to the supplied output.
    pub fn new(writer: impl Write + Send + 'static) -> Self {
        Self {
            writer: Mutex::new(Box::new(writer)),
        }
    }

    /// Creates a production logger writing to stdout.
    pub fn production() -> Self {
        Self::new(io::stdout())
    }

    /// Serializes and writes one structured event.
    fn write_event(&self, event: &str, fields: &[(&str, String)]) {
        let mut payload = Map::new();
        payload.insert("event".to_owned(), Value::String(event.to_owned()));

        // Match Python's dictionary expansion: supplied fields can override event.
        for (name, value) in fields {
            payload.insert(name.to_string(), Value::String(value.clone()));
        }

        // serde_json's default map implementation keeps keys sorted.
        let serialized =
            serde_json::to_string(&payload).expect("serializing a JSON log event should not fail");

        let mut writer = self.writer.lock().expect("logger lock poisoned");
        let _ = writeln!(writer, "{serialized}");
    }
}

impl Logger for StructuredLogger {
    fn info(&self, event: &str, fields: &[(&str, String)]) {
        self.write_event(event, fields);
    }

    fn warning(&self, event: &str, fields: &[(&str, String)]) {
        self.write_event(event, fields);
    }

    fn error(&self, event: &str, fields: &[(&str, String)]) {
        self.write_event(event, fields);
    }
}
