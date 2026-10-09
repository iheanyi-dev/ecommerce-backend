"""Gemini implementation of the provider-neutral LLM contract."""

from collections.abc import Sequence
from typing import Any

from google import genai
from google.genai import types

from infrastructure.agent.models import (
    ModelMessage,
    ModelResponse,
    ToolCall,
    ToolDefinition,
)
from infrastructure.agent.ports.llm_provider import LLMProvider


class GeminiProviderError(RuntimeError):
    """Raised when Gemini cannot produce a usable model response."""


class GeminiLLMProvider(LLMProvider):
    """Generate model responses through Google's asynchronous GenAI SDK.

    The API key can be supplied directly or provided through the SDK's
    supported environment variables. The client is owned by this adapter
    unless an async client is injected, which is useful for testing.
    """

    def __init__(
        self,
        *,
        model: str,
        api_key: str | None = None,
        client: Any | None = None,
    ) -> None:
        if not model.strip():
            raise ValueError("Gemini model must not be empty")

        self._model = model
        self._owned_client = genai.Client(api_key=api_key) if client is None else None
        self._client = client if client is not None else self._owned_client.aio

    async def generate(
        self,
        *,
        messages: list[ModelMessage],
        tools: list[ToolDefinition],
    ) -> ModelResponse:
        """Call Gemini and translate its response into provider-neutral types."""
        system_instruction = "\n\n".join(
            message.content or ""
            for message in messages
            if message.role == "system"
        ).strip()

        contents = self._build_contents(messages)
        declarations = [
            types.FunctionDeclaration(
                name=tool.name,
                description=tool.description,
                parameters_json_schema=tool.input_schema,
            )
            for tool in tools
        ]

        config = types.GenerateContentConfig(
            system_instruction=system_instruction or None,
            tools=[types.Tool(function_declarations=declarations)]
            if declarations
            else None,
        )

        try:
            response = await self._client.models.generate_content(
                model=self._model,
                contents=contents,
                config=config,
            )
        except Exception as exc:
            # Avoid exposing SDK error text, request details, or credentials
            # through the adapter's public exception message.
            raise GeminiProviderError("Gemini content generation failed") from exc

        return self._parse_response(response)

    async def aclose(self) -> None:
        """Release SDK resources when this adapter created the client."""
        if self._owned_client is not None:
            await self._client.aclose()
            self._owned_client.close()

    @staticmethod
    def _build_contents(
        messages: Sequence[ModelMessage],
    ) -> list[types.Content]:
        """Convert the conversation transcript into Gemini content objects."""
        contents: list[types.Content] = []
        pending_function_responses: list[types.Part] = []

        def flush_function_responses() -> None:
            if pending_function_responses:
                contents.append(
                    types.Content(
                        role="user",
                        parts=list(pending_function_responses),
                    )
                )
                pending_function_responses.clear()

        for message in messages:
            if message.role == "system":
                continue

            if message.role == "tool":
                if not message.tool_name:
                    raise GeminiProviderError(
                        "Tool transcript entry is missing its tool name"
                    )

                pending_function_responses.append(
                    types.Part(
                        function_response=types.FunctionResponse(
                            id=message.tool_call_id or None,
                            name=message.tool_name,
                            response={"result": message.content or ""},
                        )
                    )
                )
                continue

            flush_function_responses()

            if message.role == "assistant":
                if isinstance(message.provider_data, types.Content):
                    # Preserve the original response parts, including any
                    # provider-specific signatures needed for function calls.
                    contents.append(message.provider_data)
                else:
                    contents.append(
                        types.Content(
                            role="model",
                            parts=[types.Part.from_text(text=message.content or "")],
                        )
                    )
            elif message.role == "user":
                contents.append(
                    types.Content(
                        role="user",
                        parts=[types.Part.from_text(text=message.content or "")],
                    )
                )
            else:
                raise GeminiProviderError(
                    f"Unsupported transcript role: {message.role}"
                )

        flush_function_responses()
        return contents

    @staticmethod
    def _parse_response(response: Any) -> ModelResponse:
        """Translate Gemini's first candidate into the shared response model."""
        candidates = getattr(response, "candidates", None)
        if not candidates:
            raise GeminiProviderError("Gemini returned no response candidates")

        candidate = candidates[0]
        content = getattr(candidate, "content", None)
        if content is None:
            raise GeminiProviderError("Gemini response candidate has no content")

        text_parts: list[str] = []
        tool_calls: list[ToolCall] = []

        for part in content.parts or []:
            # Do not expose model thought parts as customer-facing text.
            if part.text is not None and not getattr(part, "thought", False):
                text_parts.append(part.text)

            function_call = part.function_call
            if function_call is None:
                continue

            name = function_call.name
            if not name:
                raise GeminiProviderError("Gemini returned a function call without a name")

            arguments = function_call.args or {}
            if not isinstance(arguments, dict):
                raise GeminiProviderError(
                    "Gemini returned function arguments that are not an object"
                )

            tool_calls.append(
                ToolCall(
                    id=function_call.id or "",
                    name=name,
                    arguments=arguments,
                )
            )

        return ModelResponse(
            content="".join(text_parts) or None,
            tool_calls=tuple(tool_calls),
            provider_data=content,
        )
