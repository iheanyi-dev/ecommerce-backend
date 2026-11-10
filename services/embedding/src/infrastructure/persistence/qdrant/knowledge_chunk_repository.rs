//! Qdrant persistence adapter for knowledge chunks.

use std::{
    collections::{HashMap, HashSet},
    error::Error,
    io,
};

use qdrant_client::{
    Payload, Qdrant,
    qdrant::{
        Condition, DeletePointsBuilder, Filter, PointId, PointStruct, QueryPointsBuilder,
        ScrollPointsBuilder, UpsertPointsBuilder, point_id, points_selector, vector_output,
        vectors_output,
    },
};
use serde_json::{Value, json};
use uuid::Uuid;

use crate::{
    application::{
        dto::ScoredKnowledgeChunk,
        ports::{KnowledgeChunkRepository, PortResult},
    },
    domain::{
        entities::KnowledgeChunk,
        value_objects::{
            ChunkIndex, Embedding, EmbeddingModelVersion, KnowledgeChunkContent, StorageReference,
        },
    },
};

/// Qdrant adapter implementing the knowledge chunk repository port.
///
/// The application port is synchronous while Qdrant's client is asynchronous.
/// This adapter owns a Tokio runtime and uses it to execute Qdrant operations.
pub struct QdrantKnowledgeChunkRepository {
    client: Qdrant,
    collection_name: String,
    runtime: tokio::runtime::Runtime,
}

impl QdrantKnowledgeChunkRepository {
    pub fn new(client: Qdrant, collection_name: impl Into<String>) -> PortResult<Self> {
        let collection_name = collection_name.into();

        if collection_name.trim().is_empty() {
            return Err(invalid_input("Qdrant collection name must not be empty"));
        }

        let runtime = tokio::runtime::Builder::new_multi_thread()
            .enable_all()
            .build()?;

        Ok(Self {
            client,
            collection_name,
            runtime,
        })
    }

    fn knowledge_filter(knowledge_id: Uuid) -> Filter {
        Filter {
            must: vec![Condition::matches("knowledge_id", knowledge_id.to_string())],
            ..Default::default()
        }
    }

    fn store_filter(store_id: Uuid) -> Filter {
        Filter {
            must: vec![Condition::matches("store_id", store_id.to_string())],
            ..Default::default()
        }
    }

    /// Retrieves every existing point ID for a knowledge document.
    ///
    /// Qdrant scroll pagination is followed until no next-page offset remains.
    async fn get_point_ids_for_knowledge(&self, knowledge_id: Uuid) -> PortResult<HashSet<Uuid>> {
        let mut ids = HashSet::new();
        let mut offset: Option<PointId> = None;

        loop {
            let mut request = ScrollPointsBuilder::new(self.collection_name.clone())
                .filter(Self::knowledge_filter(knowledge_id))
                .limit(1000)
                .with_payload(false)
                .with_vectors(false);

            if let Some(next_offset) = offset.take() {
                request = request.offset(next_offset);
            }

            let response = self.client.scroll(request).await?;

            for point in response.result {
                let point_id = point
                    .id
                    .ok_or_else(|| invalid_data("Qdrant point is missing its ID"))?;

                if let Some(id) = uuid_from_point_id(point_id) {
                    ids.insert(id);
                }
            }

            match response.next_page_offset {
                Some(next_offset) => offset = Some(next_offset),
                None => break,
            }
        }

        Ok(ids)
    }

    async fn delete_point_ids(&self, ids: Vec<Uuid>) -> PortResult<()> {
        if ids.is_empty() {
            return Ok(());
        }

        let point_ids = ids
            .into_iter()
            .map(|id| PointId {
                point_id_options: Some(point_id::PointIdOptions::Uuid(id.to_string())),
            })
            .collect();

        let selector =
            points_selector::PointsSelectorOneOf::Points(qdrant_client::qdrant::PointsIdsList {
                ids: point_ids,
            });

        self.client
            .delete_points(
                DeletePointsBuilder::new(self.collection_name.clone())
                    .points(selector)
                    .wait(true),
            )
            .await?;

        Ok(())
    }
}

