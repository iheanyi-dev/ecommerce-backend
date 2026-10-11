"""gRPC adapter for the embedding service's knowledge retrieval contract."""

from uuid import UUID

import grpc

from embedding.v1 import embedding_pb2, embedding_pb2_grpc
from infrastructure.agent.ports.knowledge_retrieval import KnowledgeChunk


class KnowledgeRetrievalError(RuntimeError):
    """Raised when the embedding service cannot retrieve knowledge."""


class EmbeddingServiceGRPCClient:
    """Call the embedding service without depending on its implementation."""

    def __init__(
        self,
        *,
        target: str,
        service_name: str,
        service_secret: str,
        timeout: float = 5.0,
        channel: grpc.aio.Channel | None = None,
        stub: embedding_pb2_grpc.EmbeddingServiceStub | None = None,
    ) -> None:
        if not target.strip() and channel is None and stub is None:
            raise ValueError("Embedding service target must not be empty")
        if not service_name.strip():
            raise ValueError("Embedding service name must not be empty")
        if not service_secret:
            raise ValueError("Embedding service secret must not be empty")
        if timeout <= 0:
            raise ValueError("Embedding service timeout must be positive")

        self._service_name = service_name
        self._service_secret = service_secret
        self._timeout = timeout
        self._owned_channel = (
            grpc.aio.insecure_channel(target)
            if channel is None and stub is None
            else None
        )
        active_channel = channel or self._owned_channel
        self._stub = stub or embedding_pb2_grpc.EmbeddingServiceStub(
            active_channel
        )

    async def retrieve_chunks(
        self,
        *,
        store_id: UUID,
        query: str,
    ) -> list[KnowledgeChunk]:
        """Retrieve relevant chunks, authenticating the service call."""
        normalized_query = query.strip()
        if not normalized_query:
            raise ValueError("Knowledge query must not be empty")

        request = embedding_pb2.RetrieveChunksRequest(
            store_id=str(store_id),
            query=normalized_query,
        )
        metadata = (
            ("x-service-name", self._service_name),
            ("x-service-secret", self._service_secret),
        )

        try:
            response = await self._stub.RetrieveChunks(
                request,
                metadata=metadata,
                timeout=self._timeout,
            )
        except grpc.RpcError as exc:
            raise KnowledgeRetrievalError(
                "Knowledge retrieval service request failed"
            ) from exc

        return [
            KnowledgeChunk(
                content=chunk.content,
                source_filename=chunk.source_filename,
                score=chunk.score,
            )
            for chunk in response.chunks
        ]

    async def aclose(self) -> None:
        """Close the channel when this adapter owns it."""
        if self._owned_channel is not None:
            await self._owned_channel.close()
