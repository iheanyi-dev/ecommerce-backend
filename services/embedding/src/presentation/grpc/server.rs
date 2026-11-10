//! gRPC server lifecycle.

use std::{net::SocketAddr, sync::Arc};

use tonic::transport::Server;

use super::tracing_interceptor::TracingLayer;

use crate::{
    application::ports::Logger,
    application::use_cases::{
        retrieve_knowledge::RetrieveKnowledgeUseCase,
        retrieve_product_embeddings::RetrieveProductEmbeddingsUseCase,
        retrieve_store_embeddings::RetrieveStoreEmbeddingsUseCase,
    },
};

use super::{
    embedding_service::{EmbeddingGrpcService, GrpcSettings},
    pb::embedding_service_server::EmbeddingServiceServer,
};

/// Starts the Embedding gRPC server and waits for shutdown.
pub async fn serve(
    address: SocketAddr,
    settings: GrpcSettings,
    retrieve_knowledge: Arc<RetrieveKnowledgeUseCase>,
    retrieve_products: Arc<RetrieveProductEmbeddingsUseCase>,
    retrieve_stores: Arc<RetrieveStoreEmbeddingsUseCase>,
    logger: Arc<dyn Logger>,
) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
    let service = EmbeddingGrpcService::new(
        retrieve_knowledge,
        retrieve_products,
        retrieve_stores,
        settings,
        Arc::clone(&logger),
    );

    logger.info(
        "embedding.grpc.started",
        &[("address", address.to_string())],
    );

    Server::builder()
        .layer(TracingLayer)
        .add_service(EmbeddingServiceServer::new(service))
        .serve_with_shutdown(address, async {
            let _ = tokio::signal::ctrl_c().await;
        })
        .await?;

    Ok(())
}
