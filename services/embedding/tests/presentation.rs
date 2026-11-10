use std::sync::{Arc, Mutex};

use embedding::{
    application::{
        dto::{ScoredKnowledgeChunk, ScoredProductEmbedding, ScoredStoreEmbedding},
        ports::{
            EmbeddingModel, KnowledgeChunkRepository, Logger, Metrics, PortResult,
            ProductEmbeddingRepository, StoreEmbeddingRepository,
        },
        use_cases::{
            retrieve_knowledge::RetrieveKnowledgeUseCase,
            retrieve_product_embeddings::RetrieveProductEmbeddingsUseCase,
            retrieve_store_embeddings::RetrieveStoreEmbeddingsUseCase,
        },
    },
    domain::{
        entities::{KnowledgeChunk, ProductEmbedding, StoreEmbedding},
        value_objects::{Embedding, EmbeddingModelVersion},
    },
    presentation::grpc::{
        embedding_service::{EmbeddingGrpcService, GrpcSettings},
        pb::{
            RetrieveChunksRequest, RetrieveSimilarProductsRequest, RetrieveSimilarStoresRequest,
            embedding_service_server::EmbeddingService,
        },
    },
};
use tonic::{Request, metadata::MetadataValue};
use uuid::Uuid;

#[derive(Default)]
struct FakeModel;

impl EmbeddingModel for FakeModel {
    fn generate(&self, _: &str) -> PortResult<(Embedding, EmbeddingModelVersion)> {
        Ok((
            Embedding::new(vec![0.1, 0.2, 0.3]).unwrap(),
            EmbeddingModelVersion::new("test-model").unwrap(),
        ))
    }
}

#[derive(Default)]
struct FakeLogger;

impl Logger for FakeLogger {
    fn info(&self, _: &str, _: &[(&str, String)]) {}
    fn warning(&self, _: &str, _: &[(&str, String)]) {}
    fn error(&self, _: &str, _: &[(&str, String)]) {}
}

#[derive(Default)]
struct FakeMetrics;

impl Metrics for FakeMetrics {
    fn increment(&self, _: &str, _: u64) {}
    fn observe(&self, _: &str, _: f64) {}
}

#[derive(Default)]
struct FakeKnowledgeRepository;

impl KnowledgeChunkRepository for FakeKnowledgeRepository {
    fn replace_for_knowledge(&self, _: Uuid, _: Vec<KnowledgeChunk>) -> PortResult<()> {
        Ok(())
    }

    fn delete_by_knowledge_id(&self, _: Uuid) -> PortResult<()> {
        Ok(())
    }

    fn search(
        &self,
        _: &Embedding,
        _: usize,
        _: usize,
        _: Uuid,
    ) -> PortResult<Vec<ScoredKnowledgeChunk>> {
        Ok(Vec::new())
    }
}

#[derive(Default)]
struct FakeProductRepository {
    pagination: Mutex<Vec<(usize, usize, Option<Uuid>)>>,
}

impl ProductEmbeddingRepository for FakeProductRepository {
    fn upsert(&self, _: ProductEmbedding) -> PortResult<()> {
        Ok(())
    }

    fn delete(&self, _: Uuid) -> PortResult<()> {
        Ok(())
    }

    fn delete_by_store_id(&self, _: Uuid) -> PortResult<()> {
        Ok(())
    }

    fn search(
        &self,
        _: &Embedding,
        top_k: usize,
        offset: usize,
        store_id: Option<Uuid>,
    ) -> PortResult<Vec<ScoredProductEmbedding>> {
        self.pagination
            .lock()
            .unwrap()
            .push((top_k, offset, store_id));
        Ok(Vec::new())
    }
}

#[derive(Default)]
struct FakeStoreRepository {
    pagination: Mutex<Vec<(usize, usize)>>,
}

impl StoreEmbeddingRepository for FakeStoreRepository {
    fn upsert(&self, _: StoreEmbedding) -> PortResult<()> {
        Ok(())
    }

    fn delete(&self, _: Uuid) -> PortResult<()> {
        Ok(())
    }

    fn search(
        &self,
        _: &Embedding,
        top_k: usize,
        offset: usize,
    ) -> PortResult<Vec<ScoredStoreEmbedding>> {
        self.pagination.lock().unwrap().push((top_k, offset));
        Ok(Vec::new())
    }
}

