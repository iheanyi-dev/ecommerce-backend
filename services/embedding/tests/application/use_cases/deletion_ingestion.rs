//! Behaviour tests for deletion and knowledge-ingestion use cases.

use std::{
    io,
    sync::{Arc, Mutex},
};

use embedding::{
    application::{
        dto::{
            DeleteKnowledgeInput, DeleteProductEmbeddingInput, DeleteStoreEmbeddingInput,
            IngestKnowledgeInput, ScoredProductEmbedding, ScoredStoreEmbedding,
        },
        ports::{
            DocumentContentReader, EmbeddingModel, KnowledgeChunkRepository, Logger, Metrics,
            PortResult, ProductEmbeddingRepository, StoreEmbeddingRepository, TextChunker,
        },
        use_cases::{
            delete_knowledge::DeleteKnowledgeUseCase,
            delete_product_embedding::DeleteProductEmbeddingUseCase,
            delete_store_embedding::DeleteStoreEmbeddingUseCase,
            ingest_knowledge::IngestKnowledgeUseCase,
        },
    },
    domain::{
        entities::{KnowledgeChunk, ProductEmbedding, StoreEmbedding},
        value_objects::{Embedding, EmbeddingModelVersion, StorageReference},
    },
};
use uuid::Uuid;

fn port_error(message: &str) -> Box<dyn std::error::Error + Send + Sync> {
    Box::new(io::Error::other(message.to_owned()))
}

#[derive(Default)]
struct FakeLogger;

impl Logger for FakeLogger {
    fn info(&self, _: &str, _: &[(&str, String)]) {}
    fn warning(&self, _: &str, _: &[(&str, String)]) {}
    fn error(&self, _: &str, _: &[(&str, String)]) {}
}

#[derive(Default)]
struct FakeMetrics {
    increments: Mutex<Vec<(String, u64)>>,
    observations: Mutex<Vec<(String, f64)>>,
}

impl Metrics for FakeMetrics {
    fn increment(&self, name: &str, value: u64) {
        self.increments
            .lock()
            .unwrap()
            .push((name.to_owned(), value));
    }

    fn observe(&self, name: &str, value: f64) {
        self.observations
            .lock()
            .unwrap()
            .push((name.to_owned(), value));
    }
}

#[derive(Default)]
struct FakeKnowledgeRepository {
    deleted: Mutex<Vec<Uuid>>,
    replaced: Mutex<Vec<(Uuid, Vec<KnowledgeChunk>)>>,
    fail_delete: bool,
    fail_replace: bool,
}

impl KnowledgeChunkRepository for FakeKnowledgeRepository {
    fn replace_for_knowledge(
        &self,
        knowledge_id: Uuid,
        chunks: Vec<KnowledgeChunk>,
    ) -> PortResult<()> {
        if self.fail_replace {
            return Err(port_error("replace failed"));
        }

        self.replaced.lock().unwrap().push((knowledge_id, chunks));

        Ok(())
    }

    fn delete_by_knowledge_id(&self, knowledge_id: Uuid) -> PortResult<()> {
        self.deleted.lock().unwrap().push(knowledge_id);

        if self.fail_delete {
            return Err(port_error("delete failed"));
        }

        Ok(())
    }

    fn search(
        &self,
        _: &Embedding,
        _: usize,
        _: usize,
        _: Uuid,
    ) -> PortResult<Vec<embedding::application::dto::ScoredKnowledgeChunk>> {
        Ok(Vec::new())
    }
}

#[derive(Default)]
struct FakeProductRepository {
    deleted: Mutex<Vec<Uuid>>,
    deleted_by_store: Mutex<Vec<Uuid>>,
    fail_delete: bool,
    fail_delete_by_store: bool,
    calls: Option<Arc<Mutex<Vec<&'static str>>>>,
}

impl ProductEmbeddingRepository for FakeProductRepository {
    fn upsert(&self, _: ProductEmbedding) -> PortResult<()> {
        Ok(())
    }

    fn delete(&self, product_id: Uuid) -> PortResult<()> {
        self.deleted.lock().unwrap().push(product_id);

        if self.fail_delete {
            return Err(port_error("product delete failed"));
        }

        Ok(())
    }

    fn delete_by_store_id(&self, store_id: Uuid) -> PortResult<()> {
        self.deleted_by_store.lock().unwrap().push(store_id);

        if let Some(calls) = &self.calls {
            calls.lock().unwrap().push("products");
        }

        if self.fail_delete_by_store {
            return Err(port_error("product store deletion failed"));
        }

        Ok(())
    }

    fn search(
        &self,
        _: &Embedding,
        _: usize,
        _: usize,
        _: Option<Uuid>,
    ) -> PortResult<Vec<ScoredProductEmbedding>> {
        Ok(Vec::new())
    }
}

