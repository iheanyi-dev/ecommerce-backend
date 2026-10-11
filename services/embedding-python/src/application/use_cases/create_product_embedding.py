"""Use case for creating or replacing a Product embedding."""

from dataclasses import dataclass

from domain.entities import ProductEmbedding
from domain.value_objects import EntityDescription, EntityID, EntityName

from ..dto.create_product_embedding import (
    CreateProductEmbeddingInput,
    CreateProductEmbeddingOutput,
)
from ..ports import EmbeddingModel, Logger, Metrics, ProductEmbeddingRepository


@dataclass(slots=True)
class CreateProductEmbedding:
    """Generate and persist a Product vector representation."""

    embedding_model: EmbeddingModel
    product_embedding_repository: ProductEmbeddingRepository
    logger: Logger
    metrics: Metrics

    def execute(
        self, input: CreateProductEmbeddingInput
    ) -> CreateProductEmbeddingOutput:
        """Validate, embed, and upsert a Product."""

        product_id = EntityID(input.product_id, "product")
        store_id = EntityID(input.store_id, "store")
        name = EntityName(input.name.strip(), "product")
        description = EntityDescription(input.description.strip(), "product")

        self.logger.info(
            "embedding.product_embedding.create_started",
            product_id=str(product_id),
            store_id=str(store_id),
        )

        text = (
            f"Product name: {input.name.strip()}\n"
            f"Product description: {input.description.strip()}"
        )
        embedding, model_version = self.embedding_model.generate(text)

        entity = ProductEmbedding(
            id=input.product_id,
            store_id=input.store_id,
            name=name.value,
            description=description.value,
            embedding=embedding,
            embedding_model_version=model_version,
        )
        self.product_embedding_repository.upsert(entity)

        self.metrics.increment("embedding.product_embeddings.created")
        self.logger.info(
            "embedding.product_embedding.created",
            product_id=str(product_id),
            store_id=str(store_id),
            model_version=str(model_version),
        )

        return CreateProductEmbeddingOutput(
            product_id=input.product_id,
            embedding=embedding,
            embedding_model_version=model_version,
        )
