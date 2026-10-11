"""Lazily loaded GTE model adapter for generating knowledge embeddings."""

from __future__ import annotations

from threading import Lock
from typing import Any

import torch
import torch.nn.functional as functional
from transformers import AutoModel, AutoTokenizer

from application.ports.embedding_model import EmbeddingModel
from domain.value_objects import Embedding, EmbeddingModelVersion

from .model_manifest import (
    GTE_MULTILINGUAL_BASE,
    EmbeddingModelManifest,
)


class GTEEmbeddingModel(EmbeddingModel):
    """Load GTE on the first request and reuse it for subsequent requests.

    Model initialization is protected by a lock so concurrent first requests
    cannot load multiple copies. A failed initialization can be retried by a
    later request.
    """

    MAX_SEQUENCE_LENGTH = 8192

    def __init__(
        self,
        manifest: EmbeddingModelManifest = GTE_MULTILINGUAL_BASE,
    ) -> None:
        self._manifest = manifest
        self._model_version = EmbeddingModelVersion(
            f"{manifest.model_id}@{manifest.revision}"
        )

        # Do not download model artifacts during service startup.
        self._loaded: tuple[Any, Any, torch.device] | None = None
        self._load_lock = Lock()

    def generate(
        self,
        text: str,
    ) -> tuple[Embedding, EmbeddingModelVersion]:
        """Generate a normalized CLS embedding for the supplied text."""
        if not isinstance(text, str):
            raise TypeError("text must be a string")
        if not text.strip():
            raise ValueError("Embedding text must not be empty")

        tokenizer, model, device = self._get_loaded_model()

        encoded = tokenizer(
            text,
            padding=True,
            truncation=True,
            max_length=self.MAX_SEQUENCE_LENGTH,
            return_tensors="pt",
        )
        encoded = {
            key: value.to(device)
            for key, value in encoded.items()
        }

        # Inference does not need gradient tracking.
        with torch.no_grad():
            model_output = model(**encoded)

        # GTE represents a text using the first token's hidden state.
        cls_vector = model_output.last_hidden_state[:, 0, :]
        normalized_vector = functional.normalize(cls_vector, p=2, dim=-1)
        values = normalized_vector[0].detach().cpu().tolist()

        embedding = Embedding.from_values(values)
        if embedding.dimension != self._manifest.embedding_dimension:
            raise ValueError(
                "Generated embedding dimension does not match the model "
                f"manifest: expected {self._manifest.embedding_dimension}, "
                f"got {embedding.dimension}."
            )

        return embedding, self._model_version

    def _get_loaded_model(self) -> tuple[Any, Any, torch.device]:
        """Return the cached model, loading it exactly once when necessary."""
        if self._loaded is not None:
            return self._loaded

        with self._load_lock:
            # Another request may have finished loading while we waited.
            if self._loaded is not None:
                return self._loaded

            device = torch.device(
                "cuda" if torch.cuda.is_available() else "cpu"
            )

            # Pin both artifacts to the same immutable model revision.
            tokenizer = AutoTokenizer.from_pretrained(
                self._manifest.model_id,
                revision=self._manifest.revision,
                trust_remote_code=True,
            )
            model = AutoModel.from_pretrained(
                self._manifest.model_id,
                revision=self._manifest.revision,
                trust_remote_code=True,
            )
            model.to(device)
            model.eval()

            # Cache only after both artifacts load successfully.
            self._loaded = (tokenizer, model, device)
            return self._loaded