#[derive(Default)]
struct FakeStoreRepository {
    deleted: Mutex<Vec<Uuid>>,
    calls: Option<Arc<Mutex<Vec<&'static str>>>>,
    fail_delete: bool,
}

impl StoreEmbeddingRepository for FakeStoreRepository {
    fn upsert(&self, _: StoreEmbedding) -> PortResult<()> {
        Ok(())
    }

    fn delete(&self, store_id: Uuid) -> PortResult<()> {
        self.deleted.lock().unwrap().push(store_id);

        if let Some(calls) = &self.calls {
            calls.lock().unwrap().push("store");
        }

        if self.fail_delete {
            return Err(port_error("store delete failed"));
        }

        Ok(())
    }

    fn search(&self, _: &Embedding, _: usize, _: usize) -> PortResult<Vec<ScoredStoreEmbedding>> {
        Ok(Vec::new())
    }
}

struct FakeDocumentReader {
    content: String,
    references: Mutex<Vec<String>>,
}

impl FakeDocumentReader {
    fn new(content: &str) -> Self {
        Self {
            content: content.to_owned(),
            references: Mutex::new(Vec::new()),
        }
    }
}

impl DocumentContentReader for FakeDocumentReader {
    fn read(&self, reference: &StorageReference) -> PortResult<String> {
        self.references
            .lock()
            .unwrap()
            .push(reference.value().to_owned());

        Ok(self.content.clone())
    }
}

struct FakeChunker {
    chunks: Vec<String>,
    calls: Mutex<Vec<String>>,
}

impl FakeChunker {
    fn new(chunks: Vec<&str>) -> Self {
        Self {
            chunks: chunks.into_iter().map(str::to_owned).collect(),
            calls: Mutex::new(Vec::new()),
        }
    }
}

impl TextChunker for FakeChunker {
    fn chunk(&self, text: &str) -> PortResult<Vec<String>> {
        self.calls.lock().unwrap().push(text.to_owned());
        Ok(self.chunks.clone())
    }
}

struct FakeEmbeddingModel {
    generated_texts: Mutex<Vec<String>>,
    fail_on_call: Option<usize>,
}

impl FakeEmbeddingModel {
    fn new() -> Self {
        Self {
            generated_texts: Mutex::new(Vec::new()),
            fail_on_call: None,
        }
    }

    fn failing_on_call(call: usize) -> Self {
        Self {
            generated_texts: Mutex::new(Vec::new()),
            fail_on_call: Some(call),
        }
    }
}

impl EmbeddingModel for FakeEmbeddingModel {
    fn generate(&self, text: &str) -> PortResult<(Embedding, EmbeddingModelVersion)> {
        let mut generated = self.generated_texts.lock().unwrap();
        generated.push(text.to_owned());

        if self.fail_on_call == Some(generated.len()) {
            return Err(port_error("embedding generation failed"));
        }

        Ok((
            Embedding::new(vec![0.1, 0.2, 0.3]).unwrap(),
            EmbeddingModelVersion::new("test-model").unwrap(),
        ))
    }
}

fn ingestion_use_case(
    reader: Arc<FakeDocumentReader>,
    model: Arc<FakeEmbeddingModel>,
    chunker: Arc<FakeChunker>,
    repository: Arc<FakeKnowledgeRepository>,
    metrics: Arc<FakeMetrics>,
) -> IngestKnowledgeUseCase {
    IngestKnowledgeUseCase::new(
        reader,
        model,
        chunker,
        repository,
        Arc::new(FakeLogger),
        metrics,
    )
}

#[test]
fn delete_knowledge_deletes_by_knowledge_id_and_records_metric() {
    let repository = Arc::new(FakeKnowledgeRepository::default());
    let metrics = Arc::new(FakeMetrics::default());
    let knowledge_id = Uuid::new_v4();

    let use_case =
        DeleteKnowledgeUseCase::new(repository.clone(), Arc::new(FakeLogger), metrics.clone());

    use_case
        .execute(DeleteKnowledgeInput { knowledge_id })
        .unwrap();

    assert_eq!(
        repository.deleted.lock().unwrap().as_slice(),
        &[knowledge_id]
    );
    assert_eq!(
        metrics.increments.lock().unwrap().as_slice(),
        &[("embedding.knowledge.deleted".into(), 1)]
    );
}

#[test]
fn delete_knowledge_propagates_repository_failure_without_recording_metric() {
    let repository = Arc::new(FakeKnowledgeRepository {
        fail_delete: true,
        ..Default::default()
    });
    let metrics = Arc::new(FakeMetrics::default());

    let use_case =
        DeleteKnowledgeUseCase::new(repository.clone(), Arc::new(FakeLogger), metrics.clone());

    let result = use_case.execute(DeleteKnowledgeInput {
        knowledge_id: Uuid::new_v4(),
    });

    assert!(result.is_err());
    assert!(metrics.increments.lock().unwrap().is_empty());
}

