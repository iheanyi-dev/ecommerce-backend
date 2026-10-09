"""Tests for infrastructure Agent orchestration."""

from collections.abc import AsyncIterator
from typing import Any
from uuid import UUID, uuid4

import pytest
from jsonschema.exceptions import SchemaError

from infrastructure.agent.agent import Agent, AgentExecutionLimitError
from infrastructure.agent.models import (
    ModelMessage,
    ModelResponse,
    ToolCall,
    ToolContext,
    ToolDefinition,
)
from infrastructure.agent.ports.llm_provider import LLMProvider
from infrastructure.agent.ports.tool import Tool


class StubLogger:
    def __init__(self) -> None:
        self.events: list[tuple[str, str, dict[str, Any]]] = []

    def info(self, event: str, **fields: Any) -> None:
        self.events.append(("info", event, fields))

    def warning(self, event: str, **fields: Any) -> None:
        self.events.append(("warning", event, fields))

    def error(self, event: str, **fields: Any) -> None:
        self.events.append(("error", event, fields))


class StubMetrics:
    def __init__(self) -> None:
        self.counters: dict[str, int] = {}

    def increment(self, name: str, value: int = 1) -> None:
        self.counters[name] = self.counters.get(name, 0) + value

    def observe(self, name: str, value: float) -> None:
        pass


class StubProvider:
    def __init__(self, responses: list[ModelResponse]) -> None:
        self.responses = responses
        self.requests: list[tuple[list[ModelMessage], list[ToolDefinition]]] = []

    async def generate(
        self,
        *,
        messages: list[ModelMessage],
        tools: list[ToolDefinition],
    ) -> ModelResponse:
        self.requests.append((list(messages), list(tools)))
        if not self.responses:
            raise AssertionError("Agent requested an unexpected model turn")
        return self.responses.pop(0)


class StubTool:
    def __init__(
        self,
        name: str,
        result: str = "tool result",
        *,
        error: Exception | None = None,
    ) -> None:
        self.name = name
        self.description = f"Test tool: {name}"
        self.input_schema: dict[str, object] = {
            "type": "object",
            "properties": {"query": {"type": "string"}},
        }
        self.result = result
        self.error = error
        self.calls: list[tuple[dict[str, object], ToolContext]] = []

    async def execute(
        self,
        *,
        arguments: dict[str, object],
        context: ToolContext,
    ) -> str:
        self.calls.append((arguments, context))
        if self.error is not None:
            raise self.error
        return self.result


async def collect(stream: AsyncIterator[str]) -> list[str]:
    return [chunk async for chunk in stream]


def build_agent(
    responses: list[ModelResponse],
    tools: list[Tool] | None = None,
    *,
    max_tool_rounds: int = 8,
) -> tuple[Agent, StubProvider, StubLogger, StubMetrics]:
    provider = StubProvider(responses)
    logger = StubLogger()
    metrics = StubMetrics()
    agent = Agent(
        provider=provider,
        tools=tools or [],
        logger=logger,
        metrics=metrics,
        max_tool_rounds=max_tool_rounds,
    )
    return agent, provider, logger, metrics


@pytest.mark.asyncio
async def test_executes_multiple_tool_rounds_and_loads_history_on_demand() -> None:
    store_id = uuid4()
    conversation_id = uuid4()
    current_message_id = uuid4()

    history = StubTool("conversation_history", "Earlier customer messages")
    products = StubTool("product_lookup", "Matching products")

    agent, provider, _, metrics = build_agent(
        [
            ModelResponse(
                tool_calls=(
                    ToolCall(
                        id="call-history",
                        name="conversation_history",
                        arguments={"limit": 10},
                    ),
                )
            ),
            ModelResponse(
                tool_calls=(
                    ToolCall(
                        id="call-products",
                        name="product_lookup",
                        arguments={"query": "running shoes"},
                    ),
                )
            ),
            ModelResponse(content="Here are the matching products."),
        ],
        [history, products],
    )

    chunks = await collect(
        agent.run(
            store_id=store_id,
            conversation_id=conversation_id,
            current_message_id=current_message_id,
            message="Find running shoes like the ones I asked about before.",
            system_prompt="Help customers shop.",
        )
    )

    assert chunks == ["Here are the matching products."]
    assert len(provider.requests) == 3

    first_messages, first_tools = provider.requests[0]
    assert [message.role for message in first_messages] == ["system", "user"]
    assert {tool.name for tool in first_tools} == {
        "conversation_history",
        "product_lookup",
    }
    assert len(history.calls) == 1
    assert history.calls[0][0] == {"limit": 10}
    assert history.calls[0][1].store_id == store_id
    assert history.calls[0][1].conversation_id == conversation_id
    assert history.calls[0][1].current_message_id == current_message_id
    assert len(products.calls) == 1
    assert products.calls[0][0] == {"query": "running shoes"}

    second_messages, _ = provider.requests[1]
    assert any(
        message.role == "tool"
        and message.content == "Earlier customer messages"
        for message in second_messages
    )

    third_messages, _ = provider.requests[2]
    assert any(
        message.role == "tool" and message.content == "Matching products"
        for message in third_messages
    )
    assert metrics.counters["chat.agent.tool_calls"] == 2
    assert metrics.counters["chat.agent.completed"] == 1


