"""Provider-neutral data structures for Agent orchestration."""

from dataclasses import dataclass
from typing import Any
from uuid import UUID


@dataclass(frozen=True, slots=True)
class ModelMessage:
    """A message exchanged with the model."""

    role: str
    content: str | None = None
    tool_call_id: str | None = None
    tool_calls: tuple["ToolCall", ...] = ()
    tool_name: str | None = None
    # Opaque provider metadata preserves information required for later turns.
    provider_data: Any | None = None


@dataclass(frozen=True, slots=True)
class ToolCall:
    """A tool invocation requested by the model."""

    id: str
    name: str
    arguments: dict[str, Any]


@dataclass(frozen=True, slots=True)
class ModelResponse:
    """A model response containing text, tool calls, or both."""

    content: str | None = None
    tool_calls: tuple[ToolCall, ...] = ()
    # Provider-specific content may be required to continue a tool-call turn.
    provider_data: Any | None = None


@dataclass(frozen=True, slots=True)
class ToolDefinition:
    """Tool metadata provided to the model."""

    name: str
    description: str
    input_schema: dict[str, Any]


@dataclass(frozen=True, slots=True)
class ToolContext:
    """Trusted request context passed to a tool."""

    store_id: UUID
    conversation_id: UUID
    current_message_id: UUID
