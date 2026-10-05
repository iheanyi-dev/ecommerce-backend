"""Recommendation domain value objects."""

from .embedding import Embedding
from .embedding_model_version import EmbeddingModelVersion
from .entity_description import EntityDescription
from .entity_id import EntityID
from .entity_name import EntityName

__all__ = [
    "Embedding",
    "EmbeddingModelVersion",
    "EntityDescription",
    "EntityID",
    "EntityName",
]
