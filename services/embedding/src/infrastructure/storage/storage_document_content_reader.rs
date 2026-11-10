use std::{error::Error, io};

use futures_util::StreamExt;
use prost::Message;
use tonic::{
    Request,
    codegen::http::uri::PathAndQuery,
    transport::{Channel, Endpoint},
};

use crate::{
    application::ports::{DocumentContentReader, PortResult},
    domain::value_objects::StorageReference,
};

const GET_PRIVATE_OBJECT_PATH: &str = "/storage.v1.StorageService/GetPrivateObject";

#[derive(Clone, PartialEq, Message)]
struct GetPrivateObjectRequest {
    #[prost(string, tag = "1")]
    reference: String,
}

#[derive(Clone, PartialEq, Message)]
struct GetPrivateObjectResponse {
    #[prost(string, tag = "1")]
    content_type: String,
    #[prost(bytes = "vec", tag = "2")]
    data: Vec<u8>,
}

/// Reads private documents from the Storage gRPC service.
pub struct StorageDocumentContentReader {
    runtime: tokio::runtime::Runtime,
    channel: Channel,
    service_name: String,
    service_secret: String,
}

impl StorageDocumentContentReader {
    pub fn connect(address: &str, service_name: &str, service_secret: &str) -> PortResult<Self> {
        if service_name.trim().is_empty() {
            return Err(invalid_input(
                "Storage caller service name must not be empty.",
            ));
        }

        if service_secret.trim().is_empty() {
            return Err(invalid_input(
                "Storage caller service secret must not be empty.",
            ));
        }

        let runtime = tokio::runtime::Builder::new_multi_thread()
            .enable_all()
            .build()?;

        let endpoint = Endpoint::from_shared(address.to_owned())?;
        let channel = runtime.block_on(endpoint.connect())?;

        Ok(Self {
            runtime,
            channel,
            service_name: service_name.to_owned(),
            service_secret: service_secret.to_owned(),
        })
    }

    async fn read_async(&self, reference: &StorageReference) -> PortResult<String> {
        let mut request = Request::new(GetPrivateObjectRequest {
            reference: reference.to_string(),
        });

        request.metadata_mut().insert(
            "x-service-name",
            self.service_name.parse().map_err(invalid_input)?,
        );
        request.metadata_mut().insert(
            "x-service-secret",
            self.service_secret.parse().map_err(invalid_input)?,
        );

        let mut client = tonic::client::Grpc::new(self.channel.clone());

        client.ready().await.map_err(|error| {
            io::Error::other(format!("Storage gRPC client is not ready: {error}"))
        })?;

        let response = client
            .server_streaming::<GetPrivateObjectRequest, GetPrivateObjectResponse, _>(
                request,
                PathAndQuery::from_static(GET_PRIVATE_OBJECT_PATH),
                tonic_prost::ProstCodec::default(),
            )
            .await?;

        let mut stream = response.into_inner();
        let mut content_type = String::new();
        let mut content = Vec::new();

        while let Some(response) = stream.next().await {
            let response = response?;

            if content_type.is_empty() && !response.content_type.trim().is_empty() {
                content_type = response.content_type.trim().to_owned();
            }

            content.extend_from_slice(&response.data);
        }

        if content_type.is_empty() {
            return Err(invalid_data(
                "Storage returned no content type for the knowledge document.",
            ));
        }

        extract_document_text(&content_type, &content)
    }
}

impl DocumentContentReader for StorageDocumentContentReader {
    fn read(&self, reference: &StorageReference) -> PortResult<String> {
        self.runtime.block_on(self.read_async(reference))
    }
}

fn extract_document_text(content_type: &str, content: &[u8]) -> PortResult<String> {
    let media_type = content_type
        .split(';')
        .next()
        .unwrap_or_default()
        .trim()
        .to_lowercase();

    if media_type == "application/pdf" {
        return pdf_extract::extract_text_from_mem(content)
            .map(|text| text.trim().to_owned())
            .map_err(|error| {
                invalid_data(format!(
                    "Could not extract text from the PDF knowledge document: {error}"
                ))
            });
    }

    if media_type.starts_with("text/") {
        let text = std::str::from_utf8(content)
            .map_err(|_| invalid_data("Text knowledge documents must be valid UTF-8."))?;

        return Ok(text
            .strip_prefix('\u{feff}')
            .unwrap_or(text)
            .trim()
            .to_owned());
    }

    Err(invalid_data(format!(
        "Unsupported knowledge document content type: {media_type}"
    )))
}

fn invalid_input(message: impl ToString) -> Box<dyn Error + Send + Sync> {
    Box::new(io::Error::new(
        io::ErrorKind::InvalidInput,
        message.to_string(),
    ))
}

fn invalid_data(message: impl ToString) -> Box<dyn Error + Send + Sync> {
    Box::new(io::Error::new(
        io::ErrorKind::InvalidData,
        message.to_string(),
    ))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn extracts_utf8_text() {
        let result = extract_document_text(
            "text/plain; charset=utf-8",
            b"\xef\xbb\xbf  Hello knowledge  ",
        )
        .unwrap();

        assert_eq!(result, "Hello knowledge");
    }

    #[test]
    fn extracts_pdf_text() {
        let result = extract_document_text(
            "application/pdf",
            include_bytes!("../../../tests/fixtures/knowledge.pdf"),
        );

        assert!(result.unwrap().contains("Knowledge document"));
    }

    #[test]
    fn rejects_missing_content_type() {
        let result = extract_document_text("", b"document");

        assert!(result.unwrap_err().to_string().contains("Unsupported"));
    }

    #[test]
    fn rejects_unsupported_content_type() {
        let result = extract_document_text("application/octet-stream", b"\x00\x01");

        assert!(result.unwrap_err().to_string().contains("Unsupported"));
    }

    #[test]
    fn rejects_invalid_utf8() {
        let result = extract_document_text("text/plain", &[0xff, 0xfe]);

        assert!(result.unwrap_err().to_string().contains("valid UTF-8"));
    }

    #[test]
    fn rejects_invalid_pdf() {
        let result = extract_document_text("application/pdf", b"not a PDF");

        assert!(
            result
                .unwrap_err()
                .to_string()
                .contains("Could not extract text")
        );
    }
}
