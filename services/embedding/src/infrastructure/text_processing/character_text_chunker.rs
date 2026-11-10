//! Character-based overlapping text chunker.

use std::error::Error;
use std::fmt;

use crate::application::ports::{PortResult, TextChunker};

/// Configuration errors for the character text chunker.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ChunkerConfigurationError {
    InvalidChunkSize,
    InvalidOverlap,
}

impl fmt::Display for ChunkerConfigurationError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::InvalidChunkSize => write!(formatter, "chunk_size must be positive"),
            Self::InvalidOverlap => {
                write!(
                    formatter,
                    "overlap must be non-negative and smaller than chunk_size"
                )
            }
        }
    }
}

impl Error for ChunkerConfigurationError {}

/// Splits text into ordered, overlapping character windows.
pub struct CharacterTextChunker {
    chunk_size: usize,
    step: usize,
}

impl CharacterTextChunker {
    /// Creates a chunker after validating its configuration.
    pub fn new(chunk_size: usize, overlap: usize) -> Result<Self, ChunkerConfigurationError> {
        if chunk_size == 0 {
            return Err(ChunkerConfigurationError::InvalidChunkSize);
        }

        if overlap >= chunk_size {
            return Err(ChunkerConfigurationError::InvalidOverlap);
        }

        Ok(Self {
            chunk_size,
            step: chunk_size - overlap,
        })
    }
}

impl Default for CharacterTextChunker {
    fn default() -> Self {
        Self::new(2000, 200).expect("default chunker configuration is valid")
    }
}

impl TextChunker for CharacterTextChunker {
    fn chunk(&self, text: &str) -> PortResult<Vec<String>> {
        // Collect Unicode scalar values so chunk sizes match Python characters,
        // rather than UTF-8 byte lengths.
        let characters: Vec<char> = text.chars().collect();

        if characters.iter().all(|character| character.is_whitespace()) {
            return Ok(Vec::new());
        }

        let mut chunks = Vec::new();
        let mut start = 0;

        while start < characters.len() {
            let end = (start + self.chunk_size).min(characters.len());
            let content: String = characters[start..end].iter().collect();

            // Keep meaningful content exactly as it appeared in the source.
            if !content.trim().is_empty() {
                chunks.push(content);
            }

            if end == characters.len() {
                break;
            }

            start += self.step;
        }

        Ok(chunks)
    }
}
