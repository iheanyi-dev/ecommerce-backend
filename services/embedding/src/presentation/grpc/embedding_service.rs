//! Authenticated gRPC adapter for semantic embedding retrieval.

use std::sync::Arc;

use tonic::{Request, Response, Status};
use uuid::Uuid;

use crate::application::{
    dto::{RetrieveKnowledgeInput, RetrieveProductEmbeddingsInput, RetrieveStoreEmbeddingsInput},
    errors::ApplicationError,
    ports::Logger,
    use_cases::{
        retrieve_knowledge::RetrieveKnowledgeUseCase,
        retrieve_product_embeddings::RetrieveProductEmbeddingsUseCase,
        retrieve_store_embeddings::RetrieveStoreEmbeddingsUseCase,
    },
};

use super::pb::{
    KnowledgeChunk, RetrieveChunksRequest, RetrieveChunksResponse, RetrieveSimilarProductsRequest,
    RetrieveSimilarProductsResponse, RetrieveSimilarStoresRequest, RetrieveSimilarStoresResponse,
    SimilarProduct, SimilarStore, embedding_service_server::EmbeddingService,
};

/// Credentials and service identity required by the gRPC adapter.
#[derive(Debug, Clone)]
pub struct GrpcSettings {
    pub caller_service_name: String,
    pub caller_service_secret: String,
}

/// Implements the public Embedding gRPC contract.
pub struct EmbeddingGrpcService {
    retrieve_knowledge: Arc<RetrieveKnowledgeUseCase>,
    retrieve_products: Arc<RetrieveProductEmbeddingsUseCase>,
    retrieve_stores: Arc<RetrieveStoreEmbeddingsUseCase>,
    settings: GrpcSettings,
    logger: Arc<dyn Logger>,
}

impl EmbeddingGrpcService {
    pub fn new(
        retrieve_knowledge: Arc<RetrieveKnowledgeUseCase>,
        retrieve_products: Arc<RetrieveProductEmbeddingsUseCase>,
        retrieve_stores: Arc<RetrieveStoreEmbeddingsUseCase>,
        settings: GrpcSettings,
        logger: Arc<dyn Logger>,
    ) -> Self {
        Self {
            retrieve_knowledge,
            retrieve_products,
            retrieve_stores,
            settings,
            logger,
        }
    }

    fn authenticate<T>(&self, request: &Request<T>) -> Result<(), Status> {
        let metadata = request.metadata();

        let service_name = metadata
            .get("x-service-name")
            .and_then(|value| value.to_str().ok());

        let service_secret = metadata
            .get("x-service-secret")
            .and_then(|value| value.to_str().ok());

        if self.settings.caller_service_secret.is_empty()
            || service_name != Some(self.settings.caller_service_name.as_str())
            || service_secret != Some(self.settings.caller_service_secret.as_str())
        {
            return Err(Status::unauthenticated("Invalid service credentials"));
        }

        Ok(())
    }

    fn application_status(error: ApplicationError) -> Status {
        match error {
            ApplicationError::Validation(message) => Status::invalid_argument(message),
            ApplicationError::Domain(error) => Status::invalid_argument(error.to_string()),
            ApplicationError::Port(error) => Status::internal(error.to_string()),
        }
    }

    fn pagination(top_k: i32, offset: i32) -> Result<(usize, usize), Status> {
        let top_k = usize::try_from(top_k)
            .map_err(|_| Status::invalid_argument("top_k must not be negative"))?;

        let offset = usize::try_from(offset)
            .map_err(|_| Status::invalid_argument("offset must not be negative"))?;

        Ok((top_k, offset))
    }

    fn log_failure(&self, event: &str, error: &str) {
        self.logger.error(event, &[("error", error.to_owned())]);
    }
}

