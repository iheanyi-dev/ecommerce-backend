"""Unit tests for Kafka event validation and dispatch."""

import asyncio
import json
from uuid import UUID

import pytest

from infrastructure.messaging.event_handlers import (
    EmbeddingEventHandlers,
    KNOWLEDGE_DELETION_REQUESTED,
    KNOWLEDGE_INGESTION_REQUESTED,
    STORE_EMBEDDING_DELETION_REQUESTED,
    STORE_EMBEDDING_REQUESTED,
)


STORE_ID = "233b7e6c-8920-4861-a778-6f11917db316"
KNOWLEDGE_ID = "e30c5d63-c7ec-48c4-9bba-c3e4c3d8f1aa"


class RecordingUseCase:
    def __init__(self):
        self.requests = []

    def execute(self, request):
        self.requests.append(request)


@pytest.fixture
def handlers():
    use_cases = [
        RecordingUseCase(),
        RecordingUseCase(),
        RecordingUseCase(),
        RecordingUseCase(),
    ]
    return EmbeddingEventHandlers(*use_cases), use_cases


def test_store_embedding_requested_dispatches_validated_event(handlers):
    event_handlers, use_cases = handlers
    event_handlers.handle(
        STORE_EMBEDDING_REQUESTED,
        json.dumps({
            "store_id": STORE_ID,
            "name": "Test Store",
            "description": "Test description",
        }).encode(),
    )

    request = use_cases[0].requests[0]
    assert request.store_id == UUID(STORE_ID)
    assert request.name == "Test Store"
    assert request.description == "Test description"


def test_store_embedding_deletion_dispatches_event(handlers):
    event_handlers, use_cases = handlers
    event_handlers.handle(
        STORE_EMBEDDING_DELETION_REQUESTED,
        json.dumps({"store_id": STORE_ID}),
    )

    assert use_cases[1].requests[0].store_id == UUID(STORE_ID)


def test_knowledge_ingestion_dispatches_event(handlers):
    event_handlers, use_cases = handlers
    event_handlers.handle(
        KNOWLEDGE_INGESTION_REQUESTED,
        json.dumps({
            "knowledge_id": KNOWLEDGE_ID,
            "store_id": STORE_ID,
            "storage_reference": f"knowledge/{STORE_ID}/manual.pdf",
        }),
    )

    request = use_cases[2].requests[0]
    assert request.knowledge_id == UUID(KNOWLEDGE_ID)
    assert request.store_id == UUID(STORE_ID)
    assert request.storage_reference == f"knowledge/{STORE_ID}/manual.pdf"


def test_knowledge_deletion_validates_and_dispatches_event(handlers):
    event_handlers, use_cases = handlers
    event_handlers.handle(
        KNOWLEDGE_DELETION_REQUESTED,
        json.dumps({
            "knowledge_id": KNOWLEDGE_ID,
            "store_id": STORE_ID,
        }),
    )

    assert use_cases[3].requests[0].knowledge_id == UUID(KNOWLEDGE_ID)


@pytest.mark.parametrize(
    ("topic", "payload"),
    [
        (STORE_EMBEDDING_REQUESTED, b"{invalid"),
        (STORE_EMBEDDING_REQUESTED, b"[]"),
        (STORE_EMBEDDING_REQUESTED, json.dumps({
            "store_id": "not-a-uuid",
            "name": "Store",
            "description": "Description",
        })),
        (STORE_EMBEDDING_REQUESTED, json.dumps({
            "store_id": STORE_ID,
            "name": "",
            "description": "Description",
        })),
        (KNOWLEDGE_DELETION_REQUESTED, json.dumps({
            "knowledge_id": KNOWLEDGE_ID,
            "store_id": "invalid",
        })),
    ],
)
def test_invalid_event_is_rejected(handlers, topic, payload):
    event_handlers, use_cases = handlers

    with pytest.raises(ValueError):
        event_handlers.handle(topic, payload)

    assert all(not use_case.requests for use_case in use_cases)


def test_unknown_topic_is_rejected(handlers):
    event_handlers, _ = handlers

    with pytest.raises(ValueError, match="Unsupported Kafka topic"):
        event_handlers.handle("unknown.topic", b"{}")


def test_use_case_failure_propagates_for_redelivery(handlers):
    event_handlers, use_cases = handlers

    def fail(_request):
        raise RuntimeError("repository unavailable")

    use_cases[0].execute = fail

    with pytest.raises(RuntimeError, match="repository unavailable"):
        event_handlers.handle(
            STORE_EMBEDDING_REQUESTED,
            json.dumps({
                "store_id": STORE_ID,
                "name": "Store",
                "description": "Description",
            }),
        )
