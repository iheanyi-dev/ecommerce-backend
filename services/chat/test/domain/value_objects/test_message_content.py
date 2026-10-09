from pytest import raises

from domain.errors.errors import InvalidMessageContentError
from domain.value_objects.message_content import MessageContent


def test_accepts_non_empty_content() -> None:
    content = MessageContent("Hello, store!")

    assert content.value == "Hello, store!"


def test_rejects_empty_content() -> None:
    with raises(InvalidMessageContentError):
        MessageContent("")


def test_rejects_whitespace_only_content() -> None:
    with raises(InvalidMessageContentError):
        MessageContent("   \n\t")


def test_preserves_original_content() -> None:
    content = MessageContent("  Hello  ")

    assert content.value == "  Hello  "
