"""Qdrant client construction."""

from __future__ import annotations

from qdrant_client import QdrantClient


def create_qdrant_client(
    url: str,
    api_key: str | None = None,
) -> QdrantClient:
    """
    Create the synchronous Qdrant client used by the persistence layer.

    The application repository ports are synchronous, so the synchronous
    Qdrant client keeps the infrastructure implementation aligned with those
    contracts.
    """
    if not url.strip():
        raise ValueError("Qdrant URL must not be empty.")

    client_kwargs: dict[str, object] = {
        "url": url,
    }

    if api_key is not None:
        if not api_key.strip():
            raise ValueError("Qdrant API key must not be empty when provided.")
        client_kwargs["api_key"] = api_key

    return QdrantClient(**client_kwargs)
