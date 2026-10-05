"""Tests for the Recommendation gRPC presentation adapter."""

from __future__ import annotations

from uuid import uuid4

import grpc
import pytest

from application.dto.create_product_embedding import CreateProductEmbeddingOutput
from application.dto.create_store_embedding import CreateStoreEmbeddingOutput
from application.use_cases.create_product_embedding import CreateProductEmbedding
from application.use_cases.create_store_embedding import CreateStoreEmbedding
from application.use_cases.delete_product_embedding import DeleteProductEmbedding
from application.use_cases.delete_store_embedding import DeleteStoreEmbedding
from domain.value_objects import Embedding, EmbeddingModelVersion
from infrastructure.proto.recommendation.v1 import recommendation_pb2
from presentation.grpc.recommendation_server import RecommendationGrpcServer


class FakeContext:
    """Minimal gRPC context for direct handler tests."""

    def __init__(self) -> None:
        self.code = None
        self.details = None

    def abort(self, code: grpc.StatusCode, details: str) -> None:
        self.code = code
        self.details = details
        raise RuntimeError(details)


class FakeCreateStoreEmbedding:
    def __init__(self) -> None:
        self.input = None

    def execute(self, input):
        self.input = input
        return CreateStoreEmbeddingOutput(
            store_id=input.store_id,
            embedding=Embedding((0.1, 0.2)),
            embedding_model_version=EmbeddingModelVersion("test@1"),
        )


class FakeDeleteStoreEmbedding:
    def __init__(self) -> None:
        self.input = None

    def execute(self, input):
        self.input = input


class FakeCreateProductEmbedding:
    def __init__(self) -> None:
        self.input = None

    def execute(self, input):
        self.input = input
        return CreateProductEmbeddingOutput(
            product_id=input.product_id,
            embedding=Embedding((0.1, 0.2)),
            embedding_model_version=EmbeddingModelVersion("test@1"),
        )


class FakeDeleteProductEmbedding:
    def __init__(self) -> None:
        self.input = None

    def execute(self, input):
        self.input = input


@pytest.fixture
def server():
    create_store = FakeCreateStoreEmbedding()
    delete_store = FakeDeleteStoreEmbedding()
    create_product = FakeCreateProductEmbedding()
    delete_product = FakeDeleteProductEmbedding()

    return (
        RecommendationGrpcServer(
            create_store_embedding=create_store,
            delete_store_embedding=delete_store,
            create_product_embedding=create_product,
            delete_product_embedding=delete_product,
        ),
        create_store,
        delete_store,
        create_product,
        delete_product,
    )


def test_upsert_store_embedding_maps_request_to_use_case(server):
    handler, create_store, *_ = server
    store_id = uuid4()

    response = handler.UpsertStoreEmbedding(
        recommendation_pb2.UpsertStoreEmbeddingRequest(
            store_id=str(store_id),
            name="Store",
            description="Description",
        ),
        FakeContext(),
    )

    assert response == recommendation_pb2.UpsertEmbeddingResponse()
    assert create_store.input.store_id == store_id
    assert create_store.input.name == "Store"
    assert create_store.input.description == "Description"


def test_delete_store_embedding_maps_request_to_use_case(server):
    handler, _, delete_store, *_ = server
    store_id = uuid4()

    response = handler.DeleteStoreEmbedding(
        recommendation_pb2.DeleteEmbeddingRequest(
            entity_id=str(store_id),
        ),
        FakeContext(),
    )

    assert response == recommendation_pb2.DeleteEmbeddingResponse()
    assert delete_store.input.store_id == store_id


def test_upsert_product_embedding_maps_request_to_use_case(server):
    handler, _, _, create_product, _ = server
    product_id = uuid4()

    response = handler.UpsertProductEmbedding(
        recommendation_pb2.UpsertProductEmbeddingRequest(
            product_id=str(product_id),
            name="Product",
            description="Description",
        ),
        FakeContext(),
    )

    assert response == recommendation_pb2.UpsertEmbeddingResponse()
    assert create_product.input.product_id == product_id
    assert create_product.input.name == "Product"
    assert create_product.input.description == "Description"


def test_delete_product_embedding_maps_request_to_use_case(server):
    handler, *_, delete_product = server
    product_id = uuid4()

    response = handler.DeleteProductEmbedding(
        recommendation_pb2.DeleteEmbeddingRequest(
            entity_id=str(product_id),
        ),
        FakeContext(),
    )

    assert response == recommendation_pb2.DeleteEmbeddingResponse()
    assert delete_product.input.product_id == product_id


def test_invalid_store_id_returns_invalid_argument(server):
    handler, *_ = server
    context = FakeContext()

    with pytest.raises(RuntimeError):
        handler.UpsertStoreEmbedding(
            recommendation_pb2.UpsertStoreEmbeddingRequest(
                store_id="not-a-uuid",
                name="Store",
                description="Description",
            ),
            context,
        )

    assert context.code == grpc.StatusCode.INVALID_ARGUMENT


def test_invalid_product_id_returns_invalid_argument(server):
    handler, *_ = server
    context = FakeContext()

    with pytest.raises(RuntimeError):
        handler.DeleteProductEmbedding(
            recommendation_pb2.DeleteEmbeddingRequest(
                entity_id="not-a-uuid",
            ),
            context,
        )

    assert context.code == grpc.StatusCode.INVALID_ARGUMENT
