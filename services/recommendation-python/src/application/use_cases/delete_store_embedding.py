"""Use case for deleting a Store embedding."""

from __future__ import annotations

from dataclasses import dataclass

from domain.value_objects import EntityID

from ..dto.delete_store_embedding import DeleteStoreEmbeddingInput
from ..ports.store_embedding_repository import StoreEmbeddingRepository


@dataclass(slots=True)
class DeleteStoreEmbedding:
    """Delete the embedding representation associated with a Store."""

    store_embedding_repository: StoreEmbeddingRepository

    def execute(
        self,
        input: DeleteStoreEmbeddingInput,
    ) -> None:
        """Validate the Store ID and delete its embedding."""

        store_id = EntityID(input.store_id, "store")

        self.store_embedding_repository.delete(
            store_id=store_id,
        )
