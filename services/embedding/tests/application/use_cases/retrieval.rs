//! Behaviour tests for semantic retrieval use cases.

use std::sync::{Arc, Mutex};

use embedding::{
    application::{
        dto::{
            RetrieveKnowledgeInput, RetrieveProductEmbeddingsInput, RetrieveStoreEmbeddingsInput,
            ScoredProductEmbedding, ScoredStoreEmbedding,
        },
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
};
use uuid::Uuid;

#[derive(Default)]
struct FakeEmbeddingModel {
    generated_texts: Mutex<Vec<String>>,
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
struct FakeKnowledgeRepository {
    searches: Mutex<Vec<(usize, usize, Uuid)>>,
}

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
        top_k: usize,
        offset: usize,
        store_id: Uuid,
    ) -> PortResult<Vec<embedding::application::dto::ScoredKnowledgeChunk>> {
        self.searches
            .lock()
            .unwrap()
            .push((top_k, offset, store_id));

        Ok(Vec::new())
    }
}

#[derive(Default)]
struct FakeProductRepository {
    searches: Mutex<Vec<(usize, usize, Option<Uuid>)>>,
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
        self.searches
            .lock()
            .unwrap()
            .push((top_k, offset, store_id));

        Ok(Vec::new())
    }
}

#[derive(Default)]
struct FakeStoreRepository {
    searches: Mutex<Vec<(usize, usize)>>,
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
        self.searches.lock().unwrap().push((top_k, offset));

        Ok(Vec::new())
    }
}

#[test]
fn retrieve_knowledge_trims_query_and_searches_with_store_scope_and_pagination() {
    let model = Arc::new(FakeEmbeddingModel::default());
    let repository = Arc::new(FakeKnowledgeRepository::default());
    let metrics = Arc::new(FakeMetrics::default());
    let store_id = Uuid::new_v4();

    let use_case = RetrieveKnowledgeUseCase::new(
        repository.clone(),
        model.clone(),
        Arc::new(FakeLogger),
        metrics.clone(),
    );

    let output = use_case
        .execute(RetrieveKnowledgeInput {
            store_id,
            query: "  return policy  ".into(),
            top_k: 5,
            offset: 10,
        })
        .unwrap();

    assert_eq!(output.store_id, store_id);
    assert!(output.chunks.is_empty());
    assert_eq!(
        model.generated_texts.lock().unwrap().as_slice(),
        &["return policy"]
    );
    assert_eq!(
        repository.searches.lock().unwrap().as_slice(),
        &[(5, 10, store_id)]
    );
    assert_eq!(
        metrics.increments.lock().unwrap().as_slice(),
        &[("knowledge.retrieval.succeeded".into(), 1)]
    );
    assert_eq!(
        metrics.observations.lock().unwrap().as_slice(),
        &[("knowledge.retrieval.result_count".into(), 0.0)]
    );
}

#[test]
fn retrieve_knowledge_rejects_blank_query_without_calling_dependencies() {
    let model = Arc::new(FakeEmbeddingModel::default());
    let repository = Arc::new(FakeKnowledgeRepository::default());
    let metrics = Arc::new(FakeMetrics::default());

    let use_case = RetrieveKnowledgeUseCase::new(
        repository.clone(),
        model.clone(),
        Arc::new(FakeLogger),
        metrics.clone(),
    );

    let result = use_case.execute(RetrieveKnowledgeInput {
        store_id: Uuid::new_v4(),
        query: "   ".into(),
        top_k: 5,
        offset: 0,
    });

    assert!(result.is_err());
    assert!(model.generated_texts.lock().unwrap().is_empty());
    assert!(repository.searches.lock().unwrap().is_empty());
    assert!(metrics.increments.lock().unwrap().is_empty());
}

#[test]
fn retrieve_product_embeddings_passes_optional_store_filter_and_pagination() {
    let model = Arc::new(FakeEmbeddingModel::default());
    let repository = Arc::new(FakeProductRepository::default());
    let metrics = Arc::new(FakeMetrics::default());
    let store_id = Uuid::new_v4();

    let use_case = RetrieveProductEmbeddingsUseCase::new(
        model.clone(),
        repository.clone(),
        Arc::new(FakeLogger),
        metrics.clone(),
    );

    let output = use_case
        .execute(RetrieveProductEmbeddingsInput {
            query: "  running shoes  ".into(),
            top_k: 8,
            offset: 4,
            store_id: Some(store_id),
        })
        .unwrap();

    assert!(output.results.is_empty());
    assert_eq!(
        model.generated_texts.lock().unwrap().as_slice(),
        &["running shoes"]
    );
    assert_eq!(
        repository.searches.lock().unwrap().as_slice(),
        &[(8, 4, Some(store_id))]
    );
    assert_eq!(
        metrics.increments.lock().unwrap().as_slice(),
        &[("embedding.product_embeddings.retrieved".into(), 1)]
    );
    assert_eq!(
        metrics.observations.lock().unwrap().as_slice(),
        &[("embedding.product_embeddings.retrieval_results".into(), 0.0)]
    );
}

