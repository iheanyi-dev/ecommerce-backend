"""Authenticated gRPC adapter for internal embedding retrieval."""

from __future__ import annotations

import asyncio
import logging
from pathlib import PurePosixPath
from uuid import UUID

import grpc

from application.dto.retrieve_knowledge import RetrieveKnowledgeInput
from application.dto.retrieve_product_embeddings import RetrieveProductEmbeddingsInput
from application.dto.retrieve_store_embeddings import RetrieveStoreEmbeddingsInput
from application.use_cases.retrieve_knowledge import RetrieveKnowledge
from application.use_cases.retrieve_product_embeddings import RetrieveProductEmbeddings
from application.use_cases.retrieve_store_embeddings import RetrieveStoreEmbeddings
from embedding.v1 import embedding_pb2, embedding_pb2_grpc
from infrastructure.config.settings import Settings

logger = logging.getLogger(__name__)


class EmbeddingService(embedding_pb2_grpc.EmbeddingServiceServicer):
    """Expose authenticated knowledge, product, and store retrieval."""

    def __init__(
        self,
        *,
        retrieve_knowledge: RetrieveKnowledge,
        retrieve_product_embeddings: RetrieveProductEmbeddings,
        retrieve_store_embeddings: RetrieveStoreEmbeddings,
        settings: Settings,
    ) -> None:
        self._retrieve_knowledge = retrieve_knowledge
        self._retrieve_product_embeddings = retrieve_product_embeddings
        self._retrieve_store_embeddings = retrieve_store_embeddings
        self._settings = settings

    async def _authenticate(self, context: grpc.aio.ServicerContext) -> None:
        """Require the configured internal caller identity and secret."""
        metadata = {
            item.key.lower(): item.value
            for item in context.invocation_metadata()
        }
        expected_name = self._settings.grpc.caller_service_name
        expected_secret = self._settings.grpc.caller_service_secret

        if (
            not expected_secret
            or metadata.get("x-service-name") != expected_name
            or metadata.get("x-service-secret") != expected_secret
        ):
            await context.abort(
                grpc.StatusCode.UNAUTHENTICATED,
                "Invalid service credentials",
            )
            raise AssertionError("context.abort must terminate the RPC")

    async def _abort_invalid(self, context, message: str) -> None:
        await context.abort(grpc.StatusCode.INVALID_ARGUMENT, message)
        raise AssertionError("context.abort must terminate the RPC")

    async def RetrieveChunks(
        self,
        request: embedding_pb2.RetrieveChunksRequest,
        context: grpc.aio.ServicerContext,
    ) -> embedding_pb2.RetrieveChunksResponse:
        """Return relevant knowledge chunks scoped to a store."""
        await self._authenticate(context)

        try:
            store_id = UUID(request.store_id)
        except (ValueError, AttributeError):
            await self._abort_invalid(context, "store_id must be a valid UUID")

        if not request.query.strip():
            await self._abort_invalid(context, "query must not be empty")

        try:
            result = await asyncio.to_thread(
                self._retrieve_knowledge.execute,
                RetrieveKnowledgeInput(store_id=store_id, query=request.query),
            )
        except ValueError as error:
            await self._abort_invalid(context, str(error))
        except Exception:
            logger.exception(
                "embedding.grpc.knowledge_retrieval_failed",
                extra={"store_id": request.store_id},
            )
            await context.abort(
                grpc.StatusCode.INTERNAL,
                "Knowledge retrieval failed",
            )
            raise AssertionError("context.abort must terminate the RPC")

        return embedding_pb2.RetrieveChunksResponse(
            chunks=[
                embedding_pb2.KnowledgeChunk(
                    content=item.chunk.content,
                    source_filename=PurePosixPath(
                        item.chunk.storage_reference
                    ).name,
                    score=item.score,
                )
                for item in result.chunks
            ]
        )

    async def RetrieveSimilarProducts(
        self,
        request: embedding_pb2.RetrieveSimilarProductsRequest,
        context: grpc.aio.ServicerContext,
    ) -> embedding_pb2.RetrieveSimilarProductsResponse:
        """Return scored products, optionally restricted to one store."""
        await self._authenticate(context)

        if not request.query.strip():
            await self._abort_invalid(context, "query must not be empty")

        store_id = None
        if request.HasField("store_id"):
            try:
                store_id = UUID(request.store_id)
            except (ValueError, AttributeError):
                await self._abort_invalid(
                    context, "store_id must be a valid UUID"
                )

        try:
            result = await asyncio.to_thread(
                self._retrieve_product_embeddings.execute,
                RetrieveProductEmbeddingsInput(
                    query=request.query,
                    top_k=request.top_k,
                    offset=request.offset,
                    store_id=store_id,
                ),
            )
        except ValueError as error:
            await self._abort_invalid(context, str(error))
        except Exception:
            logger.exception("embedding.grpc.product_retrieval_failed")
            await context.abort(
                grpc.StatusCode.INTERNAL,
                "Product retrieval failed",
            )
            raise AssertionError("context.abort must terminate the RPC")

        return embedding_pb2.RetrieveSimilarProductsResponse(
            products=[
                embedding_pb2.SimilarProduct(
                    product_id=str(item.embedding.id),
                    score=item.score,
                )
                for item in result.results
            ]
        )

    async def RetrieveSimilarStores(
        self,
        request: embedding_pb2.RetrieveSimilarStoresRequest,
        context: grpc.aio.ServicerContext,
    ) -> embedding_pb2.RetrieveSimilarStoresResponse:
        """Return scored stores for a semantic query."""
        await self._authenticate(context)

        if not request.query.strip():
            await self._abort_invalid(context, "query must not be empty")

        try:
            result = await asyncio.to_thread(
                self._retrieve_store_embeddings.execute,
                RetrieveStoreEmbeddingsInput(
                    query=request.query,
                    top_k=request.top_k,
                    offset=request.offset,
                ),
            )
        except ValueError as error:
            await self._abort_invalid(context, str(error))
        except Exception:
            logger.exception("embedding.grpc.store_retrieval_failed")
            await context.abort(
                grpc.StatusCode.INTERNAL,
                "Store retrieval failed",
            )
            raise AssertionError("context.abort must terminate the RPC")

        return embedding_pb2.RetrieveSimilarStoresResponse(
            stores=[
                embedding_pb2.SimilarStore(
                    store_id=str(item.embedding.id),
                    score=item.score,
                )
                for item in result.results
            ]
        )
