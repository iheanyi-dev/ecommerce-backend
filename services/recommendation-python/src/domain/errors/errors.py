"""Domain-specific errors for the Recommendation Service."""


class DomainError(Exception):
    """Base exception for all Recommendation Service domain errors."""


class InvalidEntityIDError(DomainError):
    """Raised when an entity identifier is invalid."""

    def __init__(self, entity_name: str) -> None:
        """Initialize the invalid entity identifier error."""
        super().__init__(f"{entity_name} id must be a UUID")


class EmptyEntityNameError(DomainError):
    """Raised when an entity name is empty."""

    def __init__(self, entity_name: str) -> None:
        """Initialize the empty entity name error."""
        super().__init__(f"{entity_name} name cannot be empty")


class EmptyEntityDescriptionError(DomainError):
    """Raised when an entity description is empty."""

    def __init__(self, entity_name: str) -> None:
        """Initialize the empty entity description error."""
        super().__init__(f"{entity_name} description cannot be empty")


class EmptyEmbeddingError(DomainError):
    """Raised when an embedding vector is empty."""

    def __init__(self) -> None:
        """Initialize the empty embedding error."""
        super().__init__("embedding vector cannot be empty")


class NonFiniteEmbeddingValueError(DomainError):
    """Raised when an embedding contains a non-finite value."""

    def __init__(self) -> None:
        """Initialize the non-finite embedding value error."""
        super().__init__("embedding vector must contain only finite values")


class EmptyEmbeddingModelVersionError(DomainError):
    """Raised when an embedding model version is empty."""

    def __init__(self) -> None:
        """Initialize the empty model version error."""
        super().__init__("embedding model version cannot be empty")
