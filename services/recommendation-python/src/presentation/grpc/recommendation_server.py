"""gRPC presentation adapter for the Recommendation Service."""

from __future__ import annotations

from uuid import UUID

import grpc

from application.dto.create_product_embedding import CreateProductEmbeddingInput
from application.dto.create_store_embedding import CreateStoreEmbeddingInput
from application.dto.delete_product_embedding import DeleteProductEmbeddingInput
from application.dto.delete_store_embedding import DeleteStoreEmbeddingInput
from application.use_cases.create_product_embedding import CreateProductEmbedding
from application.use_cases.create_store_embedding import CreateStoreEmbedding
from application.use_cases.delete_product_embedding import DeleteProductEmbedding
from application.use_cases.delete_store_embedding import DeleteStoreEmbedding
from infrastructure.proto.recommendation.v1 import recommendation_pb2
from infrastructure.proto.recommendation.v1 import recommendation_pb2_grpc


class RecommendationGrpcServer(
    recommendation_pb2_grpc.RecommendationServiceServicer
):
    """Translate gRPC requests into Recommendation application use cases."""

    def __init__(
        self,
        create_store_embedding: CreateStoreEmbedding,
        delete_store_embedding: DeleteStoreEmbedding,
        create_product_embedding: CreateProductEmbedding,
        delete_product_embedding: DeleteProductEmbedding,
    ) -> None:
        self._create_store_embedding = create_store_embedding
        self._delete_store_embedding = delete_store_embedding
        self._create_product_embedding = create_product_embedding
        self._delete_product_embedding = delete_product_embedding

    def UpsertStoreEmbedding(
        self,
        request: recommendation_pb2.UpsertStoreEmbeddingRequest,
        context: grpc.ServicerContext,
    ) -> recommendation_pb2.UpsertEmbeddingResponse:
        try:
            store_id = UUID(request.store_id)
            self._create_store_embedding.execute(
                CreateStoreEmbeddingInput(
                    store_id=store_id,
                    name=request.name,
                    description=request.description,
                )
            )
        except ValueError as exc:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, str(exc))
        except Exception:
            context.abort(
                grpc.StatusCode.INTERNAL,
                "failed to upsert Store embedding",
            )

        return recommendation_pb2.UpsertEmbeddingResponse()

    def DeleteStoreEmbedding(
        self,
        request: recommendation_pb2.DeleteEmbeddingRequest,
        context: grpc.ServicerContext,
    ) -> recommendation_pb2.DeleteEmbeddingResponse:
        try:
            store_id = UUID(request.entity_id)
            self._delete_store_embedding.execute(
                DeleteStoreEmbeddingInput(store_id=store_id)
            )
        except ValueError as exc:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, str(exc))
        except Exception:
            context.abort(
                grpc.StatusCode.INTERNAL,
                "failed to delete Store embedding",
            )

        return recommendation_pb2.DeleteEmbeddingResponse()

    def UpsertProductEmbedding(
        self,
        request: recommendation_pb2.UpsertProductEmbeddingRequest,
        context: grpc.ServicerContext,
    ) -> recommendation_pb2.UpsertEmbeddingResponse:
        try:
            product_id = UUID(request.product_id)
            self._create_product_embedding.execute(
                CreateProductEmbeddingInput(
                    product_id=product_id,
                    name=request.name,
                    description=request.description,
                )
            )
        except ValueError as exc:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, str(exc))
        except Exception:
            context.abort(
                grpc.StatusCode.INTERNAL,
                "failed to upsert Product embedding",
            )

        return recommendation_pb2.UpsertEmbeddingResponse()

    def DeleteProductEmbedding(
        self,
        request: recommendation_pb2.DeleteEmbeddingRequest,
        context: grpc.ServicerContext,
    ) -> recommendation_pb2.DeleteEmbeddingResponse:
        try:
            product_id = UUID(request.entity_id)
            self._delete_product_embedding.execute(
                DeleteProductEmbeddingInput(product_id=product_id)
            )
        except ValueError as exc:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, str(exc))
        except Exception:
            context.abort(
                grpc.StatusCode.INTERNAL,
                "failed to delete Product embedding",
            )

        return recommendation_pb2.DeleteEmbeddingResponse()
