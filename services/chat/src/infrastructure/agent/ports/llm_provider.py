"""Abstraction for model providers used by the Agent."""

from typing import Protocol

from infrastructure.agent.models import ModelMessage, ModelResponse, ToolDefinition


class LLMProvider(Protocol):
    """Generate a model response from messages and available tools."""

    async def generate(
        self,
        *,
        messages: list[ModelMessage],
        tools: list[ToolDefinition],
    ) -> ModelResponse:
        """Return the next model response."""
        ...
