"""Unit tests for the Embedding gRPC adapter."""

from types import SimpleNamespace
from uuid import uuid4

from infrastructure.embedding.grpc_client import EmbeddingGRPCClient


class FakeChannel:
    def __init__(self):
        self.closed = False

    def close(self):
        self.closed = True


class FakeStub:
    def __init__(self):
        self.product_call = None
        self.store_call = None

    def RetrieveSimilarProducts(self, request, **kwargs):
        self.product_call = (request, kwargs)
        return SimpleNamespace(
            products=[
                SimpleNamespace(product_id=str(uuid4()), score=0.91),
            ]
        )

    def RetrieveSimilarStores(self, request, **kwargs):
        self.store_call = (request, kwargs)
        return SimpleNamespace(
            stores=[
                SimpleNamespace(store_id=str(uuid4()), score=0.82),
            ]
        )


def make_client(monkeypatch):
    import infrastructure.embedding.grpc_client as module

    channel = FakeChannel()
    stub = FakeStub()
    monkeypatch.setattr(module.grpc, "insecure_channel", lambda address: channel)
    monkeypatch.setattr(
        module.embedding_pb2_grpc,
        "EmbeddingServiceStub",
        lambda received_channel: stub,
    )
    client = EmbeddingGRPCClient(
        address="localhost:8092",
        service_name="chat",
        service_secret="shared-test-secret",
    )
    return client, channel, stub


def test_product_retrieval_sends_chat_auth_and_pagination(monkeypatch):
    client, _, stub = make_client(monkeypatch)

    results = client.find_similar_products("shoes", top_k=10, offset=20)

    request, kwargs = stub.product_call
    assert request.query == "shoes"
    assert request.top_k == 10
    assert request.offset == 20
    assert ("x-service-name", "chat") in kwargs["metadata"]
    assert ("x-service-secret", "shared-test-secret") in kwargs["metadata"]
    assert len(results) == 1
    assert results[0][1] == 0.91
    client.close()


def test_store_retrieval_sends_pagination(monkeypatch):
    client, _, stub = make_client(monkeypatch)

    results = client.find_similar_stores("fashion", top_k=10, offset=10)

    request, kwargs = stub.store_call
    assert request.query == "fashion"
    assert request.top_k == 10
    assert request.offset == 10
    assert kwargs["timeout"] == 10.0
    assert len(results) == 1
    assert results[0][1] == 0.82
    client.close()


def test_close_releases_channel(monkeypatch):
    client, channel, _ = make_client(monkeypatch)

    client.close()

    assert channel.closed
