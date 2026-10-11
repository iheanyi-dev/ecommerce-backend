"""Tests for the embedding service gRPC adapter."""

from uuid import uuid4

import grpc
import pytest

from embedding.v1 import embedding_pb2
from infrastructure.knowledge.grpc_client import (
    EmbeddingServiceGRPCClient,
    KnowledgeRetrievalError,
)


class FakeUnaryCall:
    def __init__(self, response=None, error=None):
        self.response = response
        self.error = error
        self.request = None
        self.metadata = None
        self.timeout = None

    async def __call__(self, request, *, metadata, timeout):
        self.request = request
        self.metadata = metadata
        self.timeout = timeout
        if self.error is not None:
            raise self.error
        return self.response


class FakeStub:
    def __init__(self, call):
        self.RetrieveChunks = call


@pytest.mark.asyncio
async def test_retrieve_chunks_sends_store_query_and_service_auth():
    store_id = uuid4()
    call = FakeUnaryCall(
        response=embedding_pb2.RetrieveChunksResponse(
            chunks=[
                embedding_pb2.KnowledgeChunk(
                    content="Delivery takes three to five business days.",
                    source_filename="shipping.pdf",
                    score=0.88,
                )
            ]
        )
    )
    client = EmbeddingServiceGRPCClient(
        target="unused",
        service_name="chat",
        service_secret="test-secret",
        timeout=2.0,
        stub=FakeStub(call),
    )

    chunks = await client.retrieve_chunks(
        store_id=store_id,
        query=" shipping times ",
    )

    assert call.request.store_id == str(store_id)
    assert call.request.query == "shipping times"
    assert dict(call.metadata) == {
        "x-service-name": "chat",
        "x-service-secret": "test-secret",
    }
    assert call.timeout == 2.0
    assert len(chunks) == 1
    assert chunks[0].content == "Delivery takes three to five business days."
    assert chunks[0].source_filename == "shipping.pdf"
    assert chunks[0].score == pytest.approx(0.88)


@pytest.mark.asyncio
async def test_retrieve_chunks_wraps_grpc_errors():
    call = FakeUnaryCall(error=grpc.aio.AioRpcError(
        grpc.StatusCode.UNAVAILABLE,
        (),
        (),
        "unavailable",
    ))
    client = EmbeddingServiceGRPCClient(
        target="unused",
        service_name="chat",
        service_secret="test-secret",
        stub=FakeStub(call),
    )

    with pytest.raises(KnowledgeRetrievalError):
        await client.retrieve_chunks(store_id=uuid4(), query="shipping")


@pytest.mark.asyncio
async def test_retrieve_chunks_rejects_blank_query():
    call = FakeUnaryCall(
        response=embedding_pb2.RetrieveChunksResponse()
    )
    client = EmbeddingServiceGRPCClient(
        target="unused",
        service_name="chat",
        service_secret="test-secret",
        stub=FakeStub(call),
    )

    with pytest.raises(ValueError, match="query"):
        await client.retrieve_chunks(store_id=uuid4(), query="   ")

    assert call.request is None
