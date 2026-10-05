"""Use case for creating or replacing a Store embedding."""

from __future__ import annotations

from dataclasses import dataclass

from domain.value_objects import EntityDescription, EntityID, EntityName

from ..dto.create_store_embedding import (
    CreateStoreEmbeddingInput,
    CreateStoreEmbeddingOutput,
)
from ..ports.embedding_generator import EmbeddingGenerator
from ..ports.logger import Logger
from ..ports.metrics import Metrics
from ..ports.store_embedding_repository import StoreEmbeddingRepository


@dataclass(slots=True)
class CreateStoreEmbedding:
    """Generate and persist a Store embedding."""

    embedding_generator: EmbeddingGenerator
    store_embedding_repository: StoreEmbeddingRepository
    logger: Logger
    metrics: Metrics

    def execute(
        self,
        input: CreateStoreEmbeddingInput,
    ) -> CreateStoreEmbeddingOutput:
        """Validate, generate, and upsert the Store embedding."""

        store_id = EntityID(input.store_id, "store")
        name = EntityName(input.name, "store")
        description = EntityDescription(input.description, "store")

        self.logger.info(
            "recommendation.store_embedding.create_started",
            store_id=str(store_id),
        )

        embedding, model_version = (
            self.embedding_generator.generate_store_embedding(
                name=name,
                description=description,
            )
        )

        self.store_embedding_repository.upsert(
            store_id=store_id,
            name=name,
            description=description,
            embedding=embedding,
            embedding_model_version=model_version,
        )

        self.metrics.increment("recommendation.store_embeddings.created")
        self.logger.info(
            "recommendation.store_embedding.created",
            store_id=str(store_id),
            model_version=str(model_version),
        )

        return CreateStoreEmbeddingOutput(
            store_id=input.store_id,
            embedding=embedding,
            embedding_model_version=model_version,
        )
