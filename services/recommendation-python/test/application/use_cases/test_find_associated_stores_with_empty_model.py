from uuid import uuid4

from application.dto.find_associated_stores import FindAssociatedStoresInput
from application.use_cases.find_associated_stores import FindAssociatedStores
from infrastructure.behavioral_models.empty_store_behavioral_model import (
    EmptyStoreBehavioralAssociationModel,
)


def test_returns_empty_recommendations() -> None:
    use_case = FindAssociatedStores(
        store_behavioral_association_model=EmptyStoreBehavioralAssociationModel()
    )

    result = use_case.execute(
        FindAssociatedStoresInput(
            store_id=uuid4(),
            page=1,
        )
    )

    assert result.recommendations == ()
