"""Use case for deleting a Product embedding."""

from __future__ import annotations

from dataclasses import dataclass

from domain.value_objects import EntityID

from ..dto.delete_product_embedding import DeleteProductEmbeddingInput
from ..ports.logger import Logger
from ..ports.metrics import Metrics
from ..ports.product_embedding_repository import ProductEmbeddingRepository


@dataclass(slots=True)
class DeleteProductEmbedding:
    """Delete the embedding representation associated with a Product."""

    product_embedding_repository: ProductEmbeddingRepository
    logger: Logger
    metrics: Metrics

    def execute(self, input: DeleteProductEmbeddingInput) -> None:
        """Validate the Product ID and delete its embedding."""

        product_id = EntityID(input.product_id, "product")

        self.product_embedding_repository.delete(product_id=product_id)

        self.metrics.increment("recommendation.product_embeddings.deleted")
        self.logger.info(
            "recommendation.product_embedding.deleted",
            product_id=str(product_id),
        )
