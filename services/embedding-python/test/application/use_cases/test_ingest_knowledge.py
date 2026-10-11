"""Tests for the Knowledge ingestion use case."""

from dataclasses import dataclass, field
from uuid import UUID, uuid4

import pytest

from application.dto import IngestKnowledgeInput
from application.use_cases import IngestKnowledge
from domain.entities import KnowledgeChunk
from domain.value_objects import Embedding, EmbeddingModelVersion


@dataclass
class FakeDocumentReader:
    text: str = "First paragraph. Second paragraph."
    references: list = field(default_factory=list)

    def read(self, storage_reference):
        self.references.append(storage_reference)
        return self.text


@dataclass
class FakeChunker:
    chunks: list[str] = field(
        default_factory=lambda: ["First paragraph.", "Second paragraph."]
    )
    texts: list[str] = field(default_factory=list)

    def chunk(self, text):
        self.texts.append(text)
        return self.chunks


@dataclass
class FakeEmbeddingModel:
    embedding: Embedding = field(
        default_factory=lambda: Embedding.from_values([0.1, 0.2, 0.3])
    )
    model_version: EmbeddingModelVersion = field(
        default_factory=lambda: EmbeddingModelVersion("test-model@revision-1")
    )
    texts: list[str] = field(default_factory=list)

    def generate(self, text):
        self.texts.append(text)
        return self.embedding, self.model_version


@dataclass
class FakeKnowledgeRepository:
    replacements: list = field(default_factory=list)
    error: Exception | None = None

    def replace_for_knowledge(self, knowledge_id, chunks):
        if self.error:
            raise self.error
        self.replacements.append((knowledge_id, list(chunks)))


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


def dependencies():
    reader = FakeDocumentReader()
    chunker = FakeChunker()
    model = FakeEmbeddingModel()
    repository = FakeKnowledgeRepository()
    logger = FakeLogger()
    metrics = FakeMetrics()
    use_case = IngestKnowledge(
        reader, chunker, model, repository, logger, metrics
    )
    return use_case, reader, chunker, model, repository, logger, metrics


def test_ingest_knowledge_chunks_embeds_and_replaces_document_chunks():
    use_case, reader, chunker, model, repository, logger, metrics = dependencies()
    knowledge_id, store_id = uuid4(), uuid4()
    reference = "knowledge/" + str(store_id) + "/guide.txt"

    result = use_case.execute(
        IngestKnowledgeInput(
            knowledge_id=knowledge_id,
            store_id=store_id,
            storage_reference=reference,
        )
    )

    assert reader.references[0].value == reference
    assert chunker.texts == [reader.text]
    assert model.texts == ["First paragraph.", "Second paragraph."]
    assert len(repository.replacements) == 1

    saved_id, chunks = repository.replacements[0]
    assert saved_id.value == knowledge_id
    assert len(chunks) == 2
    assert all(isinstance(chunk, KnowledgeChunk) for chunk in chunks)
    assert [chunk.chunk_index for chunk in chunks] == [0, 1]
    assert [chunk.content for chunk in chunks] == [
        "First paragraph.",
        "Second paragraph.",
    ]
    assert all(chunk.knowledge_id == knowledge_id for chunk in chunks)
    assert all(chunk.store_id == store_id for chunk in chunks)
    assert all(chunk.storage_reference == reference for chunk in chunks)
    assert all(chunk.embedding == model.embedding for chunk in chunks)
    assert all(
        chunk.embedding_model_version == model.model_version for chunk in chunks
    )
    assert result.knowledge_id == knowledge_id
    assert result.store_id == store_id
    assert result.chunk_count == 2
    assert result.embedding_model_versions == (
        model.model_version,
        model.model_version,
    )
    assert metrics.counters == {"embedding.knowledge_documents.ingested": 1}
    assert metrics.observations == [
        ("embedding.knowledge_chunks.ingested", 2)
    ]
    assert logger.events[-1][0] == "embedding.knowledge.ingested"


def test_ingest_rejects_empty_document_without_replacing_existing_chunks():
    use_case, reader, _, model, repository, _, metrics = dependencies()
    reader.text = "  "

    with pytest.raises(ValueError, match="no extractable text"):
        use_case.execute(
            IngestKnowledgeInput(uuid4(), uuid4(), "knowledge/store/guide.txt")
        )

    assert model.texts == []
    assert repository.replacements == []
    assert metrics.counters == {}


def test_ingest_rejects_empty_chunk_list_without_replacing_existing_chunks():
    use_case, _, chunker, model, repository, _, metrics = dependencies()
    chunker.chunks = []

    with pytest.raises(ValueError, match="no chunks"):
        use_case.execute(
            IngestKnowledgeInput(uuid4(), uuid4(), "knowledge/store/guide.txt")
        )

    assert model.texts == []
    assert repository.replacements == []
    assert metrics.counters == {}


def test_ingest_rejects_empty_chunk_content_without_replacing_existing_chunks():
    use_case, _, chunker, model, repository, _, metrics = dependencies()
    chunker.chunks = ["valid chunk", "   "]

    with pytest.raises(ValueError, match="index 1"):
        use_case.execute(
            IngestKnowledgeInput(uuid4(), uuid4(), "knowledge/store/guide.txt")
        )

    assert model.texts == ["valid chunk"]
    assert repository.replacements == []
    assert metrics.counters == {}


def test_repository_failure_propagates_without_success_metrics():
    use_case, _, _, _, repository, logger, metrics = dependencies()
    repository.error = RuntimeError("Qdrant unavailable")

    with pytest.raises(RuntimeError, match="Qdrant unavailable"):
        use_case.execute(
            IngestKnowledgeInput(uuid4(), uuid4(), "knowledge/store/guide.txt")
        )

    assert metrics.counters == {}
    assert metrics.observations == []
    assert logger.events[-1][0] == "embedding.knowledge.ingest_started"
