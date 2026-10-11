"""Tests for the store knowledge Agent tool."""

import json
from uuid import uuid4

import pytest

from infrastructure.agent.models import ToolContext
from infrastructure.agent.ports.knowledge_retrieval import KnowledgeChunk
from infrastructure.agent.tools.knowledge import SearchStoreKnowledgeTool


class FakeKnowledgeRetrieval:
    def __init__(self, chunks: list[KnowledgeChunk]) -> None:
        self.chunks = chunks
        self.store_id = None
        self.query = None

    async def retrieve_chunks(self, *, store_id, query):
        self.store_id = store_id
        self.query = query
        return self.chunks


@pytest.fixture
def context() -> ToolContext:
    return ToolContext(
        store_id=uuid4(),
        conversation_id=uuid4(),
        current_message_id=uuid4(),
    )


@pytest.mark.asyncio
async def test_search_passes_trusted_store_id_and_query(context: ToolContext):
    retrieval = FakeKnowledgeRetrieval(
        [
            KnowledgeChunk(
                content="Returns are accepted within 30 days.",
                source_filename="returns.pdf",
                score=0.91,
            )
        ]
    )
    tool = SearchStoreKnowledgeTool(retrieval=retrieval)

    result = json.loads(
        await tool.execute(
            arguments={"query": " What is the return policy? "},
            context=context,
        )
    )

    assert retrieval.store_id == context.store_id
    assert retrieval.query == "What is the return policy?"
    assert result == {
        "status": "ok",
        "chunks": [
            {
                "content": "Returns are accepted within 30 days.",
                "source_filename": "returns.pdf",
                "score": 0.91,
            }
        ],
    }


@pytest.mark.asyncio
async def test_search_reports_when_no_relevant_knowledge(context: ToolContext):
    tool = SearchStoreKnowledgeTool(
        retrieval=FakeKnowledgeRetrieval(chunks=[])
    )

    result = json.loads(
        await tool.execute(
            arguments={"query": "unrelated question"},
            context=context,
        )
    )

    assert result == {"status": "no_relevant_knowledge", "chunks": []}


@pytest.mark.asyncio
@pytest.mark.parametrize("query", ["", "  ", None, 123])
async def test_search_rejects_invalid_query_without_calling_service(
    context: ToolContext,
    query,
):
    retrieval = FakeKnowledgeRetrieval(chunks=[])
    tool = SearchStoreKnowledgeTool(retrieval=retrieval)

    result = json.loads(
        await tool.execute(
            arguments={"query": query},
            context=context,
        )
    )

    assert result == {"status": "invalid_query", "chunks": []}
    assert retrieval.store_id is None
    assert retrieval.query is None


@pytest.mark.asyncio
async def test_search_propagates_retrieval_failures(context: ToolContext):
    class FailingRetrieval:
        async def retrieve_chunks(self, *, store_id, query):
            raise RuntimeError("service unavailable")

    tool = SearchStoreKnowledgeTool(retrieval=FailingRetrieval())

    with pytest.raises(RuntimeError, match="service unavailable"):
        await tool.execute(
            arguments={"query": "return policy"},
            context=context,
        )
