"""Domain-specific errors for the Embedding Service."""


class DomainError(Exception):
    """Base exception for domain errors."""


class InvalidEntityIDError(DomainError):
    """Raised when an entity identifier is not a UUID."""

    def __init__(self, entity_name: str) -> None:
        super().__init__(f"{entity_name} id must be a UUID")


class EmptyEntityNameError(DomainError):
    """Raised when an entity name is empty."""

    def __init__(self, entity_name: str) -> None:
        super().__init__(f"{entity_name} name cannot be empty")


class EmptyEntityDescriptionError(DomainError):
    """Raised when an entity description is empty."""

    def __init__(self, entity_name: str) -> None:
        super().__init__(f"{entity_name} description cannot be empty")


class EmptyEmbeddingError(DomainError):
    """Raised when an embedding vector is empty."""

    def __init__(self) -> None:
        super().__init__("embedding vector cannot be empty")


class NonFiniteEmbeddingValueError(DomainError):
    """Raised when an embedding contains non-finite values."""

    def __init__(self) -> None:
        super().__init__("embedding vector must contain only finite values")


class EmptyEmbeddingModelVersionError(DomainError):
    """Raised when the embedding model version is empty."""

    def __init__(self) -> None:
        super().__init__("embedding model version cannot be empty")


class EmptyKnowledgeChunkContentError(DomainError):
    """Raised when a knowledge chunk has no content."""

    def __init__(self) -> None:
        super().__init__("knowledge chunk content cannot be empty")


class EmptyStorageReferenceError(DomainError):
    """Raised when a knowledge storage reference is empty."""

    def __init__(self) -> None:
        super().__init__("storage reference cannot be empty")


class InvalidChunkIndexError(DomainError):
    """Raised when a chunk index is negative or not an integer."""

    def __init__(self) -> None:
        super().__init__("chunk index must be a non-negative integer")
