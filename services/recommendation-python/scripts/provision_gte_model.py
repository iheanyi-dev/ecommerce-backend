"""Provision the cached GTE model into the internal Storage Service."""

from __future__ import annotations

from pathlib import Path

import grpc

from infrastructure.config.settings import load_settings
from infrastructure.embeddings.model_artifact_manager import ModelArtifactManager
from infrastructure.embeddings.model_artifact_provider import (
    HuggingFaceModelArtifactProvider,
)
from infrastructure.embeddings.model_manifest import GTE_MULTILINGUAL_BASE
from infrastructure.storage.client import StorageClient


def main() -> None:
    settings = load_settings()

    cache_dir = Path(".cache/models")

    print("Preparing cached GTE model artifact...")

    provider = HuggingFaceModelArtifactProvider(
        cache_dir=cache_dir,
    )

    channel = grpc.insecure_channel(
        settings.storage.grpc_address,
    )

    client = StorageClient(
        channel=channel,
        service_name=settings.storage.service_name,
        service_secret=settings.storage.service_secret,
    )

    manager = ModelArtifactManager(
        provider=provider,
        store=client,
        cache_dir=cache_dir,
    )

    try:
        reference = manager.ensure_stored(
            GTE_MULTILINGUAL_BASE,
        )

        print("GTE model successfully provisioned.")
        print(f"Storage reference: {reference}")
    finally:
        client.close()


if __name__ == "__main__":
    main()
