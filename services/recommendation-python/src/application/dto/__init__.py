"""Public Recommendation application DTOs."""

from .find_associated_products import (
    FindAssociatedProductsInput,
    FindAssociatedProductsOutput,
)
from .find_associated_stores import (
    FindAssociatedStoresInput,
    FindAssociatedStoresOutput,
)
from .find_similar_products import (
    FindSimilarProductsInput,
    FindSimilarProductsOutput,
    ProductRecommendation,
)
from .find_similar_stores import (
    FindSimilarStoresInput,
    FindSimilarStoresOutput,
    StoreRecommendation,
)
from .recommend_products import RecommendProductsInput, RecommendProductsOutput
from .recommend_stores import RecommendStoresInput, RecommendStoresOutput

__all__ = [
    "FindAssociatedProductsInput",
    "FindAssociatedProductsOutput",
    "FindAssociatedStoresInput",
    "FindAssociatedStoresOutput",
    "FindSimilarProductsInput",
    "FindSimilarProductsOutput",
    "ProductRecommendation",
    "FindSimilarStoresInput",
    "FindSimilarStoresOutput",
    "StoreRecommendation",
    "RecommendProductsInput",
    "RecommendProductsOutput",
    "RecommendStoresInput",
    "RecommendStoresOutput",
]
