"""Port for reading the text content of a stored document."""

from typing import Protocol

from domain.value_objects import StorageReference


class DocumentContentReader(Protocol):
    """Reads a document identified by its Storage reference."""

    def read(self, storage_reference: StorageReference) -> str:
        """Return the document's extracted text."""
        ...
