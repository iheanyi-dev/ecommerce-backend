//! Behaviour tests for Product and Store embedding creation.

use std::sync::{Arc, Mutex};

use embedding::{
    application::{
        dto::{
            CreateProductEmbeddingInput, CreateStoreEmbeddingInput, ScoredProductEmbedding,
            ScoredStoreEmbedding,
        },
        ports::{
            EmbeddingModel, Logger, Metrics, PortResult, ProductEmbeddingRepository,
            StoreEmbeddingRepository,
        },
        use_cases::{
            create_product_embedding::CreateProductEmbeddingUseCase,
            create_store_embedding::CreateStoreEmbeddingUseCase,
        },
    },
    domain::{
        entities::{ProductEmbedding, StoreEmbedding},
        value_objects::{Embedding, EmbeddingModelVersion},
    },
};
use uuid::Uuid;

struct FakeEmbeddingModel {
    generated_texts: Mutex<Vec<String>>,
}

impl FakeEmbeddingModel {
    fn new() -> Self {
        Self {
            generated_texts: Mutex::new(Vec::new()),
        }
    }
}

impl EmbeddingModel for FakeEmbeddingModel {
    fn generate(&self, text: &str) -> PortResult<(Embedding, EmbeddingModelVersion)> {
        self.generated_texts.lock().unwrap().push(text.to_owned());

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
struct FakeProductRepository {
    saved: Mutex<Option<ProductEmbedding>>,
}

impl ProductEmbeddingRepository for FakeProductRepository {
    fn upsert(&self, embedding: ProductEmbedding) -> PortResult<()> {
        *self.saved.lock().unwrap() = Some(embedding);
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
        _: usize,
        _: usize,
        _: Option<Uuid>,
    ) -> PortResult<Vec<ScoredProductEmbedding>> {
        Ok(Vec::new())
    }
}

#[derive(Default)]
struct FakeStoreRepository {
    saved: Mutex<Option<StoreEmbedding>>,
}

impl StoreEmbeddingRepository for FakeStoreRepository {
    fn upsert(&self, embedding: StoreEmbedding) -> PortResult<()> {
        *self.saved.lock().unwrap() = Some(embedding);
        Ok(())
    }

    fn delete(&self, _: Uuid) -> PortResult<()> {
        Ok(())
    }

    fn search(&self, _: &Embedding, _: usize, _: usize) -> PortResult<Vec<ScoredStoreEmbedding>> {
        Ok(Vec::new())
    }
}

#[test]
fn create_product_embedding_trims_input_generates_expected_text_and_persists() {
    let model = Arc::new(FakeEmbeddingModel::new());
    let repository = Arc::new(FakeProductRepository::default());
    let metrics = Arc::new(FakeMetrics::default());

    let use_case = CreateProductEmbeddingUseCase::new(
        model.clone(),
        repository.clone(),
        Arc::new(FakeLogger),
        metrics.clone(),
    );

    let product_id = Uuid::new_v4();
    let store_id = Uuid::new_v4();

    let output = use_case
        .execute(CreateProductEmbeddingInput {
            product_id,
            store_id,
            name: "  Running Shoes  ".into(),
            description: "  Lightweight running shoes  ".into(),
        })
        .unwrap();

    assert_eq!(output.product_id, product_id);
    assert_eq!(
        model.generated_texts.lock().unwrap().as_slice(),
        ["Product name: Running Shoes\nProduct description: Lightweight running shoes"]
    );
    assert!(repository.saved.lock().unwrap().is_some());
    assert_eq!(
        metrics.increments.lock().unwrap().as_slice(),
        [("embedding.product_embeddings.created".into(), 1)]
    );
}

#[test]
fn create_product_embedding_rejects_blank_name_without_calling_dependencies() {
    let model = Arc::new(FakeEmbeddingModel::new());
    let repository = Arc::new(FakeProductRepository::default());
    let metrics = Arc::new(FakeMetrics::default());

    let use_case = CreateProductEmbeddingUseCase::new(
        model.clone(),
        repository.clone(),
        Arc::new(FakeLogger),
        metrics.clone(),
    );

    let result = use_case.execute(CreateProductEmbeddingInput {
        product_id: Uuid::new_v4(),
        store_id: Uuid::new_v4(),
        name: "   ".into(),
        description: "A valid description".into(),
    });

    assert!(result.is_err());
    assert!(model.generated_texts.lock().unwrap().is_empty());
    assert!(repository.saved.lock().unwrap().is_none());
    assert!(metrics.increments.lock().unwrap().is_empty());
}

#[test]
fn create_store_embedding_trims_input_generates_expected_text_and_persists() {
    let model = Arc::new(FakeEmbeddingModel::new());
    let repository = Arc::new(FakeStoreRepository::default());
    let metrics = Arc::new(FakeMetrics::default());

    let use_case = CreateStoreEmbeddingUseCase::new(
        model.clone(),
        repository.clone(),
        Arc::new(FakeLogger),
        metrics.clone(),
    );

    let store_id = Uuid::new_v4();

    let output = use_case
        .execute(CreateStoreEmbeddingInput {
            store_id,
            name: "  Electronics Store  ".into(),
            description: "  Consumer electronics retailer  ".into(),
        })
        .unwrap();

    assert_eq!(output.store_id, store_id);
    assert_eq!(
        model.generated_texts.lock().unwrap().as_slice(),
        ["Store name: Electronics Store\nStore description: Consumer electronics retailer"]
    );
    assert!(repository.saved.lock().unwrap().is_some());
    assert_eq!(
        metrics.increments.lock().unwrap().as_slice(),
        [("embedding.store_embeddings.created".into(), 1)]
    );
}

#[test]
fn create_store_embedding_rejects_blank_description_without_calling_dependencies() {
    let model = Arc::new(FakeEmbeddingModel::new());
    let repository = Arc::new(FakeStoreRepository::default());
    let metrics = Arc::new(FakeMetrics::default());

    let use_case = CreateStoreEmbeddingUseCase::new(
        model.clone(),
        repository.clone(),
        Arc::new(FakeLogger),
        metrics.clone(),
    );

    let result = use_case.execute(CreateStoreEmbeddingInput {
        store_id: Uuid::new_v4(),
        name: "Valid store".into(),
        description: "   ".into(),
    });

    assert!(result.is_err());
    assert!(model.generated_texts.lock().unwrap().is_empty());
    assert!(repository.saved.lock().unwrap().is_none());
    assert!(metrics.increments.lock().unwrap().is_empty());
}
