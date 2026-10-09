"""FastAPI routes for external Recommendation inference."""

from __future__ import annotations

import asyncio
from uuid import UUID

from fastapi import APIRouter, Depends, HTTPException, Query, status

from application.dto.recommend_products import RecommendProductsInput
from application.dto.recommend_stores import RecommendStoresInput
from application.use_cases.recommend_products import RecommendProducts
from application.use_cases.recommend_stores import RecommendStores

from .schemas import (
    ProductRecommendationQuery,
    RecommendationItem,
    RecommendationResponse,
    StoreRecommendationQuery,
)


def create_router(
    recommend_stores: RecommendStores,
    recommend_products: RecommendProducts,
    service_auth,
) -> APIRouter:
    """Build the Recommendation HTTP router."""

    router = APIRouter(
        prefix="/api/v1/recommendations",
        dependencies=[Depends(service_auth)],
    )

    @router.get(
        "/stores",
        response_model=RecommendationResponse,
    )
    async def recommend_store_list(
        query: str | None = Query(default=None, min_length=1),
        store_id: UUID | None = Query(default=None),
    ) -> RecommendationResponse:
        if query is None and store_id is None:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="query or store_id is required",
            )

        try:
            result = await asyncio.to_thread(
                recommend_stores.execute,
                RecommendStoresInput(
                    query=query,
                    store_id=store_id,
                ),
            )
        except ValueError as exc:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail=str(exc),
            ) from exc
        except Exception as exc:
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail="internal server error",
            ) from exc

        return RecommendationResponse(
            recommendations=tuple(
                RecommendationItem(
                    entity_id=item.store_id,
                    score=item.score,
                )
                for item in result.recommendations
            )
        )

    @router.get(
        "/products",
        response_model=RecommendationResponse,
    )
    async def recommend_product_list(
        query: str | None = Query(default=None, min_length=1),
        product_id: UUID | None = Query(default=None),
    ) -> RecommendationResponse:
        if query is None and product_id is None:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="query or product_id is required",
            )

        try:
            result = await asyncio.to_thread(
                recommend_products.execute,
                RecommendProductsInput(
                    query=query,
                    product_id=product_id,
                ),
            )
        except ValueError as exc:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail=str(exc),
            ) from exc
        except Exception as exc:
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail="internal server error",
            ) from exc

        return RecommendationResponse(
            recommendations=tuple(
                RecommendationItem(
                    entity_id=item.product_id,
                    score=item.score,
                )
                for item in result.recommendations
            )
        )

    return router
