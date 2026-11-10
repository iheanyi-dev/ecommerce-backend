//! Qdrant collection names and idempotent collection initialization.
//!
//! All Embedding service collections use cosine distance. Existing
//! collections are left unchanged, matching the Python implementation.

use std::{error::Error, io};

use qdrant_client::{
    Qdrant,
    qdrant::{CreateCollectionBuilder, Distance, VectorParamsBuilder},
};

/// Collection storing knowledge-document chunks and their embeddings.
pub const KNOWLEDGE_CHUNK_COLLECTION: &str = "knowledge_chunks";

/// Collection storing store-level embeddings.
pub const STORE_EMBEDDING_COLLECTION: &str = "store_embeddings";

/// Collection storing product embeddings.
pub const PRODUCT_EMBEDDING_COLLECTION: &str = "product_embeddings";

/// Error type returned by collection initialization operations.
pub type CollectionResult<T> = Result<T, Box<dyn Error + Send + Sync>>;

/// Minimal Qdrant collection operations required by this adapter.
///
/// Keeping these operations behind a trait allows collection initialization
/// to be tested without connecting to a running Qdrant server.
trait QdrantCollectionClient {
    /// Checks whether a collection already exists.
    async fn collection_exists(&self, collection_name: &str) -> CollectionResult<bool>;

    /// Creates a collection with the requested vector dimension.
    async fn create_collection(
        &self,
        collection_name: &str,
        vector_dimension: u64,
    ) -> CollectionResult<()>;
}

/// Implements collection operations using the official Qdrant client.
impl QdrantCollectionClient for Qdrant {
    /// Checks collection existence through Qdrant.
    async fn collection_exists(&self, collection_name: &str) -> CollectionResult<bool> {
        Ok(Qdrant::collection_exists(self, collection_name).await?)
    }

    /// Creates a cosine-distance collection through Qdrant.
    async fn create_collection(
        &self,
        collection_name: &str,
        vector_dimension: u64,
    ) -> CollectionResult<()> {
        Qdrant::create_collection(
            self,
            CreateCollectionBuilder::new(collection_name)
                .vectors_config(VectorParamsBuilder::new(vector_dimension, Distance::Cosine)),
        )
        .await?;

        Ok(())
    }
}

/// Creates a collection when it does not already exist.
///
/// A zero dimension is rejected before making any Qdrant request. If the
/// collection exists, this function returns successfully without changing
/// its configuration.
async fn ensure_collection<C: QdrantCollectionClient>(
    client: &C,
    collection_name: &str,
    vector_dimension: u64,
) -> CollectionResult<()> {
    if vector_dimension == 0 {
        return Err(io::Error::new(
            io::ErrorKind::InvalidInput,
            "Vector dimension must be a positive integer",
        )
        .into());
    }

    if client.collection_exists(collection_name).await? {
        return Ok(());
    }

    client
        .create_collection(collection_name, vector_dimension)
        .await
}

/// Ensures the knowledge-chunk collection exists.
pub async fn ensure_knowledge_chunk_collection(
    client: &Qdrant,
    vector_dimension: u64,
) -> CollectionResult<()> {
    ensure_collection(client, KNOWLEDGE_CHUNK_COLLECTION, vector_dimension).await
}

/// Ensures the store-embedding collection exists.
pub async fn ensure_store_embedding_collection(
    client: &Qdrant,
    vector_dimension: u64,
) -> CollectionResult<()> {
    ensure_collection(client, STORE_EMBEDDING_COLLECTION, vector_dimension).await
}

/// Ensures the product-embedding collection exists.
pub async fn ensure_product_embedding_collection(
    client: &Qdrant,
    vector_dimension: u64,
) -> CollectionResult<()> {
    ensure_collection(client, PRODUCT_EMBEDDING_COLLECTION, vector_dimension).await
}

#[cfg(test)]
mod tests {
    use std::sync::Mutex;

    use super::*;

    /// In-memory Qdrant substitute for collection initialization tests.
    struct FakeQdrantClient {
        exists: bool,
        created: Mutex<Vec<(String, u64)>>,
    }

    impl FakeQdrantClient {
        /// Creates a fake client with the requested initial existence state.
        fn new(exists: bool) -> Self {
            Self {
                exists,
                created: Mutex::new(Vec::new()),
            }
        }

        /// Returns a snapshot of collection creation requests.
        fn created(&self) -> Vec<(String, u64)> {
            self.created.lock().unwrap().clone()
        }
    }

    impl QdrantCollectionClient for FakeQdrantClient {
        /// Returns the configured collection existence state.
        async fn collection_exists(&self, _collection_name: &str) -> CollectionResult<bool> {
            Ok(self.exists)
        }

        /// Records a collection creation request.
        async fn create_collection(
            &self,
            collection_name: &str,
            vector_dimension: u64,
        ) -> CollectionResult<()> {
            self.created
                .lock()
                .unwrap()
                .push((collection_name.to_owned(), vector_dimension));

            Ok(())
        }
    }

    /// Ensures an absent collection is created with the requested dimension.
    #[tokio::test]
    async fn ensure_collection_creates_missing_collection() {
        let client = FakeQdrantClient::new(false);

        ensure_collection(&client, KNOWLEDGE_CHUNK_COLLECTION, 768)
            .await
            .unwrap();

        assert_eq!(
            client.created(),
            vec![(KNOWLEDGE_CHUNK_COLLECTION.to_owned(), 768)]
        );
    }

    /// Ensures an existing collection is not recreated or modified.
    #[tokio::test]
    async fn ensure_collection_does_not_recreate_existing_collection() {
        let client = FakeQdrantClient::new(true);

        ensure_collection(&client, STORE_EMBEDDING_COLLECTION, 768)
            .await
            .unwrap();

        assert!(client.created().is_empty());
    }

    /// Ensures zero vector dimensions are rejected before checking existence.
    #[tokio::test]
    async fn ensure_collection_rejects_zero_vector_dimension() {
        let client = FakeQdrantClient::new(false);

        let error = ensure_collection(&client, PRODUCT_EMBEDDING_COLLECTION, 0)
            .await
            .unwrap_err();

        assert_eq!(
            error.to_string(),
            "Vector dimension must be a positive integer"
        );
        assert!(client.created().is_empty());
    }

    /// Ensures the Python collection names remain unchanged.
    #[test]
    fn collection_names_match_python_implementation() {
        assert_eq!(KNOWLEDGE_CHUNK_COLLECTION, "knowledge_chunks");
        assert_eq!(STORE_EMBEDDING_COLLECTION, "store_embeddings");
        assert_eq!(PRODUCT_EMBEDDING_COLLECTION, "product_embeddings");
    }
}
