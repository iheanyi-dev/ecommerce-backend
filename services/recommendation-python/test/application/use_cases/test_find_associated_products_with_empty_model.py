from test.support.observability import StubLogger, StubMetrics
from uuid import uuid4

from application.dto.find_associated_products import FindAssociatedProductsInput
from application.use_cases.find_associated_products import FindAssociatedProducts
from infrastructure.behavioral_models.empty_product_behavioral_model import (
    EmptyProductBehavioralAssociationModel,
)


def test_returns_empty_recommendations() -> None:
    use_case = FindAssociatedProducts(
        product_behavioral_association_model=EmptyProductBehavioralAssociationModel(),
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    result = use_case.execute(
        FindAssociatedProductsInput(
            product_id=uuid4(),
            page=1,
        )
    )

    assert result.recommendations == ()
