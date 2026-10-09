"""Domain errors for the chat service."""


class DomainError(Exception):
    """Base exception for all chat domain errors."""


class InvalidMessageContentError(DomainError):
    """Raised when message content is empty or contains only whitespace."""


class InvalidMessageRoleError(DomainError):
    """Raised when a message role is unsupported."""
