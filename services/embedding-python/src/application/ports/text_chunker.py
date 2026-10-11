"""Port for splitting document text into chunks."""

from typing import Protocol, Sequence


class TextChunker(Protocol):
    """Splits extracted document text into ordered chunks."""

    def chunk(self, text: str) -> Sequence[str]:
        """Return chunks in their original document order."""
        ...
