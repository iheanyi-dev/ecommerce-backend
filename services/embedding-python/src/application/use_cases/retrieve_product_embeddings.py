"""Use case for semantically retrieving Product embeddings."""

from dataclasses import dataclass

from domain.value_objects import EntityID

from ..dto.retrieve_product_embeddings import (
    RetrieveProductEmbeddingsInput,
    RetrieveProductEmbeddingsOutput,
)
from ..ports import EmbeddingModel, Logger, Metrics, ProductEmbeddingRepository


@dataclass(slots=True)
class RetrieveProductEmbeddings:
    embedding_model: EmbeddingModel
    product_embedding_repository: ProductEmbeddingRepository
    logger: Logger
    metrics: Metrics

    def execute(
        self, input: RetrieveProductEmbeddingsInput
    ) -> RetrieveProductEmbeddingsOutput:
        query = input.query.strip()
        if not query:
            raise ValueError("query must not be empty")
        if isinstance(input.top_k, bool) or not isinstance(input.top_k, int) or input.top_k < 1:
            raise ValueError("top_k must be a positive integer")
        if isinstance(input.offset, bool) or not isinstance(input.offset, int) or input.offset < 0:
            raise ValueError("offset must be a non-negative integer")

        store_id = (
            EntityID(input.store_id, "store")
            if input.store_id is not None
            else None
        )

        self.logger.info(
            "embedding.product_embedding.retrieve_started",
            top_k=input.top_k,
            offset=input.offset,
            store_id=str(store_id) if store_id is not None else None,
        )

        embedding, _ = self.embedding_model.generate(query)
        results = tuple(
            self.product_embedding_repository.search(
                query_embedding=embedding,
                top_k=input.top_k,
                offset=input.offset,
                store_id=store_id,
            )
        )

        self.metrics.increment("embedding.product_embeddings.retrieved")
        self.metrics.observe(
            "embedding.product_embeddings.retrieval_results", len(results)
        )
        self.logger.info(
            "embedding.product_embedding.retrieved",
            result_count=len(results),
            top_k=input.top_k,
            offset=input.offset,
            store_id=str(store_id) if store_id is not None else None,
        )

        return RetrieveProductEmbeddingsOutput(results=results)
