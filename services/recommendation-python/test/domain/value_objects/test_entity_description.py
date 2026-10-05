"""Tests for the EntityDescription value object."""

import pytest

from domain.errors import EmptyEntityDescriptionError
from domain.value_objects import EntityDescription


def test_entity_description_accepts_non_empty_description() -> None:
    value = EntityDescription("A quality store.", "store")

    assert value.value == "A quality store."
    assert str(value) == "A quality store."


def test_entity_description_rejects_empty_description() -> None:
    with pytest.raises(EmptyEntityDescriptionError):
        EntityDescription("   ", "store")
