package ports

import "context"

type KnowledgeFileDeleter interface {
	DeleteKnowledgeFile(ctx context.Context, reference string) error
}
