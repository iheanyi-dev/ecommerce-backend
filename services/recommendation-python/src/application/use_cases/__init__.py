"""Recommendation service application use cases."""

from .create_product_embedding import CreateProductEmbedding
from .create_store_embedding import CreateStoreEmbedding
from .delete_product_embedding import DeleteProductEmbedding
from .delete_store_embedding import DeleteStoreEmbedding
from .find_associated_products import FindAssociatedProducts
from .find_associated_stores import FindAssociatedStores
from .find_similar_products import FindSimilarProducts
from .find_similar_stores import FindSimilarStores

__all__ = [
    "CreateProductEmbedding",
    "CreateStoreEmbedding",
    "DeleteProductEmbedding",
    "DeleteStoreEmbedding",
    "FindAssociatedProducts",
    "FindAssociatedStores",
    "FindSimilarProducts",
    "FindSimilarStores",
]
