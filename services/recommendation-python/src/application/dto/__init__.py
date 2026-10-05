"""Recommendation service application DTOs."""

from .create_product_embedding import (
    CreateProductEmbeddingInput,
    CreateProductEmbeddingOutput,
)
from .create_store_embedding import (
    CreateStoreEmbeddingInput,
    CreateStoreEmbeddingOutput,
)
from .delete_product_embedding import DeleteProductEmbeddingInput
from .delete_store_embedding import DeleteStoreEmbeddingInput
from .find_similar_stores import (
    FindSimilarStoresInput,
    FindSimilarStoresOutput,
    StoreRecommendation,
)
from .find_similar_products import (
    FindSimilarProductsInput,
    FindSimilarProductsOutput,
    ProductRecommendation,
)
from .find_associated_stores import (
    FindAssociatedStoresInput,
    FindAssociatedStoresOutput,
)
from .find_associated_products import (
    FindAssociatedProductsInput,
    FindAssociatedProductsOutput,
)

__all__ = [
    "CreateProductEmbeddingInput",
    "CreateProductEmbeddingOutput",
    "CreateStoreEmbeddingInput",
    "CreateStoreEmbeddingOutput",
    "DeleteProductEmbeddingInput",
    "DeleteStoreEmbeddingInput",
    "FindSimilarStoresInput",
    "FindSimilarStoresOutput",
    "StoreRecommendation",
    "FindSimilarProductsInput",
    "FindSimilarProductsOutput",
    "ProductRecommendation",
    "FindAssociatedStoresInput",
    "FindAssociatedStoresOutput",
    "FindAssociatedProductsInput",
    "FindAssociatedProductsOutput",
]
