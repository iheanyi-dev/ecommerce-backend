//! Errors produced by the application layer.

use std::error::Error;
use std::fmt;

/// Application-level errors.
#[derive(Debug)]
pub enum ApplicationError {
    /// A supplied application input is invalid.
    Validation(String),

    /// An operation failed in an application port.
    Port(Box<dyn Error + Send + Sync>),

    /// Domain validation or construction failed.
    Domain(crate::domain::errors::DomainError),
}

impl fmt::Display for ApplicationError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Validation(message) => write!(formatter, "validation error: {message}"),
            Self::Port(error) => write!(formatter, "port error: {error}"),
            Self::Domain(error) => write!(formatter, "domain error: {error}"),
        }
    }
}

impl Error for ApplicationError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Port(error) => Some(error.as_ref()),
            Self::Domain(error) => Some(error),
            Self::Validation(_) => None,
        }
    }
}

impl From<crate::domain::errors::DomainError> for ApplicationError {
    fn from(error: crate::domain::errors::DomainError) -> Self {
        Self::Domain(error)
    }
}

impl From<Box<dyn Error + Send + Sync>> for ApplicationError {
    fn from(error: Box<dyn Error + Send + Sync>) -> Self {
        Self::Port(error)
    }
}

/// Result returned by application use cases.
pub type ApplicationResult<T> = Result<T, ApplicationError>;
