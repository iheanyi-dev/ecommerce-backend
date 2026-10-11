"""Agent tool for retrieving relevant knowledge from the current store."""

import json
from typing import Any

from infrastructure.agent.models import ToolContext
from infrastructure.agent.ports.knowledge_retrieval import KnowledgeRetrieval


class SearchStoreKnowledgeTool:
    """Search store knowledge through the embedding service."""

    name = "search_store_knowledge"
    description = (
        "Search the current store's uploaded knowledge for information "
        "relevant to the customer's question. Use the returned excerpts "
        "as evidence when answering. If no relevant excerpts are returned, "
        "do not invent information."
    )
    input_schema: dict[str, Any] = {
        "type": "object",
        "properties": {
            "query": {
                "type": "string",
                "description": (
                    "The customer's question or the topic to search for."
                ),
            }
        },
        "required": ["query"],
        "additionalProperties": False,
    }

    def __init__(self, *, retrieval: KnowledgeRetrieval) -> None:
        self._retrieval = retrieval

    async def execute(
        self,
        *,
        arguments: dict[str, Any],
        context: ToolContext,
    ) -> str:
        query = arguments.get("query")
        if not isinstance(query, str) or not query.strip():
            return json.dumps(
                {"status": "invalid_query", "chunks": []},
                ensure_ascii=False,
            )

        chunks = await self._retrieval.retrieve_chunks(
            store_id=context.store_id,
            query=query.strip(),
        )

        return json.dumps(
            {
                "status": "ok" if chunks else "no_relevant_knowledge",
                "chunks": [
                    {
                        "content": chunk.content,
                        "source_filename": chunk.source_filename,
                        "score": chunk.score,
                    }
                    for chunk in chunks
                ],
            },
            ensure_ascii=False,
        )
