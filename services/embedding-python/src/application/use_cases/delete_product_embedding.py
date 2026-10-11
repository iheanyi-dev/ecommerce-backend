"""Use case for deleting a Product embedding."""

from dataclasses import dataclass

from domain.value_objects import EntityID

from ..dto.delete_product_embedding import DeleteProductEmbeddingInput
from ..ports import Logger, Metrics, ProductEmbeddingRepository


@dataclass(slots=True)
class DeleteProductEmbedding:
    """Delete the vector representation associated with a Product."""

    product_embedding_repository: ProductEmbeddingRepository
    logger: Logger
    metrics: Metrics

    def execute(self, input: DeleteProductEmbeddingInput) -> None:
        """Validate the Product ID and delete its embedding."""

        product_id = EntityID(input.product_id, "product")
        self.logger.info(
            "embedding.product_embedding.delete_started",
            product_id=str(product_id),
        )

        self.product_embedding_repository.delete(product_id)

        self.metrics.increment("embedding.product_embeddings.deleted")
        self.logger.info(
            "embedding.product_embedding.deleted",
            product_id=str(product_id),
        )
