from uuid import uuid4

import pytest

from domain.errors import InvalidEntityIDError
from domain.value_objects import EntityID

from application.dto import DeleteStoreEmbeddingInput
from application.use_cases import DeleteStoreEmbedding
from test.support.observability import StubLogger, StubMetrics


class StubStoreEmbeddingRepository:
    def __init__(self) -> None:
        self.calls: list[EntityID] = []

    def delete(
        self,
        store_id: EntityID,
    ) -> None:
        self.calls.append(store_id)


def test_delete_store_embedding_converts_input_to_domain_id() -> None:
    store_id = uuid4()

    repository = StubStoreEmbeddingRepository()

    use_case = DeleteStoreEmbedding(
        store_embedding_repository=repository,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    use_case.execute(
        DeleteStoreEmbeddingInput(
            store_id=store_id,
        )
    )

    assert repository.calls == [
        EntityID(store_id, "store"),
    ]


def test_delete_store_embedding_deletes_requested_embedding() -> None:
    first_store_id = uuid4()
    second_store_id = uuid4()

    repository = StubStoreEmbeddingRepository()

    use_case = DeleteStoreEmbedding(
        store_embedding_repository=repository,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    use_case.execute(DeleteStoreEmbeddingInput(store_id=first_store_id))
    use_case.execute(DeleteStoreEmbeddingInput(store_id=second_store_id))

    assert repository.calls == [
        EntityID(first_store_id, "store"),
        EntityID(second_store_id, "store"),
    ]


def test_delete_store_embedding_rejects_invalid_id() -> None:
    repository = StubStoreEmbeddingRepository()

    use_case = DeleteStoreEmbedding(
        store_embedding_repository=repository,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    with pytest.raises(InvalidEntityIDError):
        use_case.execute(
            DeleteStoreEmbeddingInput(
                store_id="not-a-uuid",  # type: ignore[arg-type]
            )
        )

    assert repository.calls == []