#[test]
fn retrieve_product_embeddings_allows_search_without_store_filter() {
    let model = Arc::new(FakeEmbeddingModel::default());
    let repository = Arc::new(FakeProductRepository::default());

    let use_case = RetrieveProductEmbeddingsUseCase::new(
        model,
        repository.clone(),
        Arc::new(FakeLogger),
        Arc::new(FakeMetrics::default()),
    );

    use_case
        .execute(RetrieveProductEmbeddingsInput {
            query: "laptop".into(),
            top_k: 10,
            offset: 0,
            store_id: None,
        })
        .unwrap();

    assert_eq!(
        repository.searches.lock().unwrap().as_slice(),
        &[(10, 0, None)]
    );
}

#[test]
fn retrieve_product_embeddings_rejects_invalid_query_and_top_k() {
    let model = Arc::new(FakeEmbeddingModel::default());
    let repository = Arc::new(FakeProductRepository::default());

    let use_case = RetrieveProductEmbeddingsUseCase::new(
        model.clone(),
        repository.clone(),
        Arc::new(FakeLogger),
        Arc::new(FakeMetrics::default()),
    );

    let blank_query = use_case.execute(RetrieveProductEmbeddingsInput {
        query: "  ".into(),
        top_k: 5,
        offset: 0,
        store_id: None,
    });

    let zero_top_k = use_case.execute(RetrieveProductEmbeddingsInput {
        query: "laptop".into(),
        top_k: 0,
        offset: 0,
        store_id: None,
    });

    assert!(blank_query.is_err());
    assert!(zero_top_k.is_err());
    assert!(model.generated_texts.lock().unwrap().is_empty());
    assert!(repository.searches.lock().unwrap().is_empty());
}

#[test]
fn retrieve_store_embeddings_trims_query_and_passes_pagination() {
    let model = Arc::new(FakeEmbeddingModel::default());
    let repository = Arc::new(FakeStoreRepository::default());
    let metrics = Arc::new(FakeMetrics::default());

    let use_case = RetrieveStoreEmbeddingsUseCase::new(
        model.clone(),
        repository.clone(),
        Arc::new(FakeLogger),
        metrics.clone(),
    );

    let output = use_case
        .execute(RetrieveStoreEmbeddingsInput {
            query: "  electronics retailer  ".into(),
            top_k: 6,
            offset: 3,
        })
        .unwrap();

    assert!(output.results.is_empty());
    assert_eq!(
        model.generated_texts.lock().unwrap().as_slice(),
        &["electronics retailer"]
    );
    assert_eq!(repository.searches.lock().unwrap().as_slice(), &[(6, 3)]);
    assert_eq!(
        metrics.increments.lock().unwrap().as_slice(),
        &[("embedding.store_embeddings.retrieved".into(), 1)]
    );
    assert_eq!(
        metrics.observations.lock().unwrap().as_slice(),
        &[("embedding.store_embeddings.retrieval_results".into(), 0.0)]
    );
}

#[test]
fn retrieve_store_embeddings_rejects_invalid_query_and_top_k() {
    let model = Arc::new(FakeEmbeddingModel::default());
    let repository = Arc::new(FakeStoreRepository::default());

    let use_case = RetrieveStoreEmbeddingsUseCase::new(
        model.clone(),
        repository.clone(),
        Arc::new(FakeLogger),
        Arc::new(FakeMetrics::default()),
    );

    let blank_query = use_case.execute(RetrieveStoreEmbeddingsInput {
        query: "   ".into(),
        top_k: 5,
        offset: 0,
    });

    let zero_top_k = use_case.execute(RetrieveStoreEmbeddingsInput {
        query: "electronics".into(),
        top_k: 0,
        offset: 0,
    });

    assert!(blank_query.is_err());
    assert!(zero_top_k.is_err());
    assert!(model.generated_texts.lock().unwrap().is_empty());
    assert!(repository.searches.lock().unwrap().is_empty());
}