fn setup() -> (
    EmbeddingGrpcService,
    Arc<FakeProductRepository>,
    Arc<FakeStoreRepository>,
) {
    let model = Arc::new(FakeModel);
    let logger = Arc::new(FakeLogger);
    let metrics = Arc::new(FakeMetrics);

    let knowledge = Arc::new(RetrieveKnowledgeUseCase::new(
        Arc::new(FakeKnowledgeRepository),
        model.clone(),
        logger.clone(),
        metrics.clone(),
    ));

    let product_repository = Arc::new(FakeProductRepository::default());
    let products = Arc::new(RetrieveProductEmbeddingsUseCase::new(
        model.clone(),
        product_repository.clone(),
        logger.clone(),
        metrics.clone(),
    ));

    let store_repository = Arc::new(FakeStoreRepository::default());
    let stores = Arc::new(RetrieveStoreEmbeddingsUseCase::new(
        model,
        store_repository.clone(),
        logger.clone(),
        metrics,
    ));

    let service = EmbeddingGrpcService::new(
        knowledge,
        products,
        stores,
        GrpcSettings {
            caller_service_name: "order".into(),
            caller_service_secret: "test-secret".into(),
        },
        logger,
    );

    (service, product_repository, store_repository)
}

fn authenticated<T>(message: T) -> Request<T> {
    let mut request = Request::new(message);
    request
        .metadata_mut()
        .insert("x-service-name", MetadataValue::from_static("order"));
    request.metadata_mut().insert(
        "x-service-secret",
        MetadataValue::from_static("test-secret"),
    );
    request
}

#[tokio::test]
async fn rejects_requests_without_service_credentials() {
    let (service, _, _) = setup();

    let result = service
        .retrieve_similar_stores(Request::new(RetrieveSimilarStoresRequest {
            query: "electronics".into(),
            top_k: 5,
            offset: 0,
        }))
        .await;

    assert_eq!(result.unwrap_err().code(), tonic::Code::Unauthenticated);
}

#[tokio::test]
async fn rejects_invalid_store_id_for_chunk_retrieval() {
    let (service, _, _) = setup();

    let result = service
        .retrieve_chunks(authenticated(RetrieveChunksRequest {
            store_id: "invalid".into(),
            query: "return policy".into(),
        }))
        .await;

    assert_eq!(result.unwrap_err().code(), tonic::Code::InvalidArgument);
}

#[tokio::test]
async fn rejects_blank_product_query() {
    let (service, _, _) = setup();

    let result = service
        .retrieve_similar_products(authenticated(RetrieveSimilarProductsRequest {
            query: "   ".into(),
            top_k: 5,
            offset: 0,
            store_id: None,
        }))
        .await;

    assert_eq!(result.unwrap_err().code(), tonic::Code::InvalidArgument);
}

#[tokio::test]
async fn rejects_negative_pagination() {
    let (service, _, _) = setup();

    let result = service
        .retrieve_similar_stores(authenticated(RetrieveSimilarStoresRequest {
            query: "electronics".into(),
            top_k: -1,
            offset: 0,
        }))
        .await;

    assert_eq!(result.unwrap_err().code(), tonic::Code::InvalidArgument);
}

#[tokio::test]
async fn retrieves_products_and_stores_with_pagination_and_store_filter() {
    let (service, product_repository, store_repository) = setup();
    let store_id = Uuid::new_v4();

    let products = service
        .retrieve_similar_products(authenticated(RetrieveSimilarProductsRequest {
            query: "running shoes".into(),
            top_k: 7,
            offset: 3,
            store_id: Some(store_id.to_string()),
        }))
        .await
        .unwrap()
        .into_inner();

    let stores = service
        .retrieve_similar_stores(authenticated(RetrieveSimilarStoresRequest {
            query: "electronics retailer".into(),
            top_k: 4,
            offset: 2,
        }))
        .await
        .unwrap()
        .into_inner();

    assert!(products.products.is_empty());
    assert!(stores.stores.is_empty());

    assert_eq!(
        product_repository.pagination.lock().unwrap().as_slice(),
        &[(7, 3, Some(store_id))]
    );
    assert_eq!(
        store_repository.pagination.lock().unwrap().as_slice(),
        &[(4, 2)]
    );
}

#[tokio::test]
async fn retrieves_chunks_for_valid_store() {
    let (service, _, _) = setup();
    let result = service
        .retrieve_chunks(authenticated(RetrieveChunksRequest {
            store_id: Uuid::new_v4().to_string(),
            query: "return policy".into(),
        }))
        .await
        .unwrap()
        .into_inner();

    assert!(result.chunks.is_empty());
}
