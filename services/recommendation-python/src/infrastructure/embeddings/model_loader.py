"""Pinned Hugging Face embedding model loading."""

from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path

import torch
from transformers import AutoModel, AutoTokenizer

from .model_manifest import EmbeddingModelManifest


@dataclass(frozen=True, slots=True)
class LoadedEmbeddingModel:
    """In-memory tokenizer and embedding model ready for inference."""

    tokenizer: object
    model: object
    device: torch.device


class GTEModelLoader:
    """Load a prepared GTE model from the local Hugging Face cache."""

    def __init__(
        self,
        manifest: EmbeddingModelManifest,
        cache_dir: Path,
    ) -> None:
        if not str(cache_dir).strip():
            raise ValueError("Embedding model cache directory must not be empty.")

        self._manifest = manifest
        self._cache_dir = cache_dir

    def load(self) -> LoadedEmbeddingModel:
        """Download if necessary through Hugging Face, then load into memory."""
        device = self._resolve_device()

        self._cache_dir.mkdir(parents=True, exist_ok=True)

        tokenizer = AutoTokenizer.from_pretrained(
            self._manifest.model_id,
            revision=self._manifest.revision,
            cache_dir=str(self._cache_dir),
            trust_remote_code=True,
        )

        model_kwargs: dict[str, object] = {
            "revision": self._manifest.revision,
            "cache_dir": str(self._cache_dir),
            "trust_remote_code": True,
        }

        if device.type == "cuda":
            model_kwargs["torch_dtype"] = torch.float16

        model = AutoModel.from_pretrained(
            self._manifest.model_id,
            **model_kwargs,
        )

        model.to(device)
        model.eval()

        return LoadedEmbeddingModel(
            tokenizer=tokenizer,
            model=model,
            device=device,
        )

    @staticmethod
    def _resolve_device() -> torch.device:
        if torch.cuda.is_available():
            return torch.device("cuda")

        if torch.backends.mps.is_available():
            return torch.device("mps")

        return torch.device("cpu")