#[test]
fn delete_product_embedding_deletes_by_product_id_and_records_metric() {
    let repository = Arc::new(FakeProductRepository::default());
    let metrics = Arc::new(FakeMetrics::default());
    let product_id = Uuid::new_v4();

    let use_case = DeleteProductEmbeddingUseCase::new(
        repository.clone(),
        Arc::new(FakeLogger),
        metrics.clone(),
    );

    use_case
        .execute(DeleteProductEmbeddingInput { product_id })
        .unwrap();

    assert_eq!(repository.deleted.lock().unwrap().as_slice(), &[product_id]);
    assert_eq!(
        metrics.increments.lock().unwrap().as_slice(),
        &[("embedding.product_embeddings.deleted".into(), 1)]
    );
}

#[test]
fn delete_store_embedding_deletes_products_before_store() {
    let calls = Arc::new(Mutex::new(Vec::new()));
    let products = Arc::new(FakeProductRepository {
        calls: Some(calls.clone()),
        ..Default::default()
    });
    let stores = Arc::new(FakeStoreRepository {
        calls: Some(calls.clone()),
        ..Default::default()
    });
    let metrics = Arc::new(FakeMetrics::default());
    let store_id = Uuid::new_v4();

    let use_case = DeleteStoreEmbeddingUseCase::new(
        products.clone(),
        stores.clone(),
        Arc::new(FakeLogger),
        metrics.clone(),
    );

    use_case
        .execute(DeleteStoreEmbeddingInput { store_id })
        .unwrap();

    assert_eq!(calls.lock().unwrap().as_slice(), &["products", "store"]);
    assert_eq!(
        products.deleted_by_store.lock().unwrap().as_slice(),
        &[store_id]
    );
    assert_eq!(stores.deleted.lock().unwrap().as_slice(), &[store_id]);
    assert_eq!(
        metrics.increments.lock().unwrap().as_slice(),
        &[("embedding.store_embeddings.deleted".into(), 1)]
    );
}

#[test]
fn delete_store_embedding_does_not_delete_store_when_product_deletion_fails() {
    let calls = Arc::new(Mutex::new(Vec::new()));
    let products = Arc::new(FakeProductRepository {
        calls: Some(calls.clone()),
        fail_delete_by_store: true,
        ..Default::default()
    });
    let stores = Arc::new(FakeStoreRepository {
        calls: Some(calls.clone()),
        ..Default::default()
    });
    let metrics = Arc::new(FakeMetrics::default());

    let use_case = DeleteStoreEmbeddingUseCase::new(
        products,
        stores.clone(),
        Arc::new(FakeLogger),
        metrics.clone(),
    );

    let result = use_case.execute(DeleteStoreEmbeddingInput {
        store_id: Uuid::new_v4(),
    });

    assert!(result.is_err());
    assert_eq!(calls.lock().unwrap().as_slice(), &["products"]);
    assert!(stores.deleted.lock().unwrap().is_empty());
    assert!(metrics.increments.lock().unwrap().is_empty());
}

#[test]
fn ingest_knowledge_embeds_all_chunks_before_replacing_existing_chunks() {
    let reader = Arc::new(FakeDocumentReader::new("Source document"));
    let model = Arc::new(FakeEmbeddingModel::new());
    let chunker = Arc::new(FakeChunker::new(vec![" First chunk ", "Second chunk"]));
    let repository = Arc::new(FakeKnowledgeRepository::default());
    let metrics = Arc::new(FakeMetrics::default());
    let knowledge_id = Uuid::new_v4();
    let store_id = Uuid::new_v4();

    let use_case = ingestion_use_case(
        reader.clone(),
        model.clone(),
        chunker.clone(),
        repository.clone(),
        metrics.clone(),
    );

    let output = use_case
        .execute(IngestKnowledgeInput {
            knowledge_id,
            store_id,
            storage_reference: "  documents/guide.pdf  ".into(),
        })
        .unwrap();

    assert_eq!(output.knowledge_id, knowledge_id);
    assert_eq!(output.store_id, store_id);
    assert_eq!(output.chunk_count, 2);
    assert_eq!(output.embedding_model_versions.len(), 2);

    assert_eq!(
        reader.references.lock().unwrap().as_slice(),
        &["documents/guide.pdf"]
    );
    assert_eq!(
        chunker.calls.lock().unwrap().as_slice(),
        &["Source document"]
    );
    assert_eq!(
        model.generated_texts.lock().unwrap().as_slice(),
        &["First chunk", "Second chunk"]
    );

    let replaced = repository.replaced.lock().unwrap();
    assert_eq!(replaced.len(), 1);
    assert_eq!(replaced[0].0, knowledge_id);
    assert_eq!(replaced[0].1.len(), 2);

    assert_eq!(
        metrics.increments.lock().unwrap().as_slice(),
        &[("embedding.knowledge.documents.ingested".into(), 1)]
    );
    assert_eq!(
        metrics.observations.lock().unwrap().as_slice(),
        &[("embedding.knowledge.chunks.ingested".into(), 2.0)]
    );
}

