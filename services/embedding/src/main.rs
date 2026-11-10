use std::{error::Error, sync::Arc};

use embedding::{
    application::{
        ports::{EmbeddingModel, Logger, Metrics},
        use_cases::{
            create_product_embedding::CreateProductEmbeddingUseCase,
            create_store_embedding::CreateStoreEmbeddingUseCase,
            delete_knowledge::DeleteKnowledgeUseCase,
            delete_product_embedding::DeleteProductEmbeddingUseCase,
            delete_store_embedding::DeleteStoreEmbeddingUseCase,
            ingest_knowledge::IngestKnowledgeUseCase, retrieve_knowledge::RetrieveKnowledgeUseCase,
            retrieve_product_embeddings::RetrieveProductEmbeddingsUseCase,
            retrieve_store_embeddings::RetrieveStoreEmbeddingsUseCase,
        },
    },
    config::Settings,
    infrastructure::{
        embeddings::{
            candle_embedding_model::CandleEmbeddingModel, model_manifest::multilingual_e5_small,
        },
        messaging::{
            event_handlers::{EVENT_TOPICS, EmbeddingEventHandlers},
            kafka_consumer::KafkaEventConsumer,
        },
        observability::{logger::StructuredLogger, metrics::InMemoryMetrics},
        persistence::qdrant::{
            collections::{
                KNOWLEDGE_CHUNK_COLLECTION, PRODUCT_EMBEDDING_COLLECTION,
                STORE_EMBEDDING_COLLECTION, ensure_knowledge_chunk_collection,
                ensure_product_embedding_collection, ensure_store_embedding_collection,
            },
            knowledge_chunk_repository::QdrantKnowledgeChunkRepository,
            product_embedding_repository::QdrantProductEmbeddingRepository,
            store_embedding_repository::QdrantStoreEmbeddingRepository,
        },
        storage::storage_document_content_reader::StorageDocumentContentReader,
        text_processing::character_text_chunker::CharacterTextChunker,
    },
    presentation::grpc::{embedding_service::GrpcSettings, server::serve},
};
use qdrant_client::Qdrant;
use tracing_subscriber::EnvFilter;

type ServiceResult<T> = Result<T, Box<dyn Error + Send + Sync>>;

