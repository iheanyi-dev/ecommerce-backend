"""Recommendation Service runtime configuration."""

from __future__ import annotations

import os
from dataclasses import dataclass
from pathlib import Path

from dotenv import load_dotenv


@dataclass(frozen=True, slots=True)
class ServiceSettings:
    """Configuration for Recommendation service authentication."""

    name: str
    secret: str


@dataclass(frozen=True, slots=True)
class HttpSettings:
    """Configuration for the external Recommendation HTTP server."""

    address: str


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
class KafkaSettings:
    """Configuration for Recommendation Kafka consumption."""

    brokers: str
    consumer_group: str


@dataclass(frozen=True, slots=True)
class Settings:
    """Top-level Recommendation Service configuration."""

    service: ServiceSettings
    http: HttpSettings
    qdrant: QdrantSettings
    embedding: EmbeddingSettings
    kafka: KafkaSettings


def load_settings() -> Settings:
    """Load Recommendation Service configuration from the environment."""

    project_root = Path(__file__).resolve().parents[3]
    load_dotenv(project_root / ".env")

    service_name = os.getenv(
        "RECOMMENDATION_SERVICE_NAME",
        "recommendation",
    ).strip()
    service_secret = os.getenv(
        "RECOMMENDATION_SERVICE_SECRET",
        "",
    ).strip()

    http_address = os.getenv(
        "RECOMMENDATION_HTTP_ADDR",
        "0.0.0.0:8091",
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

    kafka_brokers = os.getenv(
        "KAFKA_BROKERS",
        "localhost:9092",
    ).strip()
    kafka_consumer_group = os.getenv(
        "RECOMMENDATION_KAFKA_GROUP",
        "recommendation",
    ).strip()

    if not service_name:
        raise ValueError("RECOMMENDATION_SERVICE_NAME must not be empty.")

    if not service_secret:
        raise ValueError("RECOMMENDATION_SERVICE_SECRET is required.")

    if not http_address:
        raise ValueError("RECOMMENDATION_HTTP_ADDR must not be empty.")

    if not qdrant_url:
        raise ValueError("QDRANT_URL must not be empty.")

    if not kafka_brokers:
        raise ValueError("KAFKA_BROKERS must not be empty.")

    if not kafka_consumer_group:
        raise ValueError(
            "RECOMMENDATION_KAFKA_GROUP must not be empty."
        )

    return Settings(
        service=ServiceSettings(
            name=service_name,
            secret=service_secret,
        ),
        http=HttpSettings(
            address=http_address,
        ),
        qdrant=QdrantSettings(
            url=qdrant_url,
            api_key=qdrant_api_key,
        ),
        embedding=EmbeddingSettings(
            cache_dir=embedding_cache_dir,
        ),
        kafka=KafkaSettings(
            brokers=kafka_brokers,
            consumer_group=kafka_consumer_group,
        ),
    )
