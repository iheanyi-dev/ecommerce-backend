"""Tests for the Storage gRPC document content reader."""

from dataclasses import dataclass
from unittest.mock import Mock

import pytest

from domain.value_objects import StorageReference
from infrastructure.storage_document_content_reader import (
    StorageDocumentContentReader,
)
from storage.v1 import storage_pb2


@dataclass
class Response:
    """Small response fake matching the fields used by the Storage adapter."""

    content_type: str = ""
    data: bytes = b""


def make_reader(responses):
    stub = Mock()
    stub.GetPrivateObject.return_value = iter(responses)
    reader = StorageDocumentContentReader(
        stub=stub,
        service_name="embedding",
        service_secret="test-secret",
    )
    return reader, stub


def test_read_retrieves_private_object_with_service_auth_metadata():
    reader, stub = make_reader(
        [
            Response("text/plain; charset=utf-8", b"Hello "),
            Response("", b"knowledge"),
        ]
    )

    result = reader.read(StorageReference("knowledge/store-1/readme.txt"))

    assert result == "Hello knowledge"
    stub.GetPrivateObject.assert_called_once()
    request = stub.GetPrivateObject.call_args.args[0]
    metadata = stub.GetPrivateObject.call_args.kwargs["metadata"]
    assert isinstance(request, storage_pb2.GetPrivateObjectRequest)
    assert request.reference == "knowledge/store-1/readme.txt"
    assert ("x-service-name", "embedding") in metadata
    assert ("x-service-secret", "test-secret") in metadata


def test_read_extracts_pdf_text(monkeypatch):
    page = Mock()
    page.extract_text.return_value = "Extracted PDF text"
    pdf_reader = Mock()
    pdf_reader.pages = [page]
    monkeypatch.setattr(
        "infrastructure.storage_document_content_reader.PdfReader",
        lambda stream: pdf_reader,
    )
    reader, _ = make_reader([Response("application/pdf", b"%PDF-test")])

    assert reader.read(StorageReference("knowledge/store-1/guide.pdf")) == (
        "Extracted PDF text"
    )


def test_read_rejects_missing_content_type():
    reader, _ = make_reader([Response("", b"some bytes")])

    with pytest.raises(ValueError, match="no content type"):
        reader.read(StorageReference("knowledge/store-1/file.txt"))


def test_read_rejects_unsupported_content_type():
    reader, _ = make_reader([Response("application/octet-stream", b"\x00\x01")])

    with pytest.raises(ValueError, match="Unsupported knowledge document"):
        reader.read(StorageReference("knowledge/store-1/file.bin"))


def test_read_rejects_invalid_utf8_text():
    reader, _ = make_reader([Response("text/plain", b"\xff\xfe")])

    with pytest.raises(ValueError, match="valid UTF-8"):
        reader.read(StorageReference("knowledge/store-1/file.txt"))


def test_read_rejects_invalid_pdf(monkeypatch):
    def fail_pdf_reader(_stream):
        raise RuntimeError("invalid PDF")

    monkeypatch.setattr(
        "infrastructure.storage_document_content_reader.PdfReader",
        fail_pdf_reader,
    )
    reader, _ = make_reader([Response("application/pdf", b"not a PDF")])

    with pytest.raises(ValueError, match="Could not extract text"):
        reader.read(StorageReference("knowledge/store-1/file.pdf"))


def test_constructor_rejects_empty_service_credentials():
    with pytest.raises(ValueError, match="service secret"):
        StorageDocumentContentReader(Mock(), "embedding", " ")
