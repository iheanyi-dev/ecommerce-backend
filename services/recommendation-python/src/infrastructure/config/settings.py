"""Recommendation Service runtime configuration."""

from __future__ import annotations

import os
from dataclasses import dataclass
from pathlib import Path

from dotenv import load_dotenv


@dataclass(frozen=True, slots=True)
class ServiceSettings:
    """Credentials for requests entering Recommendation."""

    name: str
    secret: str


@dataclass(frozen=True, slots=True)
class HttpSettings:
    """External Recommendation HTTP listener."""

    address: str


@dataclass(frozen=True, slots=True)
class EmbeddingServiceSettings:
    """Connection and caller credentials for Embedding gRPC."""

    address: str
    service_name: str
    service_secret: str
    timeout_seconds: float


@dataclass(frozen=True, slots=True)
class Settings:
    """Top-level Recommendation configuration."""

    service: ServiceSettings
    http: HttpSettings
    embedding_service: EmbeddingServiceSettings


def load_settings() -> Settings:
    """Load settings from the service's development .env and environment."""
    project_root = Path(__file__).resolve().parents[3]
    load_dotenv(project_root / ".env")

    service_name = os.getenv(
        "RECOMMENDATION_SERVICE_NAME", "recommendation"
    ).strip()
    service_secret = os.getenv("RECOMMENDATION_SERVICE_SECRET", "").strip()
    http_address = os.getenv(
        "RECOMMENDATION_HTTP_ADDR", "0.0.0.0:8091"
    ).strip()

    embedding_address = os.getenv(
        "EMBEDDING_SERVICE_ADDR", "localhost:8092"
    ).strip()
    embedding_service_name = os.getenv(
        "EMBEDDING_SERVICE_NAME", "chat"
    ).strip()
    embedding_service_secret = os.getenv(
        "EMBEDDING_SERVICE_SECRET", ""
    ).strip()

    timeout_text = os.getenv("EMBEDDING_GRPC_TIMEOUT_SECONDS", "10").strip()
    try:
        timeout_seconds = float(timeout_text)
    except ValueError as exc:
        raise ValueError(
            "EMBEDDING_GRPC_TIMEOUT_SECONDS must be a positive number"
        ) from exc

    if not service_name:
        raise ValueError("RECOMMENDATION_SERVICE_NAME must not be empty")
    if not service_secret:
        raise ValueError("RECOMMENDATION_SERVICE_SECRET is required")
    if not http_address or ":" not in http_address:
        raise ValueError(
            "RECOMMENDATION_HTTP_ADDR must use host:port format"
        )
    if not embedding_address:
        raise ValueError("EMBEDDING_SERVICE_ADDR must not be empty")
    if not embedding_service_name:
        raise ValueError("EMBEDDING_SERVICE_NAME must not be empty")
    if not embedding_service_secret:
        raise ValueError("EMBEDDING_SERVICE_SECRET is required")
    if timeout_seconds <= 0:
        raise ValueError(
            "EMBEDDING_GRPC_TIMEOUT_SECONDS must be positive"
        )

    return Settings(
        service=ServiceSettings(name=service_name, secret=service_secret),
        http=HttpSettings(address=http_address),
        embedding_service=EmbeddingServiceSettings(
            address=embedding_address,
            service_name=embedding_service_name,
            service_secret=embedding_service_secret,
            timeout_seconds=timeout_seconds,
        ),
    )