@pytest.mark.asyncio
async def test_does_not_load_history_without_a_model_tool_call() -> None:
    history = StubTool("conversation_history", "History")
    agent, provider, _, _ = build_agent(
        [ModelResponse(content="Hello!")],
        [history],
    )

    chunks = await collect(
        agent.run(
            store_id=uuid4(),
            conversation_id=uuid4(),
            current_message_id=uuid4(),
            message="Hello",
            system_prompt="Be helpful.",
        )
    )

    assert chunks == ["Hello!"]
    assert history.calls == []
    assert len(provider.requests) == 1
    assert [message.role for message in provider.requests[0][0]] == [
        "system",
        "user",
    ]


@pytest.mark.asyncio
async def test_returns_safe_result_for_unknown_tool_and_continues() -> None:
    agent, provider, logger, _ = build_agent(
        [
            ModelResponse(
                tool_calls=(
                    ToolCall(
                        id="call-unknown",
                        name="private_internal_tool",
                        arguments={},
                    ),
                )
            ),
            ModelResponse(
                content="I couldn't access that capability, but I can help another way."
            ),
        ]
    )

    chunks = await collect(
        agent.run(
            store_id=uuid4(),
            conversation_id=uuid4(),
            current_message_id=uuid4(),
            message="Help me.",
            system_prompt="Be helpful.",
        )
    )

    assert chunks == [
        "I couldn't access that capability, but I can help another way."
    ]
    tool_messages = [
        message
        for message in provider.requests[1][0]
        if message.role == "tool"
    ]
    assert len(tool_messages) == 1
    assert "private_internal_tool" not in (tool_messages[0].content or "")
    assert any(level == "warning" for level, _, _ in logger.events)


@pytest.mark.asyncio
async def test_returns_safe_result_when_tool_execution_fails() -> None:
    tool = StubTool(
        "product_lookup",
        error=RuntimeError("database password must not leak"),
    )
    agent, provider, logger, _ = build_agent(
        [
            ModelResponse(
                tool_calls=(
                    ToolCall(
                        id="call-products",
                        name="product_lookup",
                        arguments={"query": "shoes"},
                    ),
                )
            ),
            ModelResponse(content="I couldn't retrieve products just now."),
        ],
        [tool],
    )

    chunks = await collect(
        agent.run(
            store_id=uuid4(),
            conversation_id=uuid4(),
            current_message_id=uuid4(),
            message="Find shoes.",
            system_prompt="Help customers.",
        )
    )

    assert chunks == ["I couldn't retrieve products just now."]
    tool_messages = [
        message
        for message in provider.requests[1][0]
        if message.role == "tool"
    ]
    assert len(tool_messages) == 1
    assert "database password" not in (tool_messages[0].content or "")
    assert any(level == "warning" for level, _, _ in logger.events)


@pytest.mark.asyncio
async def test_enforces_maximum_tool_rounds() -> None:
    tool = StubTool("repeat")
    agent, _, _, metrics = build_agent(
        [
            ModelResponse(
                tool_calls=(
                    ToolCall(id="call-1", name="repeat", arguments={}),
                )
            ),
            ModelResponse(
                tool_calls=(
                    ToolCall(id="call-2", name="repeat", arguments={}),
                )
            ),
        ],
        [tool],
        max_tool_rounds=1,
    )

    with pytest.raises(AgentExecutionLimitError):
        await collect(
            agent.run(
                store_id=uuid4(),
                conversation_id=uuid4(),
                current_message_id=uuid4(),
                message="Keep repeating.",
                system_prompt="Use tools as needed.",
            )
        )

    assert len(tool.calls) == 1
    assert metrics.counters["chat.agent.execution_limit_reached"] == 1


@pytest.mark.asyncio
async def test_rejects_empty_final_response() -> None:
    agent, _, _, metrics = build_agent([ModelResponse(content="  ")])

    with pytest.raises(ValueError, match="empty"):
        await collect(
            agent.run(
                store_id=uuid4(),
                conversation_id=uuid4(),
                current_message_id=uuid4(),
                message="Hello",
                system_prompt="Be helpful.",
            )
        )

    assert metrics.counters["chat.agent.failed"] == 1


def test_rejects_invalid_tool_schema_during_agent_initialization() -> None:
    tool = StubTool("invalid_schema")
    tool.input_schema = {"type": "not-a-valid-json-schema-type"}

    with pytest.raises(SchemaError):
        build_agent([], [tool])


@pytest.mark.asyncio
async def test_rejects_arguments_that_do_not_match_tool_schema() -> None:
    tool = StubTool("product_lookup")
    agent, provider, logger, metrics = build_agent(
        [
            ModelResponse(
                tool_calls=(
                    ToolCall(
                        id="call-invalid",
                        name="product_lookup",
                        arguments={"query": 123},
                    ),
                )
            ),
            ModelResponse(content="I couldn't use those search parameters."),
        ],
        [tool],
    )

    chunks = await collect(
        agent.run(
            store_id=uuid4(),
            conversation_id=uuid4(),
            current_message_id=uuid4(),
            message="Find products.",
            system_prompt="Help customers.",
        )
    )

    assert chunks == ["I couldn't use those search parameters."]
    assert tool.calls == []
    assert metrics.counters["chat.agent.tool_invalid_arguments"] == 1

    tool_messages = [
        message
        for message in provider.requests[1][0]
        if message.role == "tool"
    ]
    assert len(tool_messages) == 1
    assert "invalid" in (tool_messages[0].content or "").lower()
    assert any(
        event == "chat.agent.tool_invalid_arguments"
        for _, event, _ in logger.events
    )
