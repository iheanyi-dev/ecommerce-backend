"""Application use case for deleting indexed knowledge chunks."""

from application.dto.delete_knowledge import DeleteKnowledgeInput
from application.ports.knowledge_chunk_repository import KnowledgeChunkRepository
from application.ports.logger import Logger
from application.ports.metrics import Metrics
from domain.value_objects.entity_id import EntityID


class DeleteKnowledge:
    """Remove all chunks associated with a knowledge document.

    The repository operation should be idempotent: deleting a document whose
    chunks are already absent is still a successful outcome.
    """

    def __init__(
        self,
        knowledge_chunk_repository: KnowledgeChunkRepository,
        logger: Logger,
        metrics: Metrics,
    ) -> None:
        self._knowledge_chunk_repository = knowledge_chunk_repository
        self._logger = logger
        self._metrics = metrics

    def execute(self, request: DeleteKnowledgeInput) -> None:
        """Delete indexed chunks and record success only after persistence succeeds."""
        knowledge_id = EntityID(value=request.knowledge_id, entity_name="knowledge_id")

        self._logger.info(
            "knowledge.deletion.started",
            knowledge_id=str(request.knowledge_id),
        )

        # Do not report success if the repository fails. Let the failure
        # propagate so the caller or message consumer can apply its retry policy.
        self._knowledge_chunk_repository.delete_by_knowledge_id(knowledge_id)

        self._metrics.increment("knowledge.deletion.succeeded")
        self._logger.info(
            "knowledge.deletion.succeeded",
            knowledge_id=str(request.knowledge_id),
        )
