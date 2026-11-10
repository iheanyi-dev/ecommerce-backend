//! gRPC presentation adapter.

pub mod embedding_service;
pub mod server;

pub mod pb {
    tonic::include_proto!("embedding.v1");
}

pub mod tracing_interceptor;
