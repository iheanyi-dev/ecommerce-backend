"""Application use case for ingesting a knowledge document."""

from dataclasses import dataclass
from uuid import uuid4

from domain.entities import KnowledgeChunk
from domain.value_objects import EntityID, StorageReference

from ..dto.ingest_knowledge import (
    IngestKnowledgeInput,
    IngestKnowledgeOutput,
)
from ..ports import (
    DocumentContentReader,
    EmbeddingModel,
    KnowledgeChunkRepository,
    Logger,
    Metrics,
    TextChunker,
)


@dataclass(slots=True)
class IngestKnowledge:
    """Extract, chunk, embed, and persist a knowledge document."""

    document_content_reader: DocumentContentReader
    text_chunker: TextChunker
    embedding_model: EmbeddingModel
    knowledge_chunk_repository: KnowledgeChunkRepository
    logger: Logger
    metrics: Metrics

    def execute(
        self, input: IngestKnowledgeInput
    ) -> IngestKnowledgeOutput:
        """Ingest a document, replacing its previous chunks on success."""

        knowledge_id = EntityID(input.knowledge_id, "knowledge")
        store_id = EntityID(input.store_id, "store")
        storage_reference = StorageReference(input.storage_reference)

        self.logger.info(
            "embedding.knowledge.ingest_started",
            knowledge_id=str(knowledge_id),
            store_id=str(store_id),
        )

        document_text = self.document_content_reader.read(storage_reference)
        if not isinstance(document_text, str) or not document_text.strip():
            raise ValueError("knowledge document contains no extractable text")

        raw_chunks = self.text_chunker.chunk(document_text)
        if not raw_chunks:
            raise ValueError("text chunker returned no chunks")

        chunks: list[KnowledgeChunk] = []
        model_versions = []

        for index, raw_chunk in enumerate(raw_chunks):
            if not isinstance(raw_chunk, str) or not raw_chunk.strip():
                raise ValueError(
                    f"text chunker returned empty or invalid content at index {index}"
                )

            content = raw_chunk.strip()
            embedding, model_version = self.embedding_model.generate(content)

            chunks.append(
                KnowledgeChunk(
                    id=uuid4(),
                    knowledge_id=input.knowledge_id,
                    store_id=input.store_id,
                    storage_reference=storage_reference.value,
                    content=content,
                    chunk_index=index,
                    embedding=embedding,
                    embedding_model_version=model_version,
                )
            )
            model_versions.append(model_version)

        # Delegate replacement to the repository so old chunks are not
        # deleted before document reading and all embedding generation succeed.
        self.knowledge_chunk_repository.replace_for_knowledge(
            knowledge_id, chunks
        )

        self.metrics.increment("embedding.knowledge_documents.ingested")
        self.metrics.observe(
            "embedding.knowledge_chunks.ingested", len(chunks)
        )
        self.logger.info(
            "embedding.knowledge.ingested",
            knowledge_id=str(knowledge_id),
            store_id=str(store_id),
            chunk_count=len(chunks),
        )

        return IngestKnowledgeOutput(
            knowledge_id=input.knowledge_id,
            store_id=input.store_id,
            chunk_count=len(chunks),
            embedding_model_versions=tuple(model_versions),
        )
