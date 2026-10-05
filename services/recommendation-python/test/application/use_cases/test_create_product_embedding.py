from test.support.observability import StubLogger, StubMetrics
from uuid import uuid4

import pytest

from domain.errors import (
    EmptyEntityDescriptionError,
    EmptyEntityNameError,
)
from domain.value_objects import (
    Embedding,
    EmbeddingModelVersion,
    EntityDescription,
    EntityID,
    EntityName,
)

from application.dto import CreateProductEmbeddingInput
from application.use_cases import CreateProductEmbedding


class StubEmbeddingGenerator:
    def __init__(self) -> None:
        self.calls: list[tuple[EntityName, EntityDescription]] = []

    def generate_product_embedding(
        self,
        name: EntityName,
        description: EntityDescription,
    ) -> tuple[Embedding, EmbeddingModelVersion]:
        self.calls.append((name, description))

        return (
            Embedding.from_values((0.1, 0.2, 0.3)),
            EmbeddingModelVersion("product-model-v1"),
        )


class StubProductEmbeddingRepository:
    def __init__(self) -> None:
        self.calls: list[dict[str, object]] = []

    def upsert(
        self,
        product_id: EntityID,
        name: EntityName,
        description: EntityDescription,
        embedding: Embedding,
        embedding_model_version: EmbeddingModelVersion,
    ) -> None:
        self.calls.append(
            {
                "product_id": product_id,
                "name": name,
                "description": description,
                "embedding": embedding,
                "embedding_model_version": embedding_model_version,
            }
        )


def test_create_product_embedding_converts_input_to_domain_values() -> None:
    product_id = uuid4()
    name = "Premium Wireless Headphones"
    description = "Noise-cancelling wireless headphones."

    generator = StubEmbeddingGenerator()
    repository = StubProductEmbeddingRepository()

    use_case = CreateProductEmbedding(
        embedding_generator=generator,
        product_embedding_repository=repository,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    output = use_case.execute(
        CreateProductEmbeddingInput(
            product_id=product_id,
            name=name,
            description=description,
        )
    )

    assert generator.calls == [
        (
            EntityName(name, "product"),
            EntityDescription(description, "product"),
        )
    ]

    assert repository.calls[0]["product_id"] == EntityID(
        product_id,
        "product",
    )
    assert repository.calls[0]["name"] == EntityName(name, "product")
    assert repository.calls[0]["description"] == EntityDescription(
        description,
        "product",
    )
    assert repository.calls[0]["embedding"] == Embedding.from_values(
        (0.1, 0.2, 0.3)
    )
    assert repository.calls[0]["embedding_model_version"] == (
        EmbeddingModelVersion("product-model-v1")
    )

    assert output.product_id == product_id
    assert output.embedding == Embedding.from_values((0.1, 0.2, 0.3))
    assert output.embedding_model_version == EmbeddingModelVersion(
        "product-model-v1"
    )


def test_create_product_embedding_upserts_on_reprocessing() -> None:
    product_id = uuid4()

    generator = StubEmbeddingGenerator()
    repository = StubProductEmbeddingRepository()

    use_case = CreateProductEmbedding(
        embedding_generator=generator,
        product_embedding_repository=repository,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    input_data = CreateProductEmbeddingInput(
        product_id=product_id,
        name="Product",
        description="Product description.",
    )

    use_case.execute(input_data)
    use_case.execute(input_data)

    assert len(generator.calls) == 2
    assert len(repository.calls) == 2


def test_create_product_embedding_rejects_empty_name_through_domain() -> None:
    generator = StubEmbeddingGenerator()
    repository = StubProductEmbeddingRepository()

    use_case = CreateProductEmbedding(
        embedding_generator=generator,
        product_embedding_repository=repository,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    with pytest.raises(EmptyEntityNameError):
        use_case.execute(
            CreateProductEmbeddingInput(
                product_id=uuid4(),
                name="   ",
                description="Valid product description.",
            )
        )

    assert generator.calls == []
    assert repository.calls == []


def test_create_product_embedding_rejects_empty_description_through_domain() -> None:
    generator = StubEmbeddingGenerator()
    repository = StubProductEmbeddingRepository()

    use_case = CreateProductEmbedding(
        embedding_generator=generator,
        product_embedding_repository=repository,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    with pytest.raises(EmptyEntityDescriptionError):
        use_case.execute(
            CreateProductEmbeddingInput(
                product_id=uuid4(),
                name="Valid Product",
                description="   ",
            )
        )

    assert generator.calls == []
    assert repository.calls == []
