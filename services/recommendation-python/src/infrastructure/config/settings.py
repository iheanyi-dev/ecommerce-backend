"""Recommendation Service runtime configuration."""

from __future__ import annotations

import os
from dataclasses import dataclass
from pathlib import Path

from dotenv import load_dotenv


@dataclass(frozen=True, slots=True)
class StorageSettings:
    """Configuration required to connect to the internal Storage Service."""

    grpc_address: str
    service_name: str
    service_secret: str


@dataclass(frozen=True, slots=True)
class Settings:
    """Top-level Recommendation Service configuration."""

    storage: StorageSettings


def load_settings() -> Settings:
    """Load Recommendation Service configuration from the local environment."""

    project_root = Path(__file__).resolve().parents[3]
    load_dotenv(project_root / ".env")

    grpc_address = os.getenv("STORAGE_GRPC_ADDR", "").strip()
    service_name = os.getenv("STORAGE_SERVICE_NAME", "").strip()
    service_secret = os.getenv("STORAGE_SERVICE_SECRET", "").strip()

    if not grpc_address:
        raise ValueError("STORAGE_GRPC_ADDR is required.")

    if not service_name:
        raise ValueError("STORAGE_SERVICE_NAME is required.")

    if not service_secret:
        raise ValueError("STORAGE_SERVICE_SECRET is required.")

    return Settings(
        storage=StorageSettings(
            grpc_address=grpc_address,
            service_name=service_name,
            service_secret=service_secret,
        )
    )
