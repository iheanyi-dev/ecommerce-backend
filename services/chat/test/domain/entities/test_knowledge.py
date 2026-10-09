from uuid import uuid4

from domain.entities.knowledge import Knowledge


def test_creates_knowledge() -> None:
    knowledge_id = uuid4()
    store_id = uuid4()

    knowledge = Knowledge(
        id=knowledge_id,
        store_id=store_id,
        filename="returns-policy.pdf",
        content_type="application/pdf",
    )

    assert knowledge.id == knowledge_id
    assert knowledge.store_id == store_id
    assert knowledge.filename == "returns-policy.pdf"
    assert knowledge.content_type == "application/pdf"


def test_derives_storage_reference_from_store_id_and_filename() -> None:
    store_id = uuid4()

    knowledge = Knowledge(
        id=uuid4(),
        store_id=store_id,
        filename="returns-policy.pdf",
        content_type="application/pdf",
    )

    assert (
        knowledge.storage_reference
        == f"knowledge/{store_id}/returns-policy.pdf"
    )


def test_preserves_filename_exactly() -> None:
    store_id = uuid4()
    filename = "Store Returns Policy (2026).PDF"

    knowledge = Knowledge(
        id=uuid4(),
        store_id=store_id,
        filename=filename,
        content_type="application/pdf",
    )

    assert knowledge.filename == filename
    assert knowledge.storage_reference == f"knowledge/{store_id}/{filename}"
