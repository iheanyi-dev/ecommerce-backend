"""Tests for the CreateStoreEmbedding use case."""

from uuid import uuid4

from domain.value_objects import (
    Embedding,
    EmbeddingModelVersion,
    EntityDescription,
    EntityID,
    EntityName,
)

from application.dto import CreateStoreEmbeddingInput
from application.use_cases import CreateStoreEmbedding
from test.support.observability import StubLogger, StubMetrics
from test.support.observability import StubLogger, StubMetrics


class StubEmbeddingGenerator:
    """Test double for the embedding generator."""

    def __init__(self) -> None:
        self.calls: list[tuple[EntityName, EntityDescription]] = []

    def generate_store_embedding(
        self,
        name: EntityName,
        description: EntityDescription,
    ) -> tuple[Embedding, EmbeddingModelVersion]:
        self.calls.append((name, description))

        return (
            Embedding.from_values((0.1, 0.2, 0.3)),
            EmbeddingModelVersion("store-model-v1"),
        )


class StubStoreEmbeddingRepository:
    """Test double for the Store embedding repository."""

    def __init__(self) -> None:
        self.calls: list[dict[str, object]] = []

    def upsert(
        self,
        store_id: EntityID,
        name: EntityName,
        description: EntityDescription,
        embedding: Embedding,
        embedding_model_version: EmbeddingModelVersion,
    ) -> None:
        self.calls.append(
            {
                "store_id": store_id,
                "name": name,
                "description": description,
                "embedding": embedding,
                "embedding_model_version": embedding_model_version,
            }
        )


def test_create_store_embedding_converts_input_to_domain_values() -> None:
    store_id = uuid4()
    generator = StubEmbeddingGenerator()
    repository = StubStoreEmbeddingRepository()

    use_case = CreateStoreEmbedding(
        embedding_generator=generator,
        store_embedding_repository=repository,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    output = use_case.execute(
        CreateStoreEmbeddingInput(
            store_id=store_id,
            name="Acme Store",
            description="A store selling quality products.",
        )
    )

    assert generator.calls == [
        (
            EntityName("Acme Store", "store"),
            EntityDescription(
                "A store selling quality products.",
                "store",
            ),
        )
    ]

    assert len(repository.calls) == 1
    assert repository.calls[0]["store_id"] == EntityID(store_id, "store")
    assert repository.calls[0]["name"] == EntityName("Acme Store", "store")
    assert repository.calls[0]["description"] == EntityDescription(
        "A store selling quality products.",
        "store",
    )

    assert output.store_id == store_id


def test_create_store_embedding_upserts_on_reprocessing() -> None:
    store_id = uuid4()
    generator = StubEmbeddingGenerator()
    repository = StubStoreEmbeddingRepository()

    use_case = CreateStoreEmbedding(
        embedding_generator=generator,
        store_embedding_repository=repository,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    input_data = CreateStoreEmbeddingInput(
        store_id=store_id,
        name="Updated Store",
        description="Updated description.",
    )

    use_case.execute(input_data)
    use_case.execute(input_data)

    assert len(generator.calls) == 2
    assert len(repository.calls) == 2


def test_create_store_embedding_rejects_empty_name_through_domain() -> None:
    use_case = CreateStoreEmbedding(
        embedding_generator=StubEmbeddingGenerator(),
        store_embedding_repository=StubStoreEmbeddingRepository(),
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    from domain.errors import EmptyEntityNameError
    import pytest

    with pytest.raises(EmptyEntityNameError):
        use_case.execute(
            CreateStoreEmbeddingInput(
                store_id=uuid4(),
                name="   ",
                description="Valid description.",
            )
        )


def test_create_store_embedding_rejects_empty_description_through_domain() -> None:
    use_case = CreateStoreEmbedding(
        embedding_generator=StubEmbeddingGenerator(),
        store_embedding_repository=StubStoreEmbeddingRepository(),
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    from domain.errors import EmptyEntityDescriptionError
    import pytest

    with pytest.raises(EmptyEntityDescriptionError):
        use_case.execute(
            CreateStoreEmbeddingInput(
                store_id=uuid4(),
                name="Valid Store",
                description="   ",
            )
        )
