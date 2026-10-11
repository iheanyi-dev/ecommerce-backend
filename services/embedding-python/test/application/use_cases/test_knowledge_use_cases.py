"""Unit tests for knowledge deletion and retrieval application use cases."""

from uuid import uuid4

import pytest

from application.dto.delete_knowledge import DeleteKnowledgeInput
from application.dto.retrieve_knowledge import RetrieveKnowledgeInput
from application.use_cases.delete_knowledge import DeleteKnowledge
from application.use_cases.retrieve_knowledge import RetrieveKnowledge


class FakeLogger:
    """Capture log events without requiring a real logging backend."""

    def __init__(self):
        self.events = []

    def info(self, event, **fields):
        self.events.append(("info", event, fields))

    def warning(self, event, **fields):
        self.events.append(("warning", event, fields))

    def error(self, event, **fields):
        self.events.append(("error", event, fields))


class FakeMetrics:
    """Capture metrics so tests can verify success and failure behavior."""

    def __init__(self):
        self.increments = []
        self.observations = []

    def increment(self, name, value=1):
        self.increments.append((name, value))

    def observe(self, name, value):
        self.observations.append((name, value))


class FakeKnowledgeChunkRepository:
    """In-memory repository fake with injectable persistence failures."""

    def __init__(self):
        self.deleted_ids = []
        self.search_calls = []
        self.results = []
        self.delete_error = None
        self.search_error = None

    def delete_by_knowledge_id(self, knowledge_id):
        if self.delete_error:
            raise self.delete_error
        self.deleted_ids.append(knowledge_id)

    def search(self, *, query_embedding, top_k, offset, store_id):
        if self.search_error:
            raise self.search_error

        self.search_calls.append(
            {
                "query_embedding": query_embedding,
                "top_k": top_k,
                "offset": offset,
                "store_id": store_id,
            }
        )
        return self.results


class FakeEmbeddingModel:
    """Fake model that records query text and can simulate inference failure."""

    def __init__(self):
        self.calls = []
        self.embedding = object()
        self.model_version = object()
        self.error = None

    def generate(self, text):
        if self.error:
            raise self.error

        self.calls.append(text)
        return self.embedding, self.model_version


def make_delete_use_case(repository=None, logger=None, metrics=None):
    """Build the use case with controllable test doubles."""
    repository = repository or FakeKnowledgeChunkRepository()
    logger = logger or FakeLogger()
    metrics = metrics or FakeMetrics()
    return DeleteKnowledge(repository, logger, metrics), repository, logger, metrics


def make_retrieve_use_case(repository=None, model=None, logger=None, metrics=None):
    """Build retrieval with test doubles for each application boundary."""
    repository = repository or FakeKnowledgeChunkRepository()
    model = model or FakeEmbeddingModel()
    logger = logger or FakeLogger()
    metrics = metrics or FakeMetrics()
    return (
        RetrieveKnowledge(repository, model, logger, metrics),
        repository,
        model,
        logger,
        metrics,
    )


def test_delete_knowledge_deletes_chunks_and_records_success():
    use_case, repository, logger, metrics = make_delete_use_case()
    knowledge_id = uuid4()

    result = use_case.execute(DeleteKnowledgeInput(knowledge_id=knowledge_id))

    assert result is None
    assert len(repository.deleted_ids) == 1
    assert repository.deleted_ids[0].value == knowledge_id
    assert ("knowledge.deletion.succeeded", 1) in metrics.increments
    assert any(event[1] == "knowledge.deletion.succeeded" for event in logger.events)


def test_delete_knowledge_propagates_repository_failure_without_success_metrics():
    repository = FakeKnowledgeChunkRepository()
    repository.delete_error = RuntimeError("repository unavailable")
    use_case, _, _, metrics = make_delete_use_case(repository=repository)

    with pytest.raises(RuntimeError, match="repository unavailable"):
        use_case.execute(DeleteKnowledgeInput(knowledge_id=uuid4()))

    assert not any(
        name == "knowledge.deletion.succeeded"
        for name, _ in metrics.increments
    )


def test_retrieve_knowledge_uses_default_top_k_and_offset():
    use_case, repository, model, _, metrics = make_retrieve_use_case()
    store_id = uuid4()
    expected_chunk = object()
    repository.results = [expected_chunk]

    result = use_case.execute(
        RetrieveKnowledgeInput(store_id=store_id, query="  how to return an item  ")
    )

    assert model.calls == ["how to return an item"]
    assert len(repository.search_calls) == 1
    call = repository.search_calls[0]
    assert call["query_embedding"] is model.embedding
    assert call["top_k"] == 10
    assert call["offset"] == 0
    assert call["store_id"].value == store_id
    assert result.store_id == store_id
    assert result.chunks == (expected_chunk,)
    assert ("knowledge.retrieval.succeeded", 1) in metrics.increments
    assert ("knowledge.retrieval.result_count", 1) in metrics.observations


def test_retrieve_knowledge_supports_explicit_pagination():
    use_case, repository, _, _, _ = make_retrieve_use_case()

    use_case.execute(
        RetrieveKnowledgeInput(
            store_id=uuid4(),
            query="returns policy",
            top_k=4,
            offset=8,
        )
    )

    call = repository.search_calls[0]
    assert call["top_k"] == 4
    assert call["offset"] == 8


@pytest.mark.parametrize("query", ["", "   ", "\n\t"])
def test_retrieve_knowledge_rejects_empty_query_before_embedding(query):
    use_case, repository, model, _, metrics = make_retrieve_use_case()

    with pytest.raises(ValueError, match="query must not be empty"):
        use_case.execute(RetrieveKnowledgeInput(store_id=uuid4(), query=query))

    assert model.calls == []
    assert repository.search_calls == []
    assert metrics.increments == []


@pytest.mark.parametrize("top_k", [0, -1, True, 1.5])
def test_retrieve_knowledge_rejects_invalid_top_k(top_k):
    use_case, repository, model, _, _ = make_retrieve_use_case()

    with pytest.raises(ValueError, match="top_k"):
        use_case.execute(
            RetrieveKnowledgeInput(
                store_id=uuid4(),
                query="shipping",
                top_k=top_k,
            )
        )

    assert model.calls == []
    assert repository.search_calls == []


@pytest.mark.parametrize("offset", [-1, True, 1.5])
def test_retrieve_knowledge_rejects_invalid_offset(offset):
    use_case, repository, model, _, _ = make_retrieve_use_case()

    with pytest.raises(ValueError, match="offset"):
        use_case.execute(
            RetrieveKnowledgeInput(
                store_id=uuid4(),
                query="shipping",
                offset=offset,
            )
        )

    assert model.calls == []
    assert repository.search_calls == []


def test_retrieve_knowledge_propagates_embedding_failure_without_search():
    model = FakeEmbeddingModel()
    model.error = RuntimeError("embedding unavailable")
    use_case, repository, _, _, metrics = make_retrieve_use_case(model=model)

    with pytest.raises(RuntimeError, match="embedding unavailable"):
        use_case.execute(RetrieveKnowledgeInput(store_id=uuid4(), query="shipping"))

    assert repository.search_calls == []
    assert metrics.increments == []


def test_retrieve_knowledge_propagates_repository_failure_without_success_metrics():
    repository = FakeKnowledgeChunkRepository()
    repository.search_error = RuntimeError("search unavailable")
    use_case, _, _, _, metrics = make_retrieve_use_case(repository=repository)

    with pytest.raises(RuntimeError, match="search unavailable"):
        use_case.execute(RetrieveKnowledgeInput(store_id=uuid4(), query="shipping"))

    assert metrics.increments == []
