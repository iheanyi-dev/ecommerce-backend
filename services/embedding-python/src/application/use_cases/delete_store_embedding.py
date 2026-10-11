"""Use case for deleting a Store and its associated Product embeddings."""

from dataclasses import dataclass

from domain.value_objects import EntityID

from ..dto.delete_store_embedding import DeleteStoreEmbeddingInput
from ..ports import (
    Logger,
    Metrics,
    ProductEmbeddingRepository,
    StoreEmbeddingRepository,
)


@dataclass(slots=True)
class DeleteStoreEmbedding:
    """Remove a Store vector and all Product vectors belonging to it."""

    store_embedding_repository: StoreEmbeddingRepository
    product_embedding_repository: ProductEmbeddingRepository
    logger: Logger
    metrics: Metrics

    def execute(self, input: DeleteStoreEmbeddingInput) -> None:
        """Delete associated Product vectors and then the Store vector.

        Both repository operations must be idempotent. If either operation
        fails, the exception propagates so the event consumer can retry.
        """

        store_id = EntityID(input.store_id, "store")
        self.logger.info(
            "embedding.store_embedding.delete_started",
            store_id=str(store_id),
        )

        self.product_embedding_repository.delete_by_store_id(store_id)
        self.store_embedding_repository.delete(store_id)

        self.metrics.increment("embedding.store_embeddings.deleted")
        self.logger.info(
            "embedding.store_embedding.deleted",
            store_id=str(store_id),
        )
