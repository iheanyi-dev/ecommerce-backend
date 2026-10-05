"""Use case for creating or replacing a Product embedding."""

from __future__ import annotations

from dataclasses import dataclass

from domain.value_objects import EntityDescription, EntityID, EntityName

from ..dto.create_product_embedding import (
    CreateProductEmbeddingInput,
    CreateProductEmbeddingOutput,
)
from ..ports.embedding_generator import EmbeddingGenerator
from ..ports.logger import Logger
from ..ports.metrics import Metrics
from ..ports.product_embedding_repository import ProductEmbeddingRepository


@dataclass(slots=True)
class CreateProductEmbedding:
    """Generate and persist a Product embedding."""

    embedding_generator: EmbeddingGenerator
    product_embedding_repository: ProductEmbeddingRepository
    logger: Logger
    metrics: Metrics

    def execute(
        self,
        input: CreateProductEmbeddingInput,
    ) -> CreateProductEmbeddingOutput:
        """Validate, generate, and upsert the Product embedding."""

        product_id = EntityID(input.product_id, "product")
        name = EntityName(input.name, "product")
        description = EntityDescription(input.description, "product")

        self.logger.info(
            "recommendation.product_embedding.create_started",
            product_id=str(product_id),
        )

        embedding, model_version = (
            self.embedding_generator.generate_product_embedding(
                name=name,
                description=description,
            )
        )

        self.product_embedding_repository.upsert(
            product_id=product_id,
            name=name,
            description=description,
            embedding=embedding,
            embedding_model_version=model_version,
        )

        self.metrics.increment("recommendation.product_embeddings.created")
        self.logger.info(
            "recommendation.product_embedding.created",
            product_id=str(product_id),
            model_version=str(model_version),
        )

        return CreateProductEmbeddingOutput(
            product_id=input.product_id,
            embedding=embedding,
            embedding_model_version=model_version,
        )
