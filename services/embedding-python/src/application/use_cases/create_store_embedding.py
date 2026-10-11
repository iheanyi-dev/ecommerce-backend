"""Use case for creating or replacing a Store embedding."""

from dataclasses import dataclass

from domain.entities import StoreEmbedding
from domain.value_objects import EntityDescription, EntityID, EntityName

from ..dto.create_store_embedding import (
    CreateStoreEmbeddingInput,
    CreateStoreEmbeddingOutput,
)
from ..ports import EmbeddingModel, Logger, Metrics, StoreEmbeddingRepository


@dataclass(slots=True)
class CreateStoreEmbedding:
    """Generate and persist a Store vector representation."""

    embedding_model: EmbeddingModel
    store_embedding_repository: StoreEmbeddingRepository
    logger: Logger
    metrics: Metrics

    def execute(
        self, input: CreateStoreEmbeddingInput
    ) -> CreateStoreEmbeddingOutput:
        """Validate, embed, and upsert a Store."""

        store_id = EntityID(input.store_id, "store")
        name = EntityName(input.name.strip(), "store")
        description = EntityDescription(input.description.strip(), "store")

        self.logger.info(
            "embedding.store_embedding.create_started",
            store_id=str(store_id),
        )

        text = (
            f"Store name: {input.name.strip()}\n"
            f"Store description: {input.description.strip()}"
        )
        embedding, model_version = self.embedding_model.generate(text)

        entity = StoreEmbedding(
            id=input.store_id,
            name=name.value,
            description=description.value,
            embedding=embedding,
            embedding_model_version=model_version,
        )
        self.store_embedding_repository.upsert(entity)

        self.metrics.increment("embedding.store_embeddings.created")
        self.logger.info(
            "embedding.store_embedding.created",
            store_id=str(store_id),
            model_version=str(model_version),
        )

        return CreateStoreEmbeddingOutput(
            store_id=input.store_id,
            embedding=embedding,
            embedding_model_version=model_version,
        )
