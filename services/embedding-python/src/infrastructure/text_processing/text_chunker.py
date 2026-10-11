"""Character-based text chunking adapter."""

from collections.abc import Sequence


class CharacterTextChunker:
    """Split extracted document text into overlapping character windows.

    Character-based chunking is deterministic and language-agnostic. It avoids
    assuming that whitespace-delimited words work for every language.

    The overlap preserves context across adjacent chunks. Callers should choose
    chunk sizes appropriate for the embedding model's token limit.
    """

    def __init__(self, chunk_size: int = 2000, overlap: int = 200) -> None:
        if isinstance(chunk_size, bool) or not isinstance(chunk_size, int):
            raise TypeError("chunk_size must be an integer")
        if isinstance(overlap, bool) or not isinstance(overlap, int):
            raise TypeError("overlap must be an integer")
        if chunk_size <= 0:
            raise ValueError("chunk_size must be positive")
        if overlap < 0:
            raise ValueError("overlap must not be negative")
        if overlap >= chunk_size:
            raise ValueError("overlap must be smaller than chunk_size")

        self._chunk_size = chunk_size
        self._step = chunk_size - overlap

    def chunk(self, text: str) -> Sequence[str]:
        """Return ordered, non-empty chunks while preserving source characters."""
        if not isinstance(text, str):
            raise TypeError("text must be a string")

        # Empty or whitespace-only documents should not create empty vectors.
        if not text.strip():
            return ()

        chunks: list[str] = []
        start = 0

        while start < len(text):
            end = min(start + self._chunk_size, len(text))
            content = text[start:end]

            # Preserve meaningful document content rather than silently
            # stripping whitespace that may separate adjacent source segments.
            if content.strip():
                chunks.append(content)

            if end == len(text):
                break

            start += self._step

        return tuple(chunks)
