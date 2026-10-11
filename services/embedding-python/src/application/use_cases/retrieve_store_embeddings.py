"""Use case for semantically retrieving Store embeddings."""

from dataclasses import dataclass

from ..dto.retrieve_store_embeddings import (
    RetrieveStoreEmbeddingsInput,
    RetrieveStoreEmbeddingsOutput,
)
from ..ports import EmbeddingModel, Logger, Metrics, StoreEmbeddingRepository


@dataclass(slots=True)
class RetrieveStoreEmbeddings:
    embedding_model: EmbeddingModel
    store_embedding_repository: StoreEmbeddingRepository
    logger: Logger
    metrics: Metrics

    def execute(
        self, input: RetrieveStoreEmbeddingsInput
    ) -> RetrieveStoreEmbeddingsOutput:
        query = input.query.strip()
        if not query:
            raise ValueError("query must not be empty")
        if isinstance(input.top_k, bool) or not isinstance(input.top_k, int) or input.top_k < 1:
            raise ValueError("top_k must be a positive integer")
        if isinstance(input.offset, bool) or not isinstance(input.offset, int) or input.offset < 0:
            raise ValueError("offset must be a non-negative integer")

        self.logger.info(
            "embedding.store_embedding.retrieve_started",
            top_k=input.top_k,
            offset=input.offset,
        )

        embedding, _ = self.embedding_model.generate(query)
        results = tuple(
            self.store_embedding_repository.search(
                query_embedding=embedding,
                top_k=input.top_k,
                offset=input.offset,
            )
        )

        self.metrics.increment("embedding.store_embeddings.retrieved")
        self.metrics.observe(
            "embedding.store_embeddings.retrieval_results", len(results)
        )
        self.logger.info(
            "embedding.store_embedding.retrieved",
            result_count=len(results),
            top_k=input.top_k,
            offset=input.offset,
        )

        return RetrieveStoreEmbeddingsOutput(results=results)
