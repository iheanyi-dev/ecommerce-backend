//! Qdrant persistence adapter for product embeddings.

use std::{collections::HashMap, error::Error, io};

use qdrant_client::{
    Payload, Qdrant,
    qdrant::{
        Condition, DeletePointsBuilder, Filter, PointId, PointStruct, QueryPointsBuilder,
        UpsertPointsBuilder, point_id, points_selector, vector_output, vectors_output,
    },
};
use serde_json::{Value, json};
use uuid::Uuid;

use crate::{
    application::{
        dto::ScoredProductEmbedding,
        ports::{PortResult, ProductEmbeddingRepository},
    },
    domain::{
        entities::ProductEmbedding,
        value_objects::{Embedding, EmbeddingModelVersion, EntityDescription, EntityName},
    },
};

pub struct QdrantProductEmbeddingRepository {
    client: Qdrant,
    collection_name: String,
    runtime: Option<tokio::runtime::Runtime>,
}

impl QdrantProductEmbeddingRepository {
    pub fn new(client: Qdrant, collection_name: impl Into<String>) -> PortResult<Self> {
        let collection_name = collection_name.into();

        if collection_name.trim().is_empty() {
            return Err(invalid_input("Qdrant collection name must not be empty"));
        }

        Ok(Self {
            client,
            collection_name,
            runtime: Some(
                tokio::runtime::Builder::new_multi_thread()
                    .enable_all()
                    .build()?,
            ),
        })
    }

    fn store_filter(store_id: Uuid) -> Filter {
        Filter {
            must: vec![Condition::matches("store_id", store_id.to_string())],
            ..Default::default()
        }
    }
}

impl ProductEmbeddingRepository for QdrantProductEmbeddingRepository {
    fn upsert(&self, embedding: ProductEmbedding) -> PortResult<()> {
        let payload = Payload::try_from(json!({
            "store_id": embedding.store_id().to_string(),
            "name": embedding.name().value(),
            "description": embedding.description().value(),
            "embedding_model_version": embedding.embedding_model_version().value(),
        }))?;

        let point = PointStruct::new(
            embedding.id().to_string(),
            embedding.embedding().vector().to_vec(),
            payload,
        );

        self.runtime
            .as_ref()
            .expect("runtime already shut down")
            .block_on(async {
                self.client
                    .upsert_points(
                        UpsertPointsBuilder::new(self.collection_name.clone(), vec![point])
                            .wait(true),
                    )
                    .await?;
                Ok(())
            })
    }

    fn delete(&self, product_id: Uuid) -> PortResult<()> {
        let selector =
            points_selector::PointsSelectorOneOf::Points(qdrant_client::qdrant::PointsIdsList {
                ids: vec![PointId {
                    point_id_options: Some(point_id::PointIdOptions::Uuid(product_id.to_string())),
                }],
            });

        self.runtime
            .as_ref()
            .expect("runtime already shut down")
            .block_on(async {
                self.client
                    .delete_points(
                        DeletePointsBuilder::new(self.collection_name.clone())
                            .points(selector)
                            .wait(true),
                    )
                    .await?;
                Ok(())
            })
    }

    fn delete_by_store_id(&self, store_id: Uuid) -> PortResult<()> {
        self.runtime
            .as_ref()
            .expect("runtime already shut down")
            .block_on(async {
                self.client
                    .delete_points(
                        DeletePointsBuilder::new(self.collection_name.clone())
                            .points(Self::store_filter(store_id))
                            .wait(true),
                    )
                    .await?;
                Ok(())
            })
    }

    fn search(
        &self,
        query_embedding: &Embedding,
        top_k: usize,
        offset: usize,
        store_id: Option<Uuid>,
    ) -> PortResult<Vec<ScoredProductEmbedding>> {
        if top_k == 0 {
            return Err(invalid_input("top_k must be a positive integer"));
        }

        let mut request = QueryPointsBuilder::new(self.collection_name.clone())
            .query(query_embedding.vector().to_vec())
            .limit(top_k as u64)
            .offset(offset as u64)
            .with_payload(true)
            .with_vectors(true);

        if let Some(store_id) = store_id {
            request = request.filter(Self::store_filter(store_id));
        }

        let response = self
            .runtime
            .as_ref()
            .expect("runtime already shut down")
            .block_on(async { self.client.query(request).await })?;

        response
            .result
            .into_iter()
            .map(scored_product_from_point)
            .collect()
    }
}

