//! Thread-safe in-memory metrics collector.

use std::collections::HashMap;
use std::sync::Mutex;

use crate::application::ports::Metrics;

/// Stores counters and numeric observations in memory.
#[derive(Default)]
pub struct InMemoryMetrics {
    counters: Mutex<HashMap<String, f64>>,
    observations: Mutex<HashMap<String, Vec<f64>>>,
}

impl InMemoryMetrics {
    /// Creates an empty metrics collector.
    pub fn new() -> Self {
        Self::default()
    }

    /// Returns the current counter value, or zero when it does not exist.
    pub fn counter_value(&self, name: &str) -> f64 {
        let counters = self.counters.lock().expect("metrics lock poisoned");
        *counters.get(name).unwrap_or(&0.0)
    }

    /// Returns a copy of the observations for a metric.
    pub fn observation_values(&self, name: &str) -> Vec<f64> {
        let observations = self.observations.lock().expect("metrics lock poisoned");
        observations.get(name).cloned().unwrap_or_default()
    }
}

impl Metrics for InMemoryMetrics {
    fn increment(&self, name: &str, value: u64) {
        assert!(!name.trim().is_empty(), "metric name cannot be empty");

        let mut counters = self.counters.lock().expect("metrics lock poisoned");
        *counters.entry(name.to_owned()).or_insert(0.0) += value as f64;
    }

    fn observe(&self, name: &str, value: f64) {
        assert!(!name.trim().is_empty(), "metric name cannot be empty");

        let mut observations = self.observations.lock().expect("metrics lock poisoned");
        observations.entry(name.to_owned()).or_default().push(value);
    }
}
