"""Tests for the EntityID value object."""

from uuid import uuid4

import pytest

from domain.errors import InvalidEntityIDError
from domain.value_objects import EntityID


def test_entity_id_accepts_uuid() -> None:
    value = uuid4()

    entity_id = EntityID(value, "store")

    assert entity_id.value == value
    assert str(entity_id) == str(value)


def test_entity_id_rejects_non_uuid() -> None:
    with pytest.raises(InvalidEntityIDError):
        EntityID("not-a-uuid", "store")  # type: ignore[arg-type]