impl KnowledgeChunkRepository for QdrantKnowledgeChunkRepository {
    fn replace_for_knowledge(
        &self,
        knowledge_id: Uuid,
        chunks: Vec<KnowledgeChunk>,
    ) -> PortResult<()> {
        if chunks.is_empty() {
            return self.delete_by_knowledge_id(knowledge_id);
        }

        let store_id = chunks[0].store_id();

        if chunks.iter().any(|chunk| chunk.store_id() != store_id) {
            return Err(invalid_input(
                "All replacement chunks must belong to one store",
            ));
        }

        if chunks
            .iter()
            .any(|chunk| chunk.knowledge_id() != knowledge_id)
        {
            return Err(invalid_input(
                "Every replacement chunk must match the requested knowledge ID",
            ));
        }

        let replacement_ids: HashSet<Uuid> = chunks.iter().map(KnowledgeChunk::id).collect();

        let points = chunks
            .iter()
            .map(|chunk| {
                let payload = Payload::try_from(json!({
                    "knowledge_id": chunk.knowledge_id().to_string(),
                    "store_id": chunk.store_id().to_string(),
                    "storage_reference": chunk.storage_reference().value(),
                    "content": chunk.content().value(),
                    "chunk_index": chunk.chunk_index().value(),
                    "embedding_model_version": chunk.embedding_model_version().value(),
                }))?;

                Ok(PointStruct::new(
                    chunk.id().to_string(),
                    chunk.embedding().vector().to_vec(),
                    payload,
                ))
            })
            .collect::<Result<Vec<_>, qdrant_client::QdrantError>>()?;

        let existing_ids = self
            .runtime
            .block_on(self.get_point_ids_for_knowledge(knowledge_id))?;

        // Upsert before deleting stale points so a failed upsert does not
        // remove the currently persisted chunks.
        self.runtime.block_on(async {
            self.client
                .upsert_points(
                    UpsertPointsBuilder::new(self.collection_name.clone(), points).wait(true),
                )
                .await?;

            let stale_ids = existing_ids
                .difference(&replacement_ids)
                .copied()
                .collect::<Vec<_>>();

            self.delete_point_ids(stale_ids).await
        })
    }

    fn delete_by_knowledge_id(&self, knowledge_id: Uuid) -> PortResult<()> {
        self.runtime.block_on(async {
            self.client
                .delete_points(
                    DeletePointsBuilder::new(self.collection_name.clone())
                        .points(Self::knowledge_filter(knowledge_id))
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
        store_id: Uuid,
    ) -> PortResult<Vec<ScoredKnowledgeChunk>> {
        if top_k == 0 {
            return Err(invalid_input("top_k must be a positive integer"));
        }

        let response = self.runtime.block_on(async {
            self.client
                .query(
                    QueryPointsBuilder::new(self.collection_name.clone())
                        .query(query_embedding.vector().to_vec())
                        .filter(Self::store_filter(store_id))
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
            .map(scored_chunk_from_point)
            .collect()
    }
}

fn scored_chunk_from_point(
    point: qdrant_client::qdrant::ScoredPoint,
) -> PortResult<ScoredKnowledgeChunk> {
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

    let knowledge_id = Uuid::parse_str(&required_string("knowledge_id")?)?;
    let store_id = Uuid::parse_str(&required_string("store_id")?)?;

    let storage_reference = StorageReference::new(required_string("storage_reference")?)?;
    let content = KnowledgeChunkContent::new(required_string("content")?)?;

    let chunk_index = payload
        .get("chunk_index")
        .and_then(Value::as_u64)
        .and_then(|value| i64::try_from(value).ok())
        .ok_or_else(|| invalid_data("Qdrant payload has an invalid chunk_index"))?;

    let chunk_index = ChunkIndex::new(chunk_index)?;
    let embedding_model_version =
        EmbeddingModelVersion::new(required_string("embedding_model_version")?)?;

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

    let embedding = Embedding::new(vector)?;

    let chunk = KnowledgeChunk::new(
        id,
        knowledge_id,
        store_id,
        storage_reference,
        content,
        chunk_index,
        embedding,
        embedding_model_version,
    );

    Ok(ScoredKnowledgeChunk {
        chunk,
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
        let result = QdrantKnowledgeChunkRepository::new(
            Qdrant::from_url("http://localhost:6334").build().unwrap(),
            "   ",
        );

        assert!(result.is_err());
    }

    #[test]
    fn builds_knowledge_filter() {
        let id = Uuid::new_v4();
        let filter = QdrantKnowledgeChunkRepository::knowledge_filter(id);

        assert_eq!(filter.must.len(), 1);
        assert_eq!(filter.must[0].condition_one_of.is_some(), true);
    }

    #[test]
    fn builds_store_filter() {
        let id = Uuid::new_v4();
        let filter = QdrantKnowledgeChunkRepository::store_filter(id);

        assert_eq!(filter.must.len(), 1);
        assert_eq!(filter.must[0].condition_one_of.is_some(), true);
    }

    #[test]
    fn parses_uuid_point_ids() {
        let id = Uuid::new_v4();
        let point_id = PointId {
            point_id_options: Some(point_id::PointIdOptions::Uuid(id.to_string())),
        };

        assert_eq!(uuid_from_point_id(point_id), Some(id));
    }

    #[test]
    fn rejects_numeric_point_ids() {
        let point_id = PointId {
            point_id_options: Some(point_id::PointIdOptions::Num(42)),
        };

        assert_eq!(uuid_from_point_id(point_id), None);
    }
}
