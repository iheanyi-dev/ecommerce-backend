"""Contract for tools available to the Agent."""

from typing import Any, Protocol

from infrastructure.agent.models import ToolContext


class Tool(Protocol):
    """An executable capability exposed to the Agent."""

    name: str
    description: str
    input_schema: dict[str, Any]

    async def execute(
        self,
        *,
        arguments: dict[str, Any],
        context: ToolContext,
    ) -> str:
        """Execute validated arguments and return a textual result."""
        ...
