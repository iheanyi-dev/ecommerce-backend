//! Qdrant persistence adapter for store embeddings.

use std::{collections::HashMap, error::Error, io};

use qdrant_client::{
    Payload, Qdrant,
    qdrant::{
        DeletePointsBuilder, PointId, PointStruct, QueryPointsBuilder, UpsertPointsBuilder,
        point_id, points_selector, vector_output, vectors_output,
    },
};
use serde_json::{Value, json};
use uuid::Uuid;

use crate::{
    application::{
        dto::ScoredStoreEmbedding,
        ports::{PortResult, StoreEmbeddingRepository},
    },
    domain::{
        entities::StoreEmbedding,
        value_objects::{Embedding, EmbeddingModelVersion, EntityDescription, EntityName},
    },
};

pub struct QdrantStoreEmbeddingRepository {
    client: Qdrant,
    collection_name: String,
    runtime: Option<tokio::runtime::Runtime>,
}

impl QdrantStoreEmbeddingRepository {
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
}

impl StoreEmbeddingRepository for QdrantStoreEmbeddingRepository {
    fn upsert(&self, embedding: StoreEmbedding) -> PortResult<()> {
        let payload = Payload::try_from(json!({
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

    fn delete(&self, store_id: Uuid) -> PortResult<()> {
        let selector =
            points_selector::PointsSelectorOneOf::Points(qdrant_client::qdrant::PointsIdsList {
                ids: vec![PointId {
                    point_id_options: Some(point_id::PointIdOptions::Uuid(store_id.to_string())),
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

    fn search(
        &self,
        query_embedding: &Embedding,
        top_k: usize,
        offset: usize,
    ) -> PortResult<Vec<ScoredStoreEmbedding>> {
        if top_k == 0 {
            return Err(invalid_input("top_k must be a positive integer"));
        }

        let response = self
            .runtime
            .as_ref()
            .expect("runtime already shut down")
            .block_on(async {
                self.client
                    .query(
                        QueryPointsBuilder::new(self.collection_name.clone())
                            .query(query_embedding.vector().to_vec())
                            .limit(top_k as u64)
                            .offset(offset as u64)
                            .with_payload(true)
                            .with_vectors(true),
                    )
                    .await
            })?;

        response
            .result
            .into_iter()
            .map(scored_store_from_point)
            .collect()
    }
}

fn scored_store_from_point(
    point: qdrant_client::qdrant::ScoredPoint,
) -> PortResult<ScoredStoreEmbedding> {
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

    let embedding = StoreEmbedding::new(
        id,
        EntityName::new(required_string("name")?, "store")?,
        EntityDescription::new(required_string("description")?, "store")?,
        Embedding::new(vector)?,
        EmbeddingModelVersion::new(required_string("embedding_model_version")?)?,
    );

    Ok(ScoredStoreEmbedding {
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
        assert!(QdrantStoreEmbeddingRepository::new(client, "  ").is_err());
    }

    #[test]
    fn rejects_zero_top_k() {
        let client = Qdrant::from_url("http://localhost:6334").build().unwrap();
        let repository = QdrantStoreEmbeddingRepository::new(client, "stores").unwrap();
        let query = Embedding::new(vec![0.1, 0.2]).unwrap();

        assert_eq!(
            repository.search(&query, 0, 0).unwrap_err().to_string(),
            "top_k must be a positive integer"
        );
    }
}

// Runtime shutdown must happen outside any asynchronous Tokio context.
impl Drop for QdrantStoreEmbeddingRepository {
    fn drop(&mut self) {
        if let Some(runtime) = self.runtime.take() {
            let _ = std::thread::Builder::new()
                .name("embedding-runtime-shutdown".to_owned())
                .spawn(move || drop(runtime));
        }
    }
}
