"""Tests for Store and Product embedding application use cases."""

from dataclasses import dataclass, field
from uuid import uuid4

import pytest

from domain.entities import ProductEmbedding, StoreEmbedding
from domain.value_objects import Embedding, EmbeddingModelVersion
from application.dto import (
    CreateProductEmbeddingInput,
    CreateStoreEmbeddingInput,
    DeleteProductEmbeddingInput,
    DeleteStoreEmbeddingInput,
)
from application.use_cases import (
    CreateProductEmbedding,
    CreateStoreEmbedding,
    DeleteProductEmbedding,
    DeleteStoreEmbedding,
)


@dataclass
class FakeEmbeddingModel:
    embedding: Embedding = field(
        default_factory=lambda: Embedding.from_values([0.1, 0.2, 0.3])
    )
    model_version: EmbeddingModelVersion = field(
        default_factory=lambda: EmbeddingModelVersion("test-model@revision-1")
    )
    texts: list[str] = field(default_factory=list)

    def generate(self, text: str):
        self.texts.append(text)
        return self.embedding, self.model_version


@dataclass
class FakeLogger:
    events: list[tuple[str, dict]] = field(default_factory=list)

    def info(self, event: str, **fields) -> None:
        self.events.append((event, fields))

    def warning(self, event: str, **fields) -> None:
        self.events.append((event, fields))

    def error(self, event: str, **fields) -> None:
        self.events.append((event, fields))


@dataclass
class FakeMetrics:
    counters: dict[str, int] = field(default_factory=dict)

    def increment(self, name: str, value: int = 1) -> None:
        self.counters[name] = self.counters.get(name, 0) + value

    def observe(self, name: str, value: float) -> None:
        pass


@dataclass
class FakeStoreRepository:
    upserted: list[StoreEmbedding] = field(default_factory=list)
    deleted: list[object] = field(default_factory=list)

    def upsert(self, embedding: StoreEmbedding) -> None:
        self.upserted.append(embedding)

    def delete(self, store_id) -> None:
        self.deleted.append(store_id)


@dataclass
class FakeProductRepository:
    upserted: list[ProductEmbedding] = field(default_factory=list)
    deleted: list[object] = field(default_factory=list)
    deleted_store_ids: list[object] = field(default_factory=list)

    def upsert(self, embedding: ProductEmbedding) -> None:
        self.upserted.append(embedding)

    def delete(self, product_id) -> None:
        self.deleted.append(product_id)

    def delete_by_store_id(self, store_id) -> None:
        self.deleted_store_ids.append(store_id)


def dependencies():
    return (
        FakeEmbeddingModel(),
        FakeStoreRepository(),
        FakeProductRepository(),
        FakeLogger(),
        FakeMetrics(),
    )


def test_create_store_embedding_generates_and_persists_labeled_text():
    model, stores, _, logger, metrics = dependencies()
    store_id = uuid4()

    result = CreateStoreEmbedding(
        model, stores, logger, metrics
    ).execute(
        CreateStoreEmbeddingInput(
            store_id=store_id,
            name="  Home Goods  ",
            description="  Useful products  ",
        )
    )

    assert model.texts == [
        "Store name: Home Goods\nStore description: Useful products"
    ]
    assert len(stores.upserted) == 1
    entity = stores.upserted[0]
    assert entity.id == store_id
    assert entity.name == "Home Goods"
    assert entity.description == "Useful products"
    assert entity.embedding == model.embedding
    assert entity.embedding_model_version == model.model_version
    assert result.store_id == store_id
    assert metrics.counters == {"embedding.store_embeddings.created": 1}
    assert logger.events[-1][0] == "embedding.store_embedding.created"


def test_create_product_embedding_generates_and_persists_labeled_text():
    model, _, products, logger, metrics = dependencies()
    product_id, store_id = uuid4(), uuid4()

    result = CreateProductEmbedding(
        model, products, logger, metrics
    ).execute(
        CreateProductEmbeddingInput(
            product_id=product_id,
            store_id=store_id,
            name="  Cotton Shirt  ",
            description="  Breathable fabric  ",
        )
    )

    assert model.texts == [
        "Product name: Cotton Shirt\n"
        "Product description: Breathable fabric"
    ]
    assert len(products.upserted) == 1
    entity = products.upserted[0]
    assert entity.id == product_id
    assert entity.store_id == store_id
    assert entity.name == "Cotton Shirt"
    assert entity.description == "Breathable fabric"
    assert entity.embedding == model.embedding
    assert entity.embedding_model_version == model.model_version
    assert result.product_id == product_id
    assert metrics.counters == {"embedding.product_embeddings.created": 1}
    assert logger.events[-1][0] == "embedding.product_embedding.created"


def test_delete_product_embedding_deletes_and_records_success():
    _, _, products, logger, metrics = dependencies()
    product_id = uuid4()

    DeleteProductEmbedding(
        products, logger, metrics
    ).execute(DeleteProductEmbeddingInput(product_id=product_id))

    assert len(products.deleted) == 1
    assert products.deleted[0].value == product_id
    assert metrics.counters == {"embedding.product_embeddings.deleted": 1}
    assert logger.events[-1][0] == "embedding.product_embedding.deleted"


def test_delete_store_embedding_deletes_products_before_store():
    _, stores, products, logger, metrics = dependencies()
    store_id = uuid4()

    DeleteStoreEmbedding(
        stores, products, logger, metrics
    ).execute(DeleteStoreEmbeddingInput(store_id=store_id))

    assert len(products.deleted_store_ids) == 1
    assert products.deleted_store_ids[0].value == store_id
    assert len(stores.deleted) == 1
    assert stores.deleted[0].value == store_id
    assert metrics.counters == {"embedding.store_embeddings.deleted": 1}
    assert logger.events[-1][0] == "embedding.store_embedding.deleted"


def test_store_deletion_does_not_delete_store_vector_if_product_cleanup_fails():
    _, stores, products, logger, metrics = dependencies()

    def fail_delete_by_store_id(store_id):
        raise RuntimeError("Qdrant unavailable")

    products.delete_by_store_id = fail_delete_by_store_id

    with pytest.raises(RuntimeError, match="Qdrant unavailable"):
        DeleteStoreEmbedding(
            stores, products, logger, metrics
        ).execute(DeleteStoreEmbeddingInput(store_id=uuid4()))

    assert stores.deleted == []
    assert metrics.counters == {}
    assert logger.events[-1][0] == "embedding.store_embedding.delete_started"