#[tokio::main]
async fn main() -> ServiceResult<()> {
    let settings = Settings::load()?;

    tracing_subscriber::fmt()
        .with_env_filter(
            EnvFilter::try_from_default_env().unwrap_or_else(|_| EnvFilter::new("info")),
        )
        .init();

    let logger: Arc<dyn Logger> = Arc::new(StructuredLogger::production());
    let metrics: Arc<dyn Metrics> = Arc::new(InMemoryMetrics::new());

    // The model remains lazy-loaded; constructing it does not download weights.
    let embedding_model: Arc<dyn EmbeddingModel> = Arc::new(CandleEmbeddingModel::new(
        settings.model_cache_dir.clone(),
        multilingual_e5_small(),
    ));

    // Connect to Qdrant and ensure all required collections exist.
    let mut qdrant_builder = Qdrant::from_url(&settings.qdrant_url);

    if let Some(api_key) = &settings.qdrant_api_key {
        qdrant_builder = qdrant_builder.api_key(api_key.clone());
    }

    let qdrant = qdrant_builder.build()?;
    let dimension = multilingual_e5_small().embedding_dimension as u64;

    ensure_knowledge_chunk_collection(&qdrant, dimension).await?;
    ensure_store_embedding_collection(&qdrant, dimension).await?;
    ensure_product_embedding_collection(&qdrant, dimension).await?;

    let knowledge_repository = Arc::new(QdrantKnowledgeChunkRepository::new(
        qdrant.clone(),
        KNOWLEDGE_CHUNK_COLLECTION,
    )?);

    let store_repository = Arc::new(QdrantStoreEmbeddingRepository::new(
        qdrant.clone(),
        STORE_EMBEDDING_COLLECTION,
    )?);

    let product_repository = Arc::new(QdrantProductEmbeddingRepository::new(
        qdrant,
        PRODUCT_EMBEDDING_COLLECTION,
    )?);

    // Storage adapter owns a Tokio runtime, so connect outside this runtime.
    let storage_address = settings.storage_address.clone();
    let storage_service_name = settings.storage_service_name.clone();
    let storage_service_secret = settings.storage_service_secret.clone();

    let document_reader = Arc::new(
        tokio::task::spawn_blocking(move || {
            StorageDocumentContentReader::connect(
                &storage_address,
                &storage_service_name,
                &storage_service_secret,
            )
        })
        .await??,
    );

    let chunker = Arc::new(CharacterTextChunker::default());

    // Write use cases.
    let create_store = Arc::new(CreateStoreEmbeddingUseCase::new(
        Arc::clone(&embedding_model),
        store_repository.clone(),
        Arc::clone(&logger),
        Arc::clone(&metrics),
    ));

    let delete_store = Arc::new(DeleteStoreEmbeddingUseCase::new(
        product_repository.clone(),
        store_repository.clone(),
        Arc::clone(&logger),
        Arc::clone(&metrics),
    ));

    let ingest_knowledge = Arc::new(IngestKnowledgeUseCase::new(
        document_reader,
        Arc::clone(&embedding_model),
        chunker,
        knowledge_repository.clone(),
        Arc::clone(&logger),
        Arc::clone(&metrics),
    ));

    let delete_knowledge = Arc::new(DeleteKnowledgeUseCase::new(
        knowledge_repository.clone(),
        Arc::clone(&logger),
        Arc::clone(&metrics),
    ));

    let create_product = Arc::new(CreateProductEmbeddingUseCase::new(
        Arc::clone(&embedding_model),
        product_repository.clone(),
        Arc::clone(&logger),
        Arc::clone(&metrics),
    ));

    let delete_product = Arc::new(DeleteProductEmbeddingUseCase::new(
        product_repository.clone(),
        Arc::clone(&logger),
        Arc::clone(&metrics),
    ));

    // Retrieval use cases.
    let retrieve_knowledge = Arc::new(RetrieveKnowledgeUseCase::new(
        knowledge_repository,
        Arc::clone(&embedding_model),
        Arc::clone(&logger),
        Arc::clone(&metrics),
    ));

    let retrieve_products = Arc::new(RetrieveProductEmbeddingsUseCase::new(
        Arc::clone(&embedding_model),
        product_repository,
        Arc::clone(&logger),
        Arc::clone(&metrics),
    ));

    let retrieve_stores = Arc::new(RetrieveStoreEmbeddingsUseCase::new(
        Arc::clone(&embedding_model),
        store_repository,
        Arc::clone(&logger),
        Arc::clone(&metrics),
    ));

    // Kafka event handling and consumer.
    let handlers = Arc::new(EmbeddingEventHandlers::new(
        create_store,
        delete_store,
        ingest_knowledge,
        delete_knowledge,
        create_product,
        delete_product,
    ));

    let consumer = KafkaEventConsumer::new(
        settings.kafka_brokers,
        settings.kafka_group_id,
        EVENT_TOPICS
            .iter()
            .map(|topic| (*topic).to_owned())
            .collect(),
        handlers,
        Arc::clone(&logger),
    )?;

    // Run Kafka and gRPC concurrently. If either stops, shut down the other.
    let mut kafka_task = tokio::spawn(async move { consumer.run().await });

    let grpc_settings = GrpcSettings {
        caller_service_name: settings.grpc_caller_service_name,
        caller_service_secret: settings.grpc_caller_service_secret,
    };

    let grpc_result = tokio::select! {
        result = serve(
            settings.grpc_address,
            grpc_settings,
            retrieve_knowledge,
            retrieve_products,
            retrieve_stores,
            logger,
        ) => result,

        result = &mut kafka_task => {
            match result {
                Ok(result) => {
                    result?;
                    Err("Kafka consumer stopped unexpectedly".into())
                }
                Err(error) => Err(Box::new(error) as Box<dyn Error + Send + Sync>),
            }
        }
    };

    kafka_task.abort();
    grpc_result
}
