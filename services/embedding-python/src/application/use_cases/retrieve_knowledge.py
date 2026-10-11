"""Application use case for tenant-scoped semantic knowledge retrieval."""

from application.dto.retrieve_knowledge import (
    RetrieveKnowledgeInput,
    RetrieveKnowledgeOutput,
)
from application.ports.embedding_model import EmbeddingModel
from application.ports.knowledge_chunk_repository import KnowledgeChunkRepository
from application.ports.logger import Logger
from application.ports.metrics import Metrics
from domain.value_objects.entity_id import EntityID


class RetrieveKnowledge:
    """Embed a text query and retrieve its most relevant knowledge chunks."""

    DEFAULT_TOP_K = 10
    DEFAULT_OFFSET = 0

    def __init__(
        self,
        knowledge_chunk_repository: KnowledgeChunkRepository,
        embedding_model: EmbeddingModel,
        logger: Logger,
        metrics: Metrics,
    ) -> None:
        self._knowledge_chunk_repository = knowledge_chunk_repository
        self._embedding_model = embedding_model
        self._logger = logger
        self._metrics = metrics

    def execute(
        self,
        request: RetrieveKnowledgeInput,
    ) -> RetrieveKnowledgeOutput:
        """Validate the request, embed its query, and perform scoped retrieval."""
        query = request.query.strip()
        if not query:
            raise ValueError("query must not be empty")

        # Reject invalid pagination before spending resources generating an
        # embedding or making a repository request.
        top_k = request.top_k
        offset = request.offset

        if isinstance(top_k, bool) or not isinstance(top_k, int) or top_k <= 0:
            raise ValueError("top_k must be a positive integer")
        if isinstance(offset, bool) or not isinstance(offset, int) or offset < 0:
            raise ValueError("offset must be a non-negative integer")

        store_id = EntityID(value=request.store_id, entity_name="store_id")

        self._logger.info(
            "knowledge.retrieval.started",
            store_id=str(request.store_id),
            top_k=top_k,
            offset=offset,
        )

        # Generate the query vector using the same model contract used to
        # create indexed chunk embeddings.
        query_embedding, _model_version = self._embedding_model.generate(query)

        # The repository must apply store_id as a tenant filter and perform
        # similarity ranking and pagination in the vector database.
        chunks = tuple(
            self._knowledge_chunk_repository.search(
                query_embedding=query_embedding,
                top_k=top_k,
                offset=offset,
                store_id=store_id,
            )
        )

        # Success telemetry is emitted only after embedding and retrieval both
        # succeed; failures propagate to the caller.
        self._metrics.increment("knowledge.retrieval.succeeded")
        self._metrics.observe("knowledge.retrieval.result_count", len(chunks))
        self._logger.info(
            "knowledge.retrieval.succeeded",
            store_id=str(request.store_id),
            result_count=len(chunks),
        )

        return RetrieveKnowledgeOutput(
            store_id=request.store_id,
            chunks=chunks,
        )
