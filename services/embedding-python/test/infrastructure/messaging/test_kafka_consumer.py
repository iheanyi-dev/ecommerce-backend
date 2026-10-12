"""Unit tests for manual Kafka offset commits."""

import asyncio
from types import SimpleNamespace
from unittest.mock import AsyncMock

import pytest

import infrastructure.messaging.kafka_consumer as module
from infrastructure.messaging.kafka_consumer import KafkaEventConsumer


class FakeConsumer:
    def __init__(self, *topics, **kwargs):
        self.topics = topics
        self.kwargs = kwargs
        self.commits = 0
        self.stopped = False
        self.records = [
            SimpleNamespace(
                topic=topics[0],
                value=b'{"ok": true}',
                partition=0,
                offset=0,
            )
        ]

    async def start(self):
        return None

    async def stop(self):
        self.stopped = True

    async def commit(self):
        self.commits += 1

    def __aiter__(self):
        async def iterate():
            for record in self.records:
                yield record
        return iterate()


def test_consumer_commits_only_after_handler_succeeds(monkeypatch):
    fake = FakeConsumer
    monkeypatch.setattr(module, "AIOKafkaConsumer", fake)

    handled = []
    logger = SimpleNamespace(info=lambda *_args, **_kwargs: None)

    def handler(topic, value):
        handled.append((topic, value))

    consumer = KafkaEventConsumer(
        brokers="localhost:19092",
        group_id="embedding-test",
        topics=("store.embedding.requested",),
        handler=handler,
        logger=logger,
    )

    asyncio.run(consumer.run())

    assert handled == [("store.embedding.requested", b'{"ok": true}')]
    # Inspect the instance created by the patched factory.
    assert consumer._consumer is None


def test_handler_failure_stops_consumer_and_does_not_commit(monkeypatch):
    instances = []

    def factory(*topics, **kwargs):
        instance = FakeConsumer(*topics, **kwargs)
        instances.append(instance)
        return instance

    monkeypatch.setattr(module, "AIOKafkaConsumer", factory)

    def handler(_topic, _value):
        raise RuntimeError("processing failed")

    consumer = KafkaEventConsumer(
        brokers="localhost:19092",
        group_id="embedding-test",
        topics=("store.embedding.requested",),
        handler=handler,
        logger=SimpleNamespace(info=lambda *_args, **_kwargs: None),
    )

    with pytest.raises(RuntimeError, match="processing failed"):
        asyncio.run(consumer.run())

    assert instances[0].commits == 0
    assert instances[0].stopped is True


def test_consumer_rejects_empty_configuration():
    logger = SimpleNamespace(info=lambda *_args, **_kwargs: None)

    with pytest.raises(ValueError):
        KafkaEventConsumer(
            brokers=" ",
            group_id="group",
            topics=("topic",),
            handler=AsyncMock(),
            logger=logger,
        )