#[test]
fn ingest_knowledge_rejects_blank_document_without_chunking_or_replacing() {
    let reader = Arc::new(FakeDocumentReader::new("   "));
    let model = Arc::new(FakeEmbeddingModel::new());
    let chunker = Arc::new(FakeChunker::new(vec!["chunk"]));
    let repository = Arc::new(FakeKnowledgeRepository::default());
    let metrics = Arc::new(FakeMetrics::default());

    let use_case = ingestion_use_case(
        reader,
        model.clone(),
        chunker.clone(),
        repository.clone(),
        metrics.clone(),
    );

    let result = use_case.execute(IngestKnowledgeInput {
        knowledge_id: Uuid::new_v4(),
        store_id: Uuid::new_v4(),
        storage_reference: "documents/guide.pdf".into(),
    });

    assert!(result.is_err());
    assert!(chunker.calls.lock().unwrap().is_empty());
    assert!(model.generated_texts.lock().unwrap().is_empty());
    assert!(repository.replaced.lock().unwrap().is_empty());
    assert!(metrics.increments.lock().unwrap().is_empty());
}

#[test]
fn ingest_knowledge_rejects_empty_chunk_list_without_replacing_existing_chunks() {
    let reader = Arc::new(FakeDocumentReader::new("Source document"));
    let model = Arc::new(FakeEmbeddingModel::new());
    let chunker = Arc::new(FakeChunker::new(vec![]));
    let repository = Arc::new(FakeKnowledgeRepository::default());
    let metrics = Arc::new(FakeMetrics::default());

    let use_case = ingestion_use_case(reader, model, chunker, repository.clone(), metrics.clone());

    let result = use_case.execute(IngestKnowledgeInput {
        knowledge_id: Uuid::new_v4(),
        store_id: Uuid::new_v4(),
        storage_reference: "documents/guide.pdf".into(),
    });

    assert!(result.is_err());
    assert!(repository.replaced.lock().unwrap().is_empty());
    assert!(metrics.increments.lock().unwrap().is_empty());
}

#[test]
fn ingest_knowledge_does_not_replace_existing_chunks_if_embedding_generation_fails() {
    let reader = Arc::new(FakeDocumentReader::new("Source document"));
    let model = Arc::new(FakeEmbeddingModel::failing_on_call(2));
    let chunker = Arc::new(FakeChunker::new(vec!["First chunk", "Second chunk"]));
    let repository = Arc::new(FakeKnowledgeRepository::default());
    let metrics = Arc::new(FakeMetrics::default());

    let use_case = ingestion_use_case(
        reader,
        model.clone(),
        chunker,
        repository.clone(),
        metrics.clone(),
    );

    let result = use_case.execute(IngestKnowledgeInput {
        knowledge_id: Uuid::new_v4(),
        store_id: Uuid::new_v4(),
        storage_reference: "documents/guide.pdf".into(),
    });

    assert!(result.is_err());
    assert_eq!(model.generated_texts.lock().unwrap().len(), 2);
    assert!(repository.replaced.lock().unwrap().is_empty());
    assert!(metrics.increments.lock().unwrap().is_empty());
}

#[test]
fn ingest_knowledge_rejects_blank_chunk_without_replacing_existing_chunks() {
    let reader = Arc::new(FakeDocumentReader::new("Source document"));
    let model = Arc::new(FakeEmbeddingModel::new());
    let chunker = Arc::new(FakeChunker::new(vec!["First chunk", "   "]));
    let repository = Arc::new(FakeKnowledgeRepository::default());
    let metrics = Arc::new(FakeMetrics::default());

    let use_case = ingestion_use_case(
        reader,
        model.clone(),
        chunker,
        repository.clone(),
        metrics.clone(),
    );

    let result = use_case.execute(IngestKnowledgeInput {
        knowledge_id: Uuid::new_v4(),
        store_id: Uuid::new_v4(),
        storage_reference: "documents/guide.pdf".into(),
    });

    assert!(result.is_err());
    assert_eq!(model.generated_texts.lock().unwrap().len(), 1);
    assert!(repository.replaced.lock().unwrap().is_empty());
    assert!(metrics.increments.lock().unwrap().is_empty());
}
