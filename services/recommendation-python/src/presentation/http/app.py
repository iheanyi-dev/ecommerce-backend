"""FastAPI application factory."""

from __future__ import annotations

from fastapi import FastAPI

from application.use_cases.recommend_products import RecommendProducts
from application.use_cases.recommend_stores import RecommendStores

from .auth import require_service_auth
from .middleware import (
    HTTPObservabilityMiddleware,
    RequestIDMiddleware,
    TracingMiddleware,
)
from .router import create_router


def create_app(
    *,
    service_name: str,
    service_secret: str,
    recommend_stores: RecommendStores,
    recommend_products: RecommendProducts,
    logger,
    metrics,
) -> FastAPI:
    """Create the fully configured Recommendation HTTP application."""

    app = FastAPI(
        title="Recommendation Service",
        version="1.0.0",
    )

    app.add_middleware(
        HTTPObservabilityMiddleware,
        logger=logger,
        metrics=metrics,
    )
    app.add_middleware(
        TracingMiddleware,
        service_name=service_name,
    )
    app.add_middleware(RequestIDMiddleware)

    service_auth = require_service_auth(
        expected_service_name=service_name,
        expected_service_secret=service_secret,
    )

    app.include_router(
        create_router(
            recommend_stores=recommend_stores,
            recommend_products=recommend_products,
            service_auth=service_auth,
        )
    )

    @app.get("/health")
    def health() -> dict[str, str]:
        return {"status": "ok"}

    return app
