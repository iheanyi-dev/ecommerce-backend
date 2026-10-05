"""GTE-based embedding generation."""

from __future__ import annotations

from typing import Any

import torch
import torch.nn.functional as functional

from application.ports.embedding_generator import EmbeddingGenerator
from domain.value_objects import (
    Embedding,
    EmbeddingModelVersion,
    EntityDescription,
    EntityName,
)

from .model_loader import GTEModelLoader, LoadedEmbeddingModel
from .model_manifest import EmbeddingModelManifest


class GTEEmbeddingGenerator(EmbeddingGenerator):
    """
    Generate Store and Product embeddings using the pinned GTE model.

    The model is loaded once during service startup and injected into
    this generator. Subsequent embedding requests reuse the same in-memory
    tokenizer and model.

    GTE's official embedding implementation uses the first token (CLS)
    representation followed by L2 normalization. This adapter deliberately
    follows that behavior rather than substituting a different pooling
    strategy.

    The application layer remains unaware of PyTorch, Transformers, model
    loading, device selection, and tokenization.
    """

    MAX_SEQUENCE_LENGTH = 8192

    def __init__(
        self,
        loaded_model: LoadedEmbeddingModel,
        manifest: EmbeddingModelManifest,
    ) -> None:
        self._manifest = manifest
        self._loaded = loaded_model
        self._model_version = EmbeddingModelVersion(
            f"{manifest.model_id}@{manifest.revision}"
        )

    def generate_store_embedding(
        self,
        name: EntityName,
        description: EntityDescription,
    ) -> tuple[Embedding, EmbeddingModelVersion]:
        """Generate the semantic representation of a Store."""
        return self._generate_entity_embedding(name, description)

    def generate_store_query_embedding(
        self,
        query: str,
    ) -> Embedding:
        """Generate an embedding for a Store similarity query."""
        return self._generate_query_embedding(query)

    def generate_product_embedding(
        self,
        name: EntityName,
        description: EntityDescription,
    ) -> tuple[Embedding, EmbeddingModelVersion]:
        """Generate the semantic representation of a Product."""
        return self._generate_entity_embedding(name, description)

    def generate_product_query_embedding(
        self,
        query: str,
    ) -> Embedding:
        """Generate an embedding for a Product similarity query."""
        return self._generate_query_embedding(query)

    def _generate_entity_embedding(
        self,
        name: EntityName,
        description: EntityDescription,
    ) -> tuple[Embedding, EmbeddingModelVersion]:
        """
        Generate an embedding from an entity's name and description.

        A structured separator makes the boundary between the two semantic
        fields explicit while keeping the representation model-agnostic.
        """
        text = f"{name}\n{description}"

        return (
            self._encode(text),
            self._model_version,
        )

    def _generate_query_embedding(
        self,
        query: str,
    ) -> Embedding:
        """Generate an embedding from a similarity-search query."""
        if not query.strip():
            raise ValueError("Embedding query must not be empty.")

        return self._encode(query)

    def _encode(
        self,
        text: str,
    ) -> Embedding:
        """Tokenize text and generate a normalized CLS embedding."""
        tokenizer: Any = self._loaded.tokenizer
        model: Any = self._loaded.model

        encoded = tokenizer(
            text,
            padding=True,
            truncation=True,
            max_length=self.MAX_SEQUENCE_LENGTH,
            return_tensors="pt",
        )

        encoded = {
            key: value.to(self._loaded.device)
            for key, value in encoded.items()
        }

        with torch.no_grad():
            model_output = model(**encoded)

        dense_vector = model_output.last_hidden_state[:, 0, :]

        normalized_vector = functional.normalize(
            dense_vector,
            dim=-1,
        )

        values = normalized_vector[0].detach().cpu().tolist()

        embedding = Embedding.from_values(values)

        if embedding.dimension != self._manifest.embedding_dimension:
            raise ValueError(
                "Generated embedding dimension does not match the model "
                f"manifest: expected {self._manifest.embedding_dimension}, "
                f"got {embedding.dimension}."
            )

        return embedding
