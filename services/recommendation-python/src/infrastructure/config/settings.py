"""Recommendation Service runtime configuration."""

from __future__ import annotations

import os
from dataclasses import dataclass
from pathlib import Path

from dotenv import load_dotenv


@dataclass(frozen=True, slots=True)
class GrpcSettings:
    """Configuration for the internal Recommendation gRPC server."""

    address: str
    service_name: str
    service_secret: str


@dataclass(frozen=True, slots=True)
class QdrantSettings:
    """Configuration for the Recommendation Qdrant connection."""

    url: str
    api_key: str | None


@dataclass(frozen=True, slots=True)
class EmbeddingSettings:
    """Configuration for the local embedding model cache."""

    cache_dir: Path


@dataclass(frozen=True, slots=True)
class Settings:
    """Top-level Recommendation Service configuration."""

    grpc: GrpcSettings
    qdrant: QdrantSettings
    embedding: EmbeddingSettings


def load_settings() -> Settings:
    """Load Recommendation Service configuration from the environment."""

    project_root = Path(__file__).resolve().parents[3]
    load_dotenv(project_root / ".env")

    grpc_address = os.getenv(
        "RECOMMENDATION_GRPC_ADDR",
        "0.0.0.0:8090",
    ).strip()
    service_name = os.getenv(
        "RECOMMENDATION_SERVICE_NAME",
        "recommendation",
    ).strip()
    service_secret = os.getenv(
        "RECOMMENDATION_SERVICE_SECRET",
        "",
    ).strip()

    qdrant_url = os.getenv(
        "QDRANT_URL",
        "http://localhost:6333",
    ).strip()
    qdrant_api_key = os.getenv("QDRANT_API_KEY", "").strip() or None

    embedding_cache_dir = Path(
        os.getenv(
            "EMBEDDING_MODEL_CACHE_DIR",
            str(project_root / ".cache" / "huggingface"),
        ).strip()
    )

    if not grpc_address:
        raise ValueError("RECOMMENDATION_GRPC_ADDR must not be empty.")

    if not service_name:
        raise ValueError("RECOMMENDATION_SERVICE_NAME must not be empty.")

    if not service_secret:
        raise ValueError("RECOMMENDATION_SERVICE_SECRET is required.")

    if not qdrant_url:
        raise ValueError("QDRANT_URL must not be empty.")

    return Settings(
        grpc=GrpcSettings(
            address=grpc_address,
            service_name=service_name,
            service_secret=service_secret,
        ),
        qdrant=QdrantSettings(
            url=qdrant_url,
            api_key=qdrant_api_key,
        ),
        embedding=EmbeddingSettings(
            cache_dir=embedding_cache_dir,
        ),
    )
