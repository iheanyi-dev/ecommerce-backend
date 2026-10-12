"""Storage gRPC adapter for retrieving and extracting knowledge documents."""

from __future__ import annotations

from io import BytesIO
from typing import Protocol

import grpc
from pypdf import PdfReader

from domain.value_objects import StorageReference
from storage.v1 import storage_pb2, storage_pb2_grpc


class _StorageStub(Protocol):
    """Minimal Storage stub contract used by this adapter."""

    def GetPrivateObject(self, request, *, metadata):
        """Return a stream of private-object chunks."""
        ...


class StorageDocumentContentReader:
    """Retrieve a private Storage object and extract its textual content.

    The Storage RPC returns raw bytes. This adapter converts supported
    document formats into text for the application use case.
    """

    def __init__(
        self,
        stub: _StorageStub,
        service_name: str,
        service_secret: str,
    ) -> None:
        if not service_name.strip():
            raise ValueError("Storage caller service name must not be empty.")
        if not service_secret.strip():
            raise ValueError("Storage caller service secret must not be empty.")

        self._stub = stub
        self._metadata = (
            ("x-service-name", service_name),
            ("x-service-secret", service_secret),
        )

    @classmethod
    def connect(cls, settings) -> "StorageDocumentContentReader":
        """Create a reader backed by a real Storage gRPC channel."""

        channel = grpc.insecure_channel(settings.address)
        reader = cls(
            storage_pb2_grpc.StorageServiceStub(channel),
            settings.service_name,
            settings.service_secret,
        )
        reader._channel = channel
        return reader

    def close(self) -> None:
        """Close the underlying gRPC channel, if this reader owns one."""

        channel = getattr(self, "_channel", None)
        if channel is not None:
            channel.close()
            self._channel = None

    def read(self, storage_reference: StorageReference) -> str:
        """Retrieve an object and return its extracted text."""

        responses = self._stub.GetPrivateObject(
            storage_pb2.GetPrivateObjectRequest(
                reference=storage_reference.value
            ),
            metadata=self._metadata,
        )

        content_type = ""
        content = bytearray()

        for response in responses:
            # Storage normally sets this on the first response. Accept a
            # later non-empty value too, without replacing an earlier one.
            if not content_type and response.content_type.strip():
                content_type = response.content_type.strip()
            content.extend(response.data)

        if not content_type:
            raise ValueError(
                "Storage returned no content type for the knowledge document."
            )

        media_type = content_type.split(";", maxsplit=1)[0].strip().lower()
        raw_content = bytes(content)

        if media_type == "application/pdf":
            try:
                pages = PdfReader(BytesIO(raw_content)).pages
                extracted_pages = [
                    page.extract_text() or "" for page in pages
                ]
            except Exception as exc:
                raise ValueError(
                    "Could not extract text from the PDF knowledge document."
                ) from exc

            return "\n".join(extracted_pages).strip()

        if media_type.startswith("text/"):
            try:
                return raw_content.decode("utf-8-sig").strip()
            except UnicodeDecodeError as exc:
                raise ValueError(
                    "Text knowledge documents must be valid UTF-8."
                ) from exc

        raise ValueError(
            f"Unsupported knowledge document content type: {media_type}"
        )
