"""Reliable Kafka consumption with explicit offset commits."""

from __future__ import annotations

import asyncio
from collections.abc import Callable
from typing import Any

from aiokafka import AIOKafkaConsumer


class KafkaEventConsumer:
    """Consume sequentially; commit each record only after successful handling.

    Handler failures deliberately escape the loop. The worker must be restarted
    by its process supervisor, allowing the uncommitted event to be redelivered.
    """

    def __init__(
        self,
        *,
        brokers: str,
        group_id: str,
        topics: tuple[str, ...],
        handler: Callable[[str, bytes], None],
        logger: Any,
        client_id: str = "embedding-service",
    ) -> None:
        if not brokers.strip():
            raise ValueError("Kafka bootstrap servers must not be empty.")
        if not group_id.strip():
            raise ValueError("Kafka consumer group ID must not be empty.")
        if not topics:
            raise ValueError("At least one Kafka topic is required.")

        self._brokers = brokers
        self._group_id = group_id
        self._topics = topics
        self._handler = handler
        self._logger = logger
        self._client_id = client_id
        self._consumer: AIOKafkaConsumer | None = None

    async def run(self) -> None:
        consumer = AIOKafkaConsumer(
            *self._topics,
            bootstrap_servers=self._brokers.split(","),
            group_id=self._group_id,
            client_id=self._client_id,
            enable_auto_commit=False,
            auto_offset_reset="earliest",
            # Allow lengthy knowledge-ingestion events without premature
            # group eviction. Model loading happens before this worker starts.
            max_poll_interval_ms=30 * 60 * 1000,
        )
        self._consumer = consumer
        await consumer.start()

        try:
            self._logger.info(
                "embedding.kafka_consumer.started",
                group_id=self._group_id,
                topics=list(self._topics),
            )

            async for record in consumer:
                # Do not catch handler exceptions here: committing or continuing
                # after a failed event could lose the event or advance past it.
                # Keep the event loop responsive for Kafka heartbeats while
                # synchronous embedding and document processing run.
                await asyncio.to_thread(
                    self._handler, record.topic, record.value
                )
                await consumer.commit()

                self._logger.info(
                    "embedding.kafka_event.processed",
                    topic=record.topic,
                    partition=record.partition,
                    offset=record.offset,
                )
        finally:
            await consumer.stop()
            self._consumer = None
            self._logger.info("embedding.kafka_consumer.stopped")
