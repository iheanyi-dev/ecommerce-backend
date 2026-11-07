package behavioral_models_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/infrastructure/behavioral_models"
)

func TestEmptyProductBehavioralAssociationModelReturnsNoAssociations(t *testing.T) {
	model := &behavioral_models.EmptyProductBehavioralAssociationModel{}

	got, err := model.FindAssociated(context.Background(), uuid.New(), 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no associations, got %d", len(got))
	}
}

func TestEmptyStoreBehavioralAssociationModelReturnsNoAssociations(t *testing.T) {
	model := &behavioral_models.EmptyStoreBehavioralAssociationModel{}

	got, err := model.FindAssociated(context.Background(), uuid.New(), 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no associations, got %d", len(got))
	}
}
