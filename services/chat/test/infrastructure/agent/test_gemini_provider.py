"""Tests for the Gemini LLM provider adapter."""

from types import SimpleNamespace
from typing import Any

import pytest
from google.genai import types

from infrastructure.agent.models import ModelMessage, ToolDefinition
from infrastructure.agent.providers.gemini import (
    GeminiLLMProvider,
    GeminiProviderError,
)


class StubModels:
    def __init__(self, response: Any = None, error: Exception | None = None) -> None:
        self.response = response
        self.error = error
        self.calls: list[dict[str, Any]] = []

    async def generate_content(self, **kwargs: Any) -> Any:
        self.calls.append(kwargs)
        if self.error is not None:
            raise self.error
        return self.response


class StubAsyncClient:
    def __init__(self, models: StubModels) -> None:
        self.models = models


def make_response(*parts: types.Part) -> Any:
    content = types.Content(role="model", parts=list(parts))
    return SimpleNamespace(
        candidates=[SimpleNamespace(content=content)]
    )


@pytest.mark.asyncio
async def test_generates_text_and_converts_system_instruction() -> None:
    models = StubModels(
        make_response(types.Part.from_text(text="Hello from Gemini"))
    )
    provider = GeminiLLMProvider(
        model="test-model",
        client=StubAsyncClient(models),
    )

    response = await provider.generate(
        messages=[
            ModelMessage(role="system", content="Be helpful."),
            ModelMessage(role="user", content="Hello"),
        ],
        tools=[],
    )

    assert response.content == "Hello from Gemini"
    assert response.tool_calls == ()
    assert models.calls[0]["model"] == "test-model"
    assert models.calls[0]["config"].system_instruction == "Be helpful."
    assert models.calls[0]["contents"][0].role == "user"


@pytest.mark.asyncio
async def test_translates_tool_declarations_and_preserves_function_call_content() -> None:
    original_content = types.Content(
        role="model",
        parts=[
            types.Part(
                function_call=types.FunctionCall(
                    id="call-1",
                    name="product_lookup",
                    args={"query": "running shoes"},
                ),
                thought_signature=b"test-signature",
            )
        ],
    )
    models = StubModels(
        SimpleNamespace(candidates=[SimpleNamespace(content=original_content)])
    )
    provider = GeminiLLMProvider(
        model="test-model",
        client=StubAsyncClient(models),
    )

    response = await provider.generate(
        messages=[ModelMessage(role="user", content="Find shoes")],
        tools=[
            ToolDefinition(
                name="product_lookup",
                description="Search products",
                input_schema={
                    "type": "object",
                    "properties": {"query": {"type": "string"}},
                    "required": ["query"],
                },
            )
        ],
    )

    assert response.content is None
    assert len(response.tool_calls) == 1
    assert response.tool_calls[0].id == "call-1"
    assert response.tool_calls[0].name == "product_lookup"
    assert response.tool_calls[0].arguments == {"query": "running shoes"}
    assert response.provider_data is original_content

    declaration = models.calls[0]["config"].tools[0].function_declarations[0]
    assert declaration.name == "product_lookup"
    assert declaration.parameters_json_schema["required"] == ["query"]


@pytest.mark.asyncio
async def test_rebuilds_tool_history_as_gemini_function_responses() -> None:
    original_assistant_content = types.Content(
        role="model",
        parts=[
            types.Part(
                function_call=types.FunctionCall(
                    id="call-1",
                    name="product_lookup",
                    args={"query": "shoes"},
                ),
                thought_signature=b"preserve-this",
            )
        ],
    )
    models = StubModels(make_response(types.Part.from_text(text="Here are shoes.")))
    provider = GeminiLLMProvider(
        model="test-model",
        client=StubAsyncClient(models),
    )

    await provider.generate(
        messages=[
            ModelMessage(role="user", content="Find shoes"),
            ModelMessage(
                role="assistant",
                tool_calls=(),
                provider_data=original_assistant_content,
            ),
            ModelMessage(
                role="tool",
                content='{"items": []}',
                tool_call_id="call-1",
                tool_name="product_lookup",
            ),
        ],
        tools=[],
    )

    contents = models.calls[0]["contents"]
    assert contents[1] is original_assistant_content
    function_response = contents[2].parts[0].function_response
    assert function_response.id == "call-1"
    assert function_response.name == "product_lookup"
    assert function_response.response == {"result": '{"items": []}'}


@pytest.mark.asyncio
async def test_rejects_response_without_candidates() -> None:
    provider = GeminiLLMProvider(
        model="test-model",
        client=StubAsyncClient(StubModels(response=SimpleNamespace(candidates=[]))),
    )

    with pytest.raises(GeminiProviderError, match="no response candidates"):
        await provider.generate(
            messages=[ModelMessage(role="user", content="Hello")],
            tools=[],
        )


@pytest.mark.asyncio
async def test_wraps_sdk_errors_without_exposing_error_details() -> None:
    provider = GeminiLLMProvider(
        model="test-model",
        client=StubAsyncClient(
            models=StubModels(error=RuntimeError("sensitive request details"))
        ),
    )

    with pytest.raises(GeminiProviderError) as captured:
        await provider.generate(
            messages=[ModelMessage(role="user", content="Hello")],
            tools=[],
        )

    assert "sensitive request details" not in str(captured.value)


def test_rejects_empty_model_name() -> None:
    with pytest.raises(ValueError, match="model must not be empty"):
        GeminiLLMProvider(model="  ", client=StubAsyncClient(StubModels()))
