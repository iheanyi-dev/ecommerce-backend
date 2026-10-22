"""Translate Kafka event payloads into application use-case requests."""

from __future__ import annotations

import json
from collections.abc import Awaitable, Callable
from dataclasses import dataclass
from typing import Any
from uuid import UUID

from application.dto.create_store_embedding import CreateStoreEmbeddingInput
from application.dto.delete_store_embedding import DeleteStoreEmbeddingInput
from application.dto.ingest_knowledge import IngestKnowledgeInput
from application.dto.delete_knowledge import DeleteKnowledgeInput
from application.dto.create_product_embedding import CreateProductEmbeddingInput
from application.dto.delete_product_embedding import DeleteProductEmbeddingInput


STORE_EMBEDDING_REQUESTED = "store.embedding.requested"
STORE_EMBEDDING_DELETION_REQUESTED = "store.embedding.deletion.requested"
KNOWLEDGE_INGESTION_REQUESTED = "knowledge.ingestion.requested"
KNOWLEDGE_DELETION_REQUESTED = "knowledge.deletion.requested"
PRODUCT_EMBEDDING_REQUESTED = "product.embedding.requested"
PRODUCT_EMBEDDING_DELETION_REQUESTED = "product.embedding.deletion.requested"

EVENT_TOPICS = (
    STORE_EMBEDDING_REQUESTED,
    STORE_EMBEDDING_DELETION_REQUESTED,
    KNOWLEDGE_INGESTION_REQUESTED,
    KNOWLEDGE_DELETION_REQUESTED,
    PRODUCT_EMBEDDING_REQUESTED,
    PRODUCT_EMBEDDING_DELETION_REQUESTED,
)


def _payload(raw: bytes | str) -> dict[str, Any]:
    try:
        decoded = raw.decode("utf-8") if isinstance(raw, bytes) else raw
        value = json.loads(decoded)
    except (UnicodeDecodeError, json.JSONDecodeError) as exc:
        raise ValueError("Kafka event must contain valid UTF-8 JSON.") from exc

    if not isinstance(value, dict):
        raise ValueError("Kafka event payload must be a JSON object.")

    return value


def _required_string(event: dict[str, Any], field: str) -> str:
    value = event.get(field)
    if not isinstance(value, str) or not value.strip():
        raise ValueError(f"Kafka event field '{field}' must be a non-empty string.")
    return value


def _required_uuid(event: dict[str, Any], field: str) -> UUID:
    value = _required_string(event, field)
    try:
        return UUID(value)
    except ValueError as exc:
        raise ValueError(f"Kafka event field '{field}' must be a valid UUID.") from exc


@dataclass(slots=True)
class EmbeddingEventHandlers:
    """Dispatch validated event payloads to synchronous application use cases."""

    create_store_embedding: Any
    delete_store_embedding: Any
    ingest_knowledge: Any
    delete_knowledge: Any
    create_product_embedding: Any
    delete_product_embedding: Any

    def handle(self, topic: str, raw: bytes | str) -> None:
        event = _payload(raw)

        if topic == STORE_EMBEDDING_REQUESTED:
            request = CreateStoreEmbeddingInput(
                store_id=_required_uuid(event, "store_id"),
                name=_required_string(event, "name"),
                description=_required_string(event, "description"),
            )
            self.create_store_embedding.execute(request)
            return

        if topic == STORE_EMBEDDING_DELETION_REQUESTED:
            request = DeleteStoreEmbeddingInput(
                store_id=_required_uuid(event, "store_id"),
            )
            self.delete_store_embedding.execute(request)
            return

        if topic == KNOWLEDGE_INGESTION_REQUESTED:
            request = IngestKnowledgeInput(
                knowledge_id=_required_uuid(event, "knowledge_id"),
                store_id=_required_uuid(event, "store_id"),
                storage_reference=_required_string(event, "storage_reference"),
            )
            self.ingest_knowledge.execute(request)
            return

        if topic == KNOWLEDGE_DELETION_REQUESTED:
            # Validate store_id too: it is part of the published event contract,
            # even though deleting chunks currently requires only knowledge_id.
            _required_uuid(event, "store_id")
            request = DeleteKnowledgeInput(
                knowledge_id=_required_uuid(event, "knowledge_id"),
            )
            self.delete_knowledge.execute(request)
            return

        if topic == PRODUCT_EMBEDDING_REQUESTED:
            request = CreateProductEmbeddingInput(
                product_id=_required_uuid(event, "product_id"),
                store_id=_required_uuid(event, "store_id"),
                name=_required_string(event, "name"),
                description=_required_string(event, "description"),
            )
            self.create_product_embedding.execute(request)
            return

        if topic == PRODUCT_EMBEDDING_DELETION_REQUESTED:
            # Validate store_id because it is part of the published event contract.
            _required_uuid(event, "store_id")
            request = DeleteProductEmbeddingInput(
                product_id=_required_uuid(event, "product_id"),
            )
            self.delete_product_embedding.execute(request)
            return

        raise ValueError(f"Unsupported Kafka topic: {topic}")
