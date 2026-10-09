"""Integration tests for the Recommendation Kafka embedding consumer."""

from __future__ import annotations

import asyncio
import json
import os
from dataclasses import dataclass
from uuid import UUID, uuid4

import pytest
from aiokafka import AIOKafkaConsumer, AIOKafkaProducer

from application.dto.create_store_embedding import CreateStoreEmbeddingInput
from application.dto.delete_store_embedding import DeleteStoreEmbeddingInput
from application.use_cases.create_store_embedding import CreateStoreEmbedding
from application.use_cases.delete_store_embedding import DeleteStoreEmbedding
from infrastructure.messaging.kafka.embedding_consumer import (
    KafkaEmbeddingConsumer,
    STORE_EMBEDDING_DELETION_REQUESTED_TOPIC,
    STORE_EMBEDDING_REQUESTED_TOPIC,
)


KAFKA_BROKERS = os.getenv(
    "KAFKA_BROKERS",
    "localhost:19092",
)


@dataclass
class RecordingCreateStoreEmbedding:
    """Record create requests received by the Kafka consumer."""

    inputs: list[CreateStoreEmbeddingInput]

    def execute(self, input: CreateStoreEmbeddingInput) -> None:
        self.inputs.append(input)


@dataclass
class RecordingDeleteStoreEmbedding:
    """Record delete requests received by the Kafka consumer."""

    inputs: list[DeleteStoreEmbeddingInput]

    def execute(self, input: DeleteStoreEmbeddingInput) -> None:
        self.inputs.append(input)


@pytest.mark.asyncio
async def test_real_kafka_events_reach_embedding_use_cases() -> None:
    """Real Redpanda events are consumed and dispatched to use cases."""
    create_use_case = RecordingCreateStoreEmbedding(inputs=[])
    delete_use_case = RecordingDeleteStoreEmbedding(inputs=[])

    group_id = f"recommendation-integration-{uuid4()}"

    consumer = KafkaEmbeddingConsumer(
        bootstrap_servers=KAFKA_BROKERS,
        group_id=group_id,
        create_store_embedding=create_use_case,  # type: ignore[arg-type]
        delete_store_embedding=delete_use_case,  # type: ignore[arg-type]
    )

    producer = AIOKafkaProducer(
        bootstrap_servers=KAFKA_BROKERS,
    )

    create_store_id = uuid4()
    delete_store_id = uuid4()

    try:
        await producer.start()
        await consumer.start()

        await producer.send_and_wait(
            STORE_EMBEDDING_REQUESTED_TOPIC,
            json.dumps(
                {
                    "store_id": str(create_store_id),
                    "name": "Kafka Integration Store",
                    "description": "Kafka integration test description",
                }
            ).encode(),
        )

        await producer.send_and_wait(
            STORE_EMBEDDING_DELETION_REQUESTED_TOPIC,
            json.dumps(
                {
                    "store_id": str(delete_store_id),
                }
            ).encode(),
        )

        expected_create = CreateStoreEmbeddingInput(
            store_id=create_store_id,
            name="Kafka Integration Store",
            description="Kafka integration test description",
        )
        expected_delete = DeleteStoreEmbeddingInput(
            store_id=delete_store_id,
        )

        await _wait_for(
            lambda: (
                expected_create in create_use_case.inputs
                and expected_delete in delete_use_case.inputs
            )
        )

        assert expected_create in create_use_case.inputs
        assert expected_delete in delete_use_case.inputs

    finally:
        await consumer.stop()
        await producer.stop()


async def _wait_for(
    condition,
    timeout: float = 10.0,
) -> None:
    """Wait briefly for an asynchronous Kafka condition."""
    deadline = asyncio.get_running_loop().time() + timeout

    while not condition():
        if asyncio.get_running_loop().time() >= deadline:
            raise AssertionError(
                "Timed out waiting for Kafka consumer processing."
            )

        await asyncio.sleep(0.1)
