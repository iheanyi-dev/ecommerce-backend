"""Tests for Store and Product semantic retrieval use cases."""

from dataclasses import dataclass, field
from uuid import uuid4

import pytest

from application.dto import (
    RetrieveProductEmbeddingsInput,
    RetrieveStoreEmbeddingsInput,
)
from application.use_cases import (
    RetrieveProductEmbeddings,
    RetrieveStoreEmbeddings,
)
from domain.entities import ProductEmbedding, StoreEmbedding
from domain.value_objects import Embedding, EmbeddingModelVersion


@dataclass
class FakeModel:
    embedding: Embedding = field(
        default_factory=lambda: Embedding.from_values([0.1, 0.2, 0.3])
    )
    texts: list[str] = field(default_factory=list)

    def generate(self, text):
        self.texts.append(text)
        return self.embedding, EmbeddingModelVersion("test-model@revision-1")


@dataclass
class FakeLogger:
    events: list = field(default_factory=list)

    def info(self, event, **fields):
        self.events.append((event, fields))

    def warning(self, event, **fields):
        self.events.append((event, fields))

    def error(self, event, **fields):
        self.events.append((event, fields))


@dataclass
class FakeMetrics:
    counters: dict = field(default_factory=dict)
    observations: list = field(default_factory=list)

    def increment(self, name, value=1):
        self.counters[name] = self.counters.get(name, 0) + value

    def observe(self, name, value):
        self.observations.append((name, value))


@dataclass
class FakeStoreRepository:
    results: list[StoreEmbedding] = field(default_factory=list)
    searches: list[dict] = field(default_factory=list)

    def search(self, **kwargs):
        self.searches.append(kwargs)
        return self.results


@dataclass
class FakeProductRepository:
    results: list[ProductEmbedding] = field(default_factory=list)
    searches: list[dict] = field(default_factory=list)

    def search(self, **kwargs):
        self.searches.append(kwargs)
        return self.results


def dependencies():
    return FakeModel(), FakeLogger(), FakeMetrics()


def test_retrieve_stores_generates_query_embedding_and_passes_pagination():
    model, logger, metrics = dependencies()
    repository = FakeStoreRepository()
    result = RetrieveStoreEmbeddings(
        model, repository, logger, metrics
    ).execute(RetrieveStoreEmbeddingsInput("  home decor  ", top_k=5, offset=10))

    assert model.texts == ["home decor"]
    assert repository.searches == [{
        "query_embedding": model.embedding,
        "top_k": 5,
        "offset": 10,
    }]
    assert result.results == ()
    assert metrics.counters == {"embedding.store_embeddings.retrieved": 1}
    assert metrics.observations == [
        ("embedding.store_embeddings.retrieval_results", 0)
    ]


def test_retrieve_products_passes_optional_store_filter():
    model, logger, metrics = dependencies()
    repository = FakeProductRepository()
    store_id = uuid4()

    RetrieveProductEmbeddings(
        model, repository, logger, metrics
    ).execute(
        RetrieveProductEmbeddingsInput(
            query="cotton shirt",
            top_k=7,
            offset=2,
            store_id=store_id,
        )
    )

    search = repository.searches[0]
    assert search["query_embedding"] == model.embedding
    assert search["top_k"] == 7
    assert search["offset"] == 2
    assert search["store_id"].value == store_id


def test_retrieve_products_without_store_filter_searches_all_stores():
    model, logger, metrics = dependencies()
    repository = FakeProductRepository()

    RetrieveProductEmbeddings(
        model, repository, logger, metrics
    ).execute(RetrieveProductEmbeddingsInput(query="shirt"))

    assert repository.searches[0]["store_id"] is None


@pytest.mark.parametrize("query", ["", "   "])
def test_retrieval_rejects_empty_query(query):
    model, logger, metrics = dependencies()
    repository = FakeStoreRepository()

    with pytest.raises(ValueError, match="query must not be empty"):
        RetrieveStoreEmbeddings(
            model, repository, logger, metrics
        ).execute(RetrieveStoreEmbeddingsInput(query=query))

    assert model.texts == []
    assert repository.searches == []


@pytest.mark.parametrize("top_k", [0, -1, True, 1.5])
def test_store_retrieval_rejects_invalid_top_k(top_k):
    model, logger, metrics = dependencies()

    with pytest.raises(ValueError, match="top_k"):
        RetrieveStoreEmbeddings(
            model, FakeStoreRepository(), logger, metrics
        ).execute(RetrieveStoreEmbeddingsInput(query="decor", top_k=top_k))


@pytest.mark.parametrize("offset", [-1, True, 1.5])
def test_product_retrieval_rejects_invalid_offset(offset):
    model, logger, metrics = dependencies()

    with pytest.raises(ValueError, match="offset"):
        RetrieveProductEmbeddings(
            model, FakeProductRepository(), logger, metrics
        ).execute(RetrieveProductEmbeddingsInput(query="shirt", offset=offset))


def test_repository_failure_propagates_without_success_metric():
    model, logger, metrics = dependencies()

    class FailingRepository:
        def search(self, **kwargs):
            raise RuntimeError("Qdrant unavailable")

    with pytest.raises(RuntimeError, match="Qdrant unavailable"):
        RetrieveStoreEmbeddings(
            model, FailingRepository(), logger, metrics
        ).execute(RetrieveStoreEmbeddingsInput(query="decor"))

    assert metrics.counters == {}
    assert metrics.observations == []
