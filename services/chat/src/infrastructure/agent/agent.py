"""Concrete multi-step Agent orchestration."""

import json
from collections.abc import AsyncIterator, Iterable
from time import perf_counter
from typing import Any
from uuid import UUID

from jsonschema import Draft202012Validator, ValidationError

from application.ports.agent import Agent as AgentPort
from application.ports.logger import Logger
from application.ports.metrics import Metrics
from infrastructure.agent.models import (
    ModelMessage,
    ModelResponse,
    ToolCall,
    ToolContext,
    ToolDefinition,
)
from infrastructure.agent.ports.llm_provider import LLMProvider
from infrastructure.agent.ports.tool import Tool


class AgentExecutionLimitError(RuntimeError):
    """Raised when the Agent exceeds its configured tool-round limit."""


class Agent(AgentPort):
    """Coordinate model turns and on-demand tool execution."""

    def __init__(
        self,
        *,
        provider: LLMProvider,
        tools: Iterable[Tool],
        logger: Logger,
        metrics: Metrics,
        max_tool_rounds: int = 8,
    ) -> None:
        if max_tool_rounds < 1:
            raise ValueError("max_tool_rounds must be at least 1")

        self._provider = provider
        self._logger = logger
        self._metrics = metrics
        self._max_tool_rounds = max_tool_rounds

        tool_map: dict[str, Tool] = {}
        validators: dict[str, Draft202012Validator] = {}
        for tool in tools:
            if not tool.name or tool.name in tool_map:
                raise ValueError("tool names must be non-empty and unique")
            if not isinstance(tool.input_schema, dict):
                raise ValueError(f"tool '{tool.name}' schema must be an object")
            Draft202012Validator.check_schema(tool.input_schema)
            tool_map[tool.name] = tool
            validators[tool.name] = Draft202012Validator(tool.input_schema)
        self._tools = tool_map
        self._validators = validators

    async def run(
        self,
        store_id: UUID,
        conversation_id: UUID,
        current_message_id: UUID,
        message: str,
        system_prompt: str,
    ) -> AsyncIterator[str]:
        """Run the model/tool loop and yield only the final response text."""
        started_at = perf_counter()
        messages = [
            ModelMessage(role="system", content=system_prompt),
            ModelMessage(role="user", content=message),
        ]
        context = ToolContext(
            store_id=store_id,
            conversation_id=conversation_id,
            current_message_id=current_message_id,
        )
        tool_rounds = 0
        tool_calls_total = 0

        try:
            while True:
                response = await self._provider.generate(
                    messages=messages,
                    tools=[self._definition(tool) for tool in self._tools.values()],
                )
                if not isinstance(response, ModelResponse):
                    raise TypeError("LLM provider must return ModelResponse")

                if response.tool_calls:
                    tool_rounds += 1
                    if tool_rounds > self._max_tool_rounds:
                        self._metrics.increment(
                            "chat.agent.execution_limit_reached"
                        )
                        raise AgentExecutionLimitError(
                            "Agent exceeded the maximum number of tool rounds"
                        )

                    # Preserve the model's tool-call request in the transcript.
                    messages.append(
                        ModelMessage(
                            role="assistant",
                            content=response.content,
                            tool_calls=response.tool_calls,
                            provider_data=response.provider_data,
                        )
                    )

                    for call in response.tool_calls:
                        result = await self._execute_tool(call, context)
                        tool_calls_total += 1
                        messages.append(
                            ModelMessage(
                                role="tool",
                                content=result,
                                tool_call_id=call.id,
                                tool_name=call.name,
                            )
                        )
                    continue

                content = response.content or ""
                if not content.strip():
                    raise ValueError("Agent returned an empty response")

                # Intermediate text is withheld when the same response requests
                # tools. Only a final response without tool calls is streamed.
                yield content
                self._metrics.increment("chat.agent.completed")
                self._metrics.observe(
                    "chat.agent.duration_seconds",
                    perf_counter() - started_at,
                )
                self._logger.info(
                    "chat.agent.completed",
                    store_id=str(store_id),
                    conversation_id=str(conversation_id),
                    tool_calls=tool_calls_total,
                    tool_rounds=tool_rounds,
                )
                return

        except Exception as exc:
            self._metrics.increment("chat.agent.failed")
            self._metrics.observe(
                "chat.agent.duration_seconds",
                perf_counter() - started_at,
            )
            self._logger.error(
                "chat.agent.failed",
                store_id=str(store_id),
                conversation_id=str(conversation_id),
                error_type=type(exc).__name__,
            )
            raise

    @staticmethod
    def _definition(tool: Tool) -> ToolDefinition:
        """Build a provider-neutral definition from a tool."""
        return ToolDefinition(
            name=tool.name,
            description=tool.description,
            input_schema=tool.input_schema,
        )

    async def _execute_tool(
        self,
        call: ToolCall,
        context: ToolContext,
    ) -> str:
        """Execute one requested tool without leaking internal errors."""
        tool = self._tools.get(call.name)
        if tool is None:
            self._logger.warning(
                "chat.agent.tool_unknown",
                tool_name=call.name,
            )
            return json.dumps({"error": "Requested tool is unavailable."})

        if not isinstance(call.arguments, dict):
            self._metrics.increment("chat.agent.tool_invalid_arguments")
            self._logger.warning(
                "chat.agent.tool_invalid_arguments",
                tool_name=call.name,
                reason="not_an_object",
            )
            return json.dumps({"error": "Tool arguments must be an object."})

        try:
            self._validators[call.name].validate(call.arguments)
        except ValidationError:
            self._metrics.increment("chat.agent.tool_invalid_arguments")
            self._logger.warning(
                "chat.agent.tool_invalid_arguments",
                tool_name=call.name,
                reason="schema_mismatch",
            )
            return json.dumps({
                "error": "Tool arguments are invalid.",
                "hint": "Provide arguments matching the tool's input schema.",
            })

        try:
            result = await tool.execute(arguments=call.arguments, context=context)
            if not isinstance(result, str):
                raise TypeError("Tool results must be strings")
            self._metrics.increment("chat.agent.tool_calls")
            return result
        except Exception as exc:
            self._logger.warning(
                "chat.agent.tool_failed",
                tool_name=call.name,
                error_type=type(exc).__name__,
            )
            self._metrics.increment("chat.agent.tool_failures")
            return json.dumps(
                {"error": "The requested operation failed. Try another approach."}
            )
