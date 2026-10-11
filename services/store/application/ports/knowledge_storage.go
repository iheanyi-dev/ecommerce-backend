package ports

import "context"

// KnowledgeFile is a file that must be persisted in Storage.
type KnowledgeFile struct {
	Reference   string
	Content     []byte
	ContentType string
}

// KnowledgeStorage synchronously stores Knowledge files.
type KnowledgeStorage interface {
	PutKnowledgeFile(ctx context.Context, file KnowledgeFile) error
}
