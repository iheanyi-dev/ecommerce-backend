"""Tests for the EntityName value object."""

import pytest

from domain.errors import EmptyEntityNameError
from domain.value_objects import EntityName


def test_entity_name_accepts_non_empty_name() -> None:
    value = EntityName("Acme Store", "store")

    assert value.value == "Acme Store"
    assert str(value) == "Acme Store"


def test_entity_name_rejects_empty_name() -> None:
    with pytest.raises(EmptyEntityNameError):
        EntityName("   ", "store")