#[tonic::async_trait]
impl EmbeddingService for EmbeddingGrpcService {
    async fn retrieve_chunks(
        &self,
        request: Request<RetrieveChunksRequest>,
    ) -> Result<Response<RetrieveChunksResponse>, Status> {
        self.authenticate(&request)?;

        let request = request.into_inner();

        let store_id = Uuid::parse_str(&request.store_id)
            .map_err(|_| Status::invalid_argument("store_id must be a valid UUID"))?;

        if request.query.trim().is_empty() {
            return Err(Status::invalid_argument("query must not be empty"));
        }

        let input = RetrieveKnowledgeInput::new(store_id, request.query);
        let use_case = Arc::clone(&self.retrieve_knowledge);

        let result = tokio::task::spawn_blocking(move || use_case.execute(input))
            .await
            .map_err(|error| {
                self.log_failure(
                    "embedding.grpc.knowledge_retrieval_failed",
                    &error.to_string(),
                );
                Status::internal("Knowledge retrieval failed")
            })?;

        let output = result.map_err(|error| {
            if matches!(error, ApplicationError::Port(_)) {
                self.log_failure(
                    "embedding.grpc.knowledge_retrieval_failed",
                    &error.to_string(),
                );
            }
            Self::application_status(error)
        })?;

        let chunks = output
            .chunks
            .into_iter()
            .map(|item| {
                let filename = item
                    .chunk
                    .storage_reference()
                    .value()
                    .rsplit('/')
                    .next()
                    .unwrap_or_default()
                    .to_owned();

                KnowledgeChunk {
                    content: item.chunk.content().value().to_owned(),
                    source_filename: filename,
                    score: item.score,
                }
            })
            .collect();

        Ok(Response::new(RetrieveChunksResponse { chunks }))
    }

    async fn retrieve_similar_products(
        &self,
        request: Request<RetrieveSimilarProductsRequest>,
    ) -> Result<Response<RetrieveSimilarProductsResponse>, Status> {
        self.authenticate(&request)?;

        let request = request.into_inner();

        if request.query.trim().is_empty() {
            return Err(Status::invalid_argument("query must not be empty"));
        }

        let (top_k, offset) = Self::pagination(request.top_k, request.offset)?;

        let store_id = request
            .store_id
            .map(|value| {
                Uuid::parse_str(&value)
                    .map_err(|_| Status::invalid_argument("store_id must be a valid UUID"))
            })
            .transpose()?;

        let input = RetrieveProductEmbeddingsInput {
            query: request.query,
            top_k,
            offset,
            store_id,
        };

        let use_case = Arc::clone(&self.retrieve_products);

        let result = tokio::task::spawn_blocking(move || use_case.execute(input))
            .await
            .map_err(|error| {
                self.log_failure(
                    "embedding.grpc.product_retrieval_failed",
                    &error.to_string(),
                );
                Status::internal("Product retrieval failed")
            })?;

        let output = result.map_err(|error| {
            if matches!(error, ApplicationError::Port(_)) {
                self.log_failure(
                    "embedding.grpc.product_retrieval_failed",
                    &error.to_string(),
                );
            }
            Self::application_status(error)
        })?;

        let products = output
            .results
            .into_iter()
            .map(|item| SimilarProduct {
                product_id: item.embedding.id().to_string(),
                score: item.score,
            })
            .collect();

        Ok(Response::new(RetrieveSimilarProductsResponse { products }))
    }

    async fn retrieve_similar_stores(
        &self,
        request: Request<RetrieveSimilarStoresRequest>,
    ) -> Result<Response<RetrieveSimilarStoresResponse>, Status> {
        self.authenticate(&request)?;

        let request = request.into_inner();

        if request.query.trim().is_empty() {
            return Err(Status::invalid_argument("query must not be empty"));
        }

        let (top_k, offset) = Self::pagination(request.top_k, request.offset)?;

        let input = RetrieveStoreEmbeddingsInput {
            query: request.query,
            top_k,
            offset,
        };

        let use_case = Arc::clone(&self.retrieve_stores);

        let result = tokio::task::spawn_blocking(move || use_case.execute(input))
            .await
            .map_err(|error| {
                self.log_failure("embedding.grpc.store_retrieval_failed", &error.to_string());
                Status::internal("Store retrieval failed")
            })?;

        let output = result.map_err(|error| {
            if matches!(error, ApplicationError::Port(_)) {
                self.log_failure("embedding.grpc.store_retrieval_failed", &error.to_string());
            }
            Self::application_status(error)
        })?;

        let stores = output
            .results
            .into_iter()
            .map(|item| SimilarStore {
                store_id: item.embedding.id().to_string(),
                score: item.score,
            })
            .collect();

        Ok(Response::new(RetrieveSimilarStoresResponse { stores }))
    }
}
