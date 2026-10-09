from pytest import raises

from domain.errors.errors import InvalidMessageRoleError
from domain.value_objects.message_role import MessageRole


def test_supports_user_role() -> None:
    assert MessageRole.from_value("user") is MessageRole.USER


def test_supports_assistant_role() -> None:
    assert MessageRole.from_value("assistant") is MessageRole.ASSISTANT


def test_rejects_unsupported_role() -> None:
    with raises(InvalidMessageRoleError):
        MessageRole.from_value("system")
