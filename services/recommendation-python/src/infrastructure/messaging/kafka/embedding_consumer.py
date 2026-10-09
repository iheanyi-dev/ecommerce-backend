"""Kafka consumer for Store embedding lifecycle events."""

from __future__ import annotations

import asyncio
import json
import logging
from typing import Any
from uuid import UUID

from aiokafka import AIOKafkaConsumer
from aiokafka.structs import ConsumerRecord, OffsetAndMetadata, TopicPartition

from application.dto.create_store_embedding import CreateStoreEmbeddingInput
from application.dto.delete_store_embedding import DeleteStoreEmbeddingInput
from application.use_cases.create_store_embedding import CreateStoreEmbedding
from application.use_cases.delete_store_embedding import DeleteStoreEmbedding


STORE_EMBEDDING_REQUESTED_TOPIC = "store.embedding.requested"
STORE_EMBEDDING_DELETION_REQUESTED_TOPIC = (
    "store.embedding.deletion.requested"
)


class KafkaEmbeddingConsumer:
    """Consume Store embedding lifecycle events from Kafka."""

    def __init__(
        self,
        *,
        bootstrap_servers: str,
        group_id: str,
        create_store_embedding: CreateStoreEmbedding,
        delete_store_embedding: DeleteStoreEmbedding,
    ) -> None:
        self._bootstrap_servers = bootstrap_servers
        self._group_id = group_id
        self._consumer: AIOKafkaConsumer | None = None
        self._create_store_embedding = create_store_embedding
        self._delete_store_embedding = delete_store_embedding
        self._task: asyncio.Task[None] | None = None
        self._logger = logging.getLogger(__name__)

    async def start(self) -> None:
        """Create and start the Kafka consumer."""

        self._consumer = AIOKafkaConsumer(
            STORE_EMBEDDING_REQUESTED_TOPIC,
            STORE_EMBEDDING_DELETION_REQUESTED_TOPIC,
            bootstrap_servers=self._bootstrap_servers,
            group_id=self._group_id,
            enable_auto_commit=False,
            auto_offset_reset="earliest",
        )

        await self._consumer.start()
        self._task = asyncio.create_task(self._consume())
        self._logger.info("Recommendation Kafka consumer started")

    async def stop(self) -> None:
        """Stop the consumer and its background task."""

        if self._task is not None:
            self._task.cancel()

            try:
                await self._task
            except asyncio.CancelledError:
                pass

            self._task = None

        if self._consumer is not None:
            await self._consumer.stop()
            self._consumer = None

        self._logger.info("Recommendation Kafka consumer stopped")

    async def _consume(self) -> None:
        """Poll Kafka continuously and process each partition independently."""

        if self._consumer is None:
            raise RuntimeError("Kafka consumer has not been started")

        queues: dict[
            TopicPartition,
            asyncio.Queue[ConsumerRecord],
        ] = {}
        workers: dict[
            TopicPartition,
            asyncio.Task[None],
        ] = {}

        try:
            while True:
                records = await self._consumer.getmany(
                    timeout_ms=1000,
                    max_records=10,
                )

                for partition, partition_records in records.items():
                    if partition not in queues:
                        queue: asyncio.Queue[ConsumerRecord] = asyncio.Queue(
                            maxsize=1
                        )
                        queues[partition] = queue
                        workers[partition] = asyncio.create_task(
                            self._process_partition(
                                partition=partition,
                                queue=queue,
                            )
                        )

                    queue = queues[partition]

                    for record in partition_records:
                        await queue.put(record)

        except asyncio.CancelledError:
            for worker in workers.values():
                worker.cancel()

            for worker in workers.values():
                try:
                    await worker
                except asyncio.CancelledError:
                    pass

            raise

    async def _process_partition(
        self,
        *,
        partition: TopicPartition,
        queue: asyncio.Queue[ConsumerRecord],
    ) -> None:
        """Process one Kafka partition sequentially."""

        if self._consumer is None:
            raise RuntimeError("Kafka consumer has not been started")

        while True:
            record = await queue.get()

            try:
                await self.handle_record(record)

                await self._consumer.commit(
                    {
                        partition: OffsetAndMetadata(
                            record.offset + 1,
                            "",
                        )
                    }
                )

            except Exception:
                self._logger.exception(
                    "Failed to process Recommendation Kafka event",
                    extra={
                        "topic": record.topic,
                        "partition": record.partition,
                        "offset": record.offset,
                    },
                )

                await asyncio.sleep(1)

            finally:
                queue.task_done()

    async def handle_record(self, record: ConsumerRecord) -> None:
        """Process one Kafka record without blocking the event loop."""

        payload = self._decode_payload(record.value)

        if record.topic == STORE_EMBEDDING_REQUESTED_TOPIC:
            await asyncio.to_thread(
                self._handle_embedding_requested,
                payload,
            )
            return

        if record.topic == STORE_EMBEDDING_DELETION_REQUESTED_TOPIC:
            await asyncio.to_thread(
                self._handle_embedding_deletion_requested,
                payload,
            )
            return

        raise ValueError(f"Unsupported Kafka topic: {record.topic}")

    @staticmethod
    def _decode_payload(value: bytes) -> dict[str, Any]:
        """Decode and validate a JSON event payload."""

        try:
            payload = json.loads(value)
        except (UnicodeDecodeError, json.JSONDecodeError) as exc:
            raise ValueError("Kafka event contains invalid JSON") from exc

        if not isinstance(payload, dict):
            raise ValueError("Kafka event payload must be a JSON object")

        return payload

    def _handle_embedding_requested(
        self,
        payload: dict[str, Any],
    ) -> None:
        """Create or replace the Store embedding."""

        store_id = self._parse_store_id(payload)

        name = payload.get("name")
        description = payload.get("description")

        if not isinstance(name, str):
            raise ValueError("Store embedding event name must be a string")

        if not isinstance(description, str):
            raise ValueError(
                "Store embedding event description must be a string"
            )

        self._create_store_embedding.execute(
            CreateStoreEmbeddingInput(
                store_id=store_id,
                name=name,
                description=description,
            )
        )

    def _handle_embedding_deletion_requested(
        self,
        payload: dict[str, Any],
    ) -> None:
        """Delete the Store embedding."""

        store_id = self._parse_store_id(payload)

        self._delete_store_embedding.execute(
            DeleteStoreEmbeddingInput(store_id=store_id)
        )

    @staticmethod
    def _parse_store_id(payload: dict[str, Any]) -> UUID:
        """Parse the Store UUID from an event payload."""

        store_id = payload.get("store_id")

        if not isinstance(store_id, str):
            raise ValueError("Store embedding event store_id must be a string")

        try:
            return UUID(store_id)
        except ValueError as exc:
            raise ValueError(
                "Store embedding event store_id must be a valid UUID"
            ) from exc
