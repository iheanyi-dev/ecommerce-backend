package postgres_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/persistence/postgres"
)

func TestKnowledgeRepository_SaveAndGetByStoreID(t *testing.T) {
	db := NewTestDatabase(t)
	repository := postgres.NewKnowledgeRepository(postgres.NewQueries(db.Pool))
	ctx := context.Background()

	storeID := uuid.New()
	knowledge, err := entities.NewKnowledge(
		storeID,
		"catalog.pdf",
		"application/pdf",
	)
	if err != nil {
		t.Fatalf("create knowledge entity: %v", err)
	}

	if err := repository.Save(ctx, knowledge); err != nil {
		t.Fatalf("save knowledge: %v", err)
	}

	got, err := repository.GetByStoreID(ctx, storeID)
	if err != nil {
		t.Fatalf("get knowledge by store ID: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 knowledge record, got %d", len(got))
	}

	if got[0].ID() != knowledge.ID() {
		t.Errorf("ID mismatch: got %s, want %s", got[0].ID(), knowledge.ID())
	}
	if got[0].StoreID() != storeID {
		t.Errorf("StoreID mismatch: got %s, want %s", got[0].StoreID(), storeID)
	}
	if got[0].Filename() != "catalog.pdf" {
		t.Errorf("filename mismatch: got %q", got[0].Filename())
	}
	if got[0].ContentType() != "application/pdf" {
		t.Errorf("content type mismatch: got %q", got[0].ContentType())
	}
	if got[0].StorageReference() != "knowledge/"+storeID.String()+"/catalog.pdf" {
		t.Errorf("unexpected storage reference: %q", got[0].StorageReference())
	}
}

func TestKnowledgeRepository_StorageReferenceMustBeUnique(t *testing.T) {
	db := NewTestDatabase(t)
	repository := postgres.NewKnowledgeRepository(postgres.NewQueries(db.Pool))
	ctx := context.Background()

	storeID := uuid.New()
	first, err := entities.NewKnowledge(storeID, "duplicate.pdf", "application/pdf")
	if err != nil {
		t.Fatalf("create first knowledge: %v", err)
	}
	second, err := entities.NewKnowledge(storeID, "duplicate.pdf", "application/pdf")
	if err != nil {
		t.Fatalf("create second knowledge: %v", err)
	}

	if err := repository.Save(ctx, first); err != nil {
		t.Fatalf("save first knowledge: %v", err)
	}

	err = repository.Save(ctx, second)
	if err == nil {
		t.Fatal("expected duplicate storage reference to be rejected")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "duplicate key") &&
		!strings.Contains(strings.ToLower(err.Error()), "unique constraint") {
		t.Fatalf("expected unique-constraint error, got: %v", err)
	}
}

func TestKnowledgeRepository_DeleteByID(t *testing.T) {
	db := NewTestDatabase(t)
	repository := postgres.NewKnowledgeRepository(postgres.NewQueries(db.Pool))
	ctx := context.Background()

	storeID := uuid.New()
	knowledge, err := entities.NewKnowledge(storeID, "remove.txt", "text/plain")
	if err != nil {
		t.Fatalf("create knowledge entity: %v", err)
	}

	if err := repository.Save(ctx, knowledge); err != nil {
		t.Fatalf("save knowledge: %v", err)
	}
	if err := repository.DeleteByID(ctx, knowledge.ID()); err != nil {
		t.Fatalf("delete knowledge: %v", err)
	}

	got, err := repository.GetByStoreID(ctx, storeID)
	if err != nil {
		t.Fatalf("get knowledge after deletion: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no knowledge records after deletion, got %d", len(got))
	}
}
