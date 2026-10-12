"""Environment-backed configuration for the Embedding service."""

from __future__ import annotations

import os
from dataclasses import dataclass
from pathlib import Path

from dotenv import load_dotenv


@dataclass(frozen=True, slots=True)
class StorageSettings:
    """Credentials and address used by the Storage gRPC client."""

    address: str
    service_name: str
    service_secret: str


@dataclass(frozen=True, slots=True)
class GrpcSettings:
    """Embedding gRPC listener and trusted caller credentials."""

    address: str
    caller_service_name: str
    caller_service_secret: str


@dataclass(frozen=True, slots=True)
class QdrantSettings:
    """Connection details for the vector database."""

    url: str
    api_key: str | None


@dataclass(frozen=True, slots=True)
class KafkaSettings:
    """Kafka consumer connection and consumer-group settings."""

    bootstrap_servers: str
    group_id: str


@dataclass(frozen=True, slots=True)
class EmbeddingSettings:
    """Local cache directory used by the embedding model."""

    cache_dir: Path


@dataclass(frozen=True, slots=True)
class Settings:
    """Complete runtime configuration."""

    storage: StorageSettings
    grpc: GrpcSettings
    qdrant: QdrantSettings
    kafka: KafkaSettings
    embedding: EmbeddingSettings




def _resolve_model_cache_dir(service_root: Path) -> Path:
    """Resolve the Hugging Face cache path portably for local and container runs."""
    configured = Path(
        os.getenv("EMBEDDING_MODEL_CACHE_DIR", ".cache/models/hub")
    ).expanduser()
    if not configured.is_absolute():
        configured = service_root / configured
    return configured.resolve()


def load_settings() -> Settings:
    """Load service-local environment configuration and validate essentials."""
    service_root = Path(__file__).resolve().parents[3]
    load_dotenv(service_root / ".env")

    storage_address = os.getenv("EMBEDDING_STORAGE_SERVICE_ADDR", "").strip()
    storage_secret = os.getenv("EMBEDDING_STORAGE_SERVICE_SECRET", "").strip()
    if not storage_address:
        raise ValueError("EMBEDDING_STORAGE_SERVICE_ADDR must be configured")
    if not storage_secret:
        raise ValueError("EMBEDDING_STORAGE_SERVICE_SECRET must be configured")

    qdrant_url = os.getenv("EMBEDDING_QDRANT_URL", "http://localhost:6333").strip()
    if not qdrant_url:
        raise ValueError("EMBEDDING_QDRANT_URL must not be empty")

    kafka_brokers = os.getenv(
        "EMBEDDING_KAFKA_BOOTSTRAP_SERVERS", "localhost:19092"
    ).strip()
    if not kafka_brokers:
        raise ValueError("EMBEDDING_KAFKA_BOOTSTRAP_SERVERS must not be empty")

    return Settings(
        storage=StorageSettings(
            address=storage_address,
            service_name=os.getenv(
                "EMBEDDING_STORAGE_SERVICE_NAME", "embedding"
            ).strip(),
            service_secret=storage_secret,
        ),
        grpc=GrpcSettings(
            address=os.getenv("EMBEDDING_GRPC_ADDR", "0.0.0.0:8092").strip(),
            caller_service_name=os.getenv(
                "EMBEDDING_CALLER_SERVICE_NAME", "chat"
            ).strip(),
            caller_service_secret=os.getenv(
                "EMBEDDING_CALLER_SERVICE_SECRET", ""
            ).strip(),
        ),
        qdrant=QdrantSettings(
            url=qdrant_url,
            api_key=os.getenv("EMBEDDING_QDRANT_API_KEY") or None,
        ),
        kafka=KafkaSettings(
            bootstrap_servers=kafka_brokers,
            group_id=os.getenv("EMBEDDING_KAFKA_GROUP_ID", "embedding").strip(),
        ),
        embedding=EmbeddingSettings(
            cache_dir=_resolve_model_cache_dir(service_root).expanduser(),
        ),
    )