fn scored_product_from_point(
    point: qdrant_client::qdrant::ScoredPoint,
) -> PortResult<ScoredProductEmbedding> {
    let point_id = point
        .id
        .ok_or_else(|| invalid_data("Qdrant point is missing its ID"))?;

    let id = uuid_from_point_id(point_id)
        .ok_or_else(|| invalid_data("Qdrant point ID is not a UUID"))?;

    let payload = point
        .payload
        .into_iter()
        .map(|(key, value)| (key, Value::from(value)))
        .collect::<HashMap<_, _>>();

    let required_string = |key: &str| -> PortResult<String> {
        payload
            .get(key)
            .and_then(Value::as_str)
            .map(str::to_owned)
            .ok_or_else(|| invalid_data(format!("Qdrant payload is missing string field '{key}'")))
    };

    let store_id = Uuid::parse_str(&required_string("store_id")?)?;

    let vectors = point
        .vectors
        .ok_or_else(|| invalid_data(format!("Qdrant point {id} is missing vector data")))?;

    let vector = match vectors.vectors_options {
        Some(vectors_output::VectorsOptions::Vector(output)) => match output.into_vector() {
            vector_output::Vector::Dense(dense) => dense.data,
            _ => {
                return Err(invalid_data(format!(
                    "Qdrant point {id} does not contain a dense vector"
                )));
            }
        },
        _ => {
            return Err(invalid_data(format!(
                "Qdrant point {id} is missing vector data"
            )));
        }
    };

    let embedding = ProductEmbedding::new(
        id,
        store_id,
        EntityName::new(required_string("name")?, "product")?,
        EntityDescription::new(required_string("description")?, "product")?,
        Embedding::new(vector)?,
        EmbeddingModelVersion::new(required_string("embedding_model_version")?)?,
    );

    Ok(ScoredProductEmbedding {
        embedding,
        score: point.score,
    })
}

fn uuid_from_point_id(point_id: PointId) -> Option<Uuid> {
    match point_id.point_id_options? {
        point_id::PointIdOptions::Uuid(value) => Uuid::parse_str(&value).ok(),
        point_id::PointIdOptions::Num(_) => None,
    }
}

fn invalid_input(message: impl Into<String>) -> Box<dyn Error + Send + Sync> {
    Box::new(io::Error::new(io::ErrorKind::InvalidInput, message.into()))
}

fn invalid_data(message: impl Into<String>) -> Box<dyn Error + Send + Sync> {
    Box::new(io::Error::new(io::ErrorKind::InvalidData, message.into()))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn rejects_blank_collection_name() {
        let client = Qdrant::from_url("http://localhost:6334").build().unwrap();
        assert!(QdrantProductEmbeddingRepository::new(client, " ").is_err());
    }

    #[test]
    fn builds_store_filter() {
        let store_id = Uuid::new_v4();
        let filter = QdrantProductEmbeddingRepository::store_filter(store_id);

        assert_eq!(filter.must.len(), 1);
        assert!(filter.must[0].condition_one_of.is_some());
    }

    #[test]
    fn rejects_zero_top_k() {
        let client = Qdrant::from_url("http://localhost:6334").build().unwrap();
        let repository = QdrantProductEmbeddingRepository::new(client, "products").unwrap();
        let query = Embedding::new(vec![0.1, 0.2]).unwrap();

        assert_eq!(
            repository
                .search(&query, 0, 0, None)
                .unwrap_err()
                .to_string(),
            "top_k must be a positive integer"
        );
    }
}

// Runtime shutdown must happen outside any asynchronous Tokio context.
impl Drop for QdrantProductEmbeddingRepository {
    fn drop(&mut self) {
        if let Some(runtime) = self.runtime.take() {
            let _ = std::thread::Builder::new()
                .name("embedding-runtime-shutdown".to_owned())
                .spawn(move || drop(runtime));
        }
    }
}
