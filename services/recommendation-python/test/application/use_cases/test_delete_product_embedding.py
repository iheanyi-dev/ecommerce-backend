from uuid import uuid4

import pytest

from domain.errors import InvalidEntityIDError
from domain.value_objects import EntityID

from application.dto import DeleteProductEmbeddingInput
from application.use_cases import DeleteProductEmbedding
from test.support.observability import StubLogger, StubMetrics


class StubProductEmbeddingRepository:
    def __init__(self) -> None:
        self.calls: list[EntityID] = []

    def delete(
        self,
        product_id: EntityID,
    ) -> None:
        self.calls.append(product_id)


def test_delete_product_embedding_converts_input_to_domain_id() -> None:
    product_id = uuid4()

    repository = StubProductEmbeddingRepository()

    use_case = DeleteProductEmbedding(
        product_embedding_repository=repository,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    use_case.execute(
        DeleteProductEmbeddingInput(
            product_id=product_id,
        )
    )

    assert repository.calls == [
        EntityID(product_id, "product"),
    ]


def test_delete_product_embedding_deletes_requested_embedding() -> None:
    first_product_id = uuid4()
    second_product_id = uuid4()

    repository = StubProductEmbeddingRepository()

    use_case = DeleteProductEmbedding(
        product_embedding_repository=repository,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    use_case.execute(DeleteProductEmbeddingInput(product_id=first_product_id))
    use_case.execute(DeleteProductEmbeddingInput(product_id=second_product_id))

    assert repository.calls == [
        EntityID(first_product_id, "product"),
        EntityID(second_product_id, "product"),
    ]


def test_delete_product_embedding_rejects_invalid_id() -> None:
    repository = StubProductEmbeddingRepository()

    use_case = DeleteProductEmbedding(
        product_embedding_repository=repository,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    with pytest.raises(InvalidEntityIDError):
        use_case.execute(
            DeleteProductEmbeddingInput(
                product_id="not-a-uuid",  # type: ignore[arg-type]
            )
        )

    assert repository.calls == []
