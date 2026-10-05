"""Tests for the FindAssociatedStores use case."""

from __future__ import annotations

from uuid import UUID, uuid4

import pytest

from application.dto import FindAssociatedStoresInput
from application.use_cases import FindAssociatedStores
from test.support.observability import StubLogger, StubMetrics


class StubStoreBehavioralAssociationModel:
    """Fake behavioral association repository for application tests."""

    def __init__(
        self,
        recommendations: list[tuple[UUID, float]],
    ) -> None:
        self.recommendations = recommendations
        self.calls: list[tuple[UUID, int]] = []

    def find_associated(
        self,
        store_id: UUID,
        limit: int,
    ) -> list[tuple[UUID, float]]:
        self.calls.append((store_id, limit))

        return self.recommendations[:limit]


def make_recommendations(
    count: int,
) -> list[tuple[UUID, float]]:
    """Create deterministic behavioral recommendations for tests."""

    return [
        (uuid4(), 1.0 - (index * 0.01))
        for index in range(count)
    ]


def test_requests_associations_for_source_store() -> None:
    """The use case queries behavioral associations for the supplied Store."""

    source_store_id = uuid4()
    model = StubStoreBehavioralAssociationModel(
        make_recommendations(10),
    )

    use_case = FindAssociatedStores(
        store_behavioral_association_model=model,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    use_case.execute(
        FindAssociatedStoresInput(
            store_id=source_store_id,
            page=1,
        )
    )

    assert model.calls[0][0] == source_store_id


def test_page_two_requests_twenty_candidates_and_returns_second_page() -> None:
    """Page two retrieves twenty candidates and returns candidates 10-19."""

    source_store_id = uuid4()
    recommendations = make_recommendations(20)
    model = StubStoreBehavioralAssociationModel(
        recommendations,
    )

    use_case = FindAssociatedStores(
        store_behavioral_association_model=model,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    output = use_case.execute(
        FindAssociatedStoresInput(
            store_id=source_store_id,
            page=2,
        )
    )

    assert model.calls[0][1] == 20

    assert len(output.recommendations) == 10

    assert [
        recommendation.store_id
        for recommendation in output.recommendations
    ] == [
        store_id
        for store_id, _ in recommendations[10:20]
    ]


def test_last_page_returns_remaining_candidates() -> None:
    """A final partial page returns remaining behavioral recommendations."""

    source_store_id = uuid4()
    recommendations = make_recommendations(13)
    model = StubStoreBehavioralAssociationModel(
        recommendations,
    )

    use_case = FindAssociatedStores(
        store_behavioral_association_model=model,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    output = use_case.execute(
        FindAssociatedStoresInput(
            store_id=source_store_id,
            page=2,
        )
    )

    assert [
        recommendation.store_id
        for recommendation in output.recommendations
    ] == [
        store_id
        for store_id, _ in recommendations[10:]
    ]


def test_page_zero_is_rejected() -> None:
    """Recommendation pages are one-based."""

    model = StubStoreBehavioralAssociationModel([])

    use_case = FindAssociatedStores(
        store_behavioral_association_model=model,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    with pytest.raises(
        ValueError,
        match="page must be greater than or equal to 1",
    ):
        use_case.execute(
            FindAssociatedStoresInput(
                store_id=uuid4(),
                page=0,
            )
        )


def test_page_one_returns_no_more_than_page_size() -> None:
    """The first behavioral recommendation page contains at most ten."""

    source_store_id = uuid4()
    recommendations = make_recommendations(20)
    model = StubStoreBehavioralAssociationModel(
        recommendations,
    )

    use_case = FindAssociatedStores(
        store_behavioral_association_model=model,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    output = use_case.execute(
        FindAssociatedStoresInput(
            store_id=source_store_id,
            page=1,
        )
    )

    assert len(output.recommendations) == 10
    assert model.calls[0][1] == 10
