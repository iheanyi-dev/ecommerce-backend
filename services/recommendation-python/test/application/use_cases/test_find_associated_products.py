"""Tests for the FindAssociatedProducts use case."""

from __future__ import annotations

from uuid import UUID, uuid4

import pytest

from application.dto import FindAssociatedProductsInput
from application.use_cases import FindAssociatedProducts
from test.support.observability import StubLogger, StubMetrics


class StubProductBehavioralAssociationModel:
    """Fake behavioral model for application-level tests."""

    def __init__(
        self,
        recommendations: list[tuple[UUID, float]],
    ) -> None:
        self.recommendations = recommendations
        self.calls: list[tuple[UUID, int]] = []

    def find_associated(
        self,
        product_id: UUID,
        limit: int,
    ) -> list[tuple[UUID, float]]:
        self.calls.append((product_id, limit))

        return self.recommendations[:limit]


def make_recommendations(
    count: int,
) -> list[tuple[UUID, float]]:
    """Create deterministic behavioral recommendations for tests."""

    return [
        (uuid4(), 1.0 - (index * 0.01))
        for index in range(count)
    ]


def test_requests_associations_for_source_product() -> None:
    """The use case queries the model for the supplied Product."""

    source_product_id = uuid4()
    model = StubProductBehavioralAssociationModel(
        make_recommendations(10),
    )

    use_case = FindAssociatedProducts(
        product_behavioral_association_model=model,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    use_case.execute(
        FindAssociatedProductsInput(
            product_id=source_product_id,
            page=1,
        )
    )

    assert model.calls[0][0] == source_product_id


def test_page_two_requests_twenty_candidates_and_returns_second_page() -> None:
    """Page two retrieves twenty candidates and returns candidates 10-19."""

    source_product_id = uuid4()
    recommendations = make_recommendations(20)
    model = StubProductBehavioralAssociationModel(
        recommendations,
    )

    use_case = FindAssociatedProducts(
        product_behavioral_association_model=model,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    output = use_case.execute(
        FindAssociatedProductsInput(
            product_id=source_product_id,
            page=2,
        )
    )

    assert model.calls[0][1] == 20

    assert len(output.recommendations) == 10

    assert [
        recommendation.product_id
        for recommendation in output.recommendations
    ] == [
        product_id
        for product_id, _ in recommendations[10:20]
    ]


def test_last_page_returns_remaining_candidates() -> None:
    """A final partial page returns remaining behavioral recommendations."""

    source_product_id = uuid4()
    recommendations = make_recommendations(13)
    model = StubProductBehavioralAssociationModel(
        recommendations,
    )

    use_case = FindAssociatedProducts(
        product_behavioral_association_model=model,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    output = use_case.execute(
        FindAssociatedProductsInput(
            product_id=source_product_id,
            page=2,
        )
    )

    assert [
        recommendation.product_id
        for recommendation in output.recommendations
    ] == [
        product_id
        for product_id, _ in recommendations[10:]
    ]


def test_page_zero_is_rejected() -> None:
    """Recommendation pages are one-based."""

    model = StubProductBehavioralAssociationModel([])

    use_case = FindAssociatedProducts(
        product_behavioral_association_model=model,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    with pytest.raises(
        ValueError,
        match="page must be greater than or equal to 1",
    ):
        use_case.execute(
            FindAssociatedProductsInput(
                product_id=uuid4(),
                page=0,
            )
        )


def test_page_one_returns_no_more_than_page_size() -> None:
    """The first behavioral recommendation page contains at most ten."""

    source_product_id = uuid4()
    recommendations = make_recommendations(20)
    model = StubProductBehavioralAssociationModel(
        recommendations,
    )

    use_case = FindAssociatedProducts(
        product_behavioral_association_model=model,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    output = use_case.execute(
        FindAssociatedProductsInput(
            product_id=source_product_id,
            page=1,
        )
    )

    assert len(output.recommendations) == 10
    assert model.calls[0][1] == 10
