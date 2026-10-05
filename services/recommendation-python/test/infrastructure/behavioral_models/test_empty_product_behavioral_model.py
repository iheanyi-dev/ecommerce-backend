from uuid import uuid4

from infrastructure.behavioral_models.empty_product_behavioral_model import (
    EmptyProductBehavioralAssociationModel,
)


def test_returns_empty_list() -> None:
    model = EmptyProductBehavioralAssociationModel()

    result = model.find_associated(
        product_id=uuid4(),
        limit=10,
    )

    assert result == []
