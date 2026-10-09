"""Tests for the Recommendation Store embedding Kafka consumer."""

from __future__ import annotations

import json
from types import SimpleNamespace
from uuid import uuid4

import pytest

from application.dto.create_store_embedding import CreateStoreEmbeddingInput
from application.dto.delete_store_embedding import DeleteStoreEmbeddingInput
from infrastructure.messaging.kafka.embedding_consumer import (
    KafkaEmbeddingConsumer,
    STORE_EMBEDDING_DELETION_REQUESTED_TOPIC,
    STORE_EMBEDDING_REQUESTED_TOPIC,
)


class FakeCreateStoreEmbedding:
    def __init__(self) -> None:
        self.inputs: list[CreateStoreEmbeddingInput] = []

    def execute(self, input: CreateStoreEmbeddingInput) -> None:
        self.inputs.append(input)


class FakeDeleteStoreEmbedding:
    def __init__(self) -> None:
        self.inputs: list[DeleteStoreEmbeddingInput] = []

    def execute(self, input: DeleteStoreEmbeddingInput) -> None:
        self.inputs.append(input)


def create_consumer() -> tuple[
    KafkaEmbeddingConsumer,
    FakeCreateStoreEmbedding,
    FakeDeleteStoreEmbedding,
]:
    create_use_case = FakeCreateStoreEmbedding()
    delete_use_case = FakeDeleteStoreEmbedding()

    consumer = KafkaEmbeddingConsumer(
        bootstrap_servers="localhost:9092",
        group_id="recommendation-test",
        create_store_embedding=create_use_case,  # type: ignore[arg-type]
        delete_store_embedding=delete_use_case,  # type: ignore[arg-type]
    )

    return consumer, create_use_case, delete_use_case


@pytest.mark.asyncio
async def test_embedding_requested_event_invokes_create_use_case() -> None:
    consumer, create_use_case, _ = create_consumer()

    store_id = uuid4()

    record = SimpleNamespace(
        topic=STORE_EMBEDDING_REQUESTED_TOPIC,
        value=json.dumps(
            {
                "store_id": str(store_id),
                "name": "Test Store",
                "description": "Test description",
            }
        ).encode(),
    )

    await consumer.handle_record(record)

    assert len(create_use_case.inputs) == 1
    assert create_use_case.inputs[0] == CreateStoreEmbeddingInput(
        store_id=store_id,
        name="Test Store",
        description="Test description",
    )


@pytest.mark.asyncio
async def test_embedding_deletion_event_invokes_delete_use_case() -> None:
    consumer, _, delete_use_case = create_consumer()

    store_id = uuid4()

    record = SimpleNamespace(
        topic=STORE_EMBEDDING_DELETION_REQUESTED_TOPIC,
        value=json.dumps(
            {"store_id": str(store_id)}
        ).encode(),
    )

    await consumer.handle_record(record)

    assert len(delete_use_case.inputs) == 1
    assert delete_use_case.inputs[0] == DeleteStoreEmbeddingInput(
        store_id=store_id,
    )


@pytest.mark.asyncio
async def test_invalid_store_id_is_rejected() -> None:
    consumer, create_use_case, _ = create_consumer()

    record = SimpleNamespace(
        topic=STORE_EMBEDDING_REQUESTED_TOPIC,
        value=json.dumps(
            {
                "store_id": "not-a-uuid",
                "name": "Test Store",
                "description": "Test description",
            }
        ).encode(),
    )

    with pytest.raises(ValueError, match="valid UUID"):
        await consumer.handle_record(record)

    assert create_use_case.inputs == []


@pytest.mark.asyncio
async def test_invalid_json_is_rejected() -> None:
    consumer, create_use_case, _ = create_consumer()

    record = SimpleNamespace(
        topic=STORE_EMBEDDING_REQUESTED_TOPIC,
        value=b"not-json",
    )

    with pytest.raises(ValueError, match="invalid JSON"):
        await consumer.handle_record(record)

    assert create_use_case.inputs == []


@pytest.mark.asyncio
async def test_unknown_topic_is_rejected() -> None:
    consumer, _, _ = create_consumer()

    record = SimpleNamespace(
        topic="unknown.topic",
        value=b"{}",
    )

    with pytest.raises(ValueError, match="Unsupported Kafka topic"):
        await consumer.handle_record(record)
