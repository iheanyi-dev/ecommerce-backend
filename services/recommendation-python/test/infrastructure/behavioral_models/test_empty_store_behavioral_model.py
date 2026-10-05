from uuid import uuid4

from infrastructure.behavioral_models.empty_store_behavioral_model import (
    EmptyStoreBehavioralAssociationModel,
)


def test_returns_empty_list() -> None:
    model = EmptyStoreBehavioralAssociationModel()

    result = model.find_associated(
        store_id=uuid4(),
        limit=10,
    )

    assert result == []
