"""Use case for creating or replacing a Store embedding."""

from __future__ import annotations

from dataclasses import dataclass

from domain.value_objects import (
    EntityDescription,
    EntityID,
    EntityName,
)

from ..dto.create_store_embedding import (
    CreateStoreEmbeddingInput,
    CreateStoreEmbeddingOutput,
)
from ..ports.embedding_generator import EmbeddingGenerator
from ..ports.store_embedding_repository import StoreEmbeddingRepository


@dataclass(slots=True)
class CreateStoreEmbedding:
    """Generate and persist a Store embedding.

    External input remains primitive at the application boundary.
    The use case converts that input into validated domain values
    before invoking its application ports.
    """

    embedding_generator: EmbeddingGenerator
    store_embedding_repository: StoreEmbeddingRepository

    def execute(
        self,
        input: CreateStoreEmbeddingInput,
    ) -> CreateStoreEmbeddingOutput:
        """Validate, generate, and upsert the Store embedding."""

        store_id = EntityID(input.store_id, "store")
        name = EntityName(input.name, "store")
        description = EntityDescription(input.description, "store")

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

        return CreateStoreEmbeddingOutput(
            store_id=input.store_id,
            embedding=embedding,
            embedding_model_version=model_version,
        )
