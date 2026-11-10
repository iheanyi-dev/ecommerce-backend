//! Environment-backed service configuration.

use std::{
    env,
    error::Error,
    net::SocketAddr,
    path::{Path, PathBuf},
};

pub struct Settings {
    pub storage_address: String,
    pub storage_service_name: String,
    pub storage_service_secret: String,
    pub grpc_address: SocketAddr,
    pub grpc_caller_service_name: String,
    pub grpc_caller_service_secret: String,
    pub qdrant_url: String,
    pub qdrant_api_key: Option<String>,
    pub kafka_brokers: String,
    pub kafka_group_id: String,
    pub model_cache_dir: PathBuf,
}

impl Settings {
    pub fn load() -> Result<Self, Box<dyn Error + Send + Sync>> {
        let service_root = Path::new(env!("CARGO_MANIFEST_DIR"));
        let env_path = service_root.join(".env");

        if env_path.exists() {
            dotenvy::from_path(&env_path)?;
        }

        let storage_address = required("EMBEDDING_STORAGE_SERVICE_ADDR")?;
        let storage_secret = required("EMBEDDING_STORAGE_SERVICE_SECRET")?;

        let storage_address =
            if storage_address.starts_with("http://") || storage_address.starts_with("https://") {
                storage_address
            } else {
                format!("http://{storage_address}")
            };

        let grpc_address: SocketAddr = value("EMBEDDING_GRPC_ADDR", "0.0.0.0:8092")?.parse()?;

        let grpc_caller_service_name = value("EMBEDDING_CALLER_SERVICE_NAME", "chat")?;
        let grpc_caller_service_secret = required("EMBEDDING_CALLER_SERVICE_SECRET")?;

        let qdrant_url = value("EMBEDDING_QDRANT_URL", "http://localhost:6334")?;

        let qdrant_api_key = optional("EMBEDDING_QDRANT_API_KEY");

        let kafka_brokers = value("EMBEDDING_KAFKA_BOOTSTRAP_SERVERS", "localhost:19092")?;
        let kafka_group_id = value("EMBEDDING_KAFKA_GROUP_ID", "embedding")?;

        let configured_cache =
            PathBuf::from(value("EMBEDDING_MODEL_CACHE_DIR", ".cache/models/hub")?);

        let model_cache_dir = if configured_cache.is_absolute() {
            configured_cache
        } else {
            service_root.join(configured_cache)
        };

        Ok(Self {
            storage_address,
            storage_service_name: value("EMBEDDING_STORAGE_SERVICE_NAME", "embedding")?,
            storage_service_secret: storage_secret,
            grpc_address,
            grpc_caller_service_name,
            grpc_caller_service_secret,
            qdrant_url,
            qdrant_api_key,
            kafka_brokers,
            kafka_group_id,
            model_cache_dir,
        })
    }
}

fn required(key: &str) -> Result<String, Box<dyn Error + Send + Sync>> {
    let value = optional(key)
        .filter(|value| !value.is_empty())
        .ok_or_else(|| format!("{key} must be configured"))?;

    Ok(value)
}

fn optional(key: &str) -> Option<String> {
    env::var(key)
        .ok()
        .map(|value| value.trim().to_owned())
        .filter(|value| !value.is_empty())
}

fn value(key: &str, default: &str) -> Result<String, Box<dyn Error + Send + Sync>> {
    Ok(optional(key).unwrap_or_else(|| default.to_owned()))
}
