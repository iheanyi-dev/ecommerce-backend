package entities_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/entities"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/domain/errors"
)

func TestNewKnowledge(t *testing.T) {
	storeID := uuid.New()

	knowledge, err := entities.NewKnowledge(
		storeID,
		"returns-policy.pdf",
		"application/pdf",
	)

	require.NoError(t, err)
	require.NotNil(t, knowledge)

	assert.NotEqual(t, uuid.Nil, knowledge.ID())
	assert.Equal(t, storeID, knowledge.StoreID())
	assert.Equal(t, "returns-policy.pdf", knowledge.Filename())
	assert.Equal(t, "application/pdf", knowledge.ContentType())
	assert.Equal(
		t,
		"knowledge/"+storeID.String()+"/returns-policy.pdf",
		knowledge.StorageReference(),
	)
}

func TestReconstituteKnowledge(t *testing.T) {
	id := uuid.New()
	storeID := uuid.New()
	filename := "Store Returns Policy (2026).PDF"

	knowledge, err := entities.ReconstituteKnowledge(
		id,
		storeID,
		filename,
		"application/pdf",
	)

	require.NoError(t, err)
	require.NotNil(t, knowledge)

	assert.Equal(t, id, knowledge.ID())
	assert.Equal(t, storeID, knowledge.StoreID())
	assert.Equal(t, filename, knowledge.Filename())
	assert.Equal(t, "application/pdf", knowledge.ContentType())
	assert.Equal(
		t,
		"knowledge/"+storeID.String()+"/"+filename,
		knowledge.StorageReference(),
	)
}

func TestReconstituteKnowledge_RejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name        string
		id          uuid.UUID
		storeID     uuid.UUID
		filename    string
		contentType string
		expectedErr error
	}{
		{
			name:        "nil knowledge ID",
			id:          uuid.Nil,
			storeID:     uuid.New(),
			filename:    "policy.pdf",
			contentType: "application/pdf",
			expectedErr: domainerrors.ErrInvalidKnowledgeID,
		},
		{
			name:        "nil Store ID",
			id:          uuid.New(),
			storeID:     uuid.Nil,
			filename:    "policy.pdf",
			contentType: "application/pdf",
			expectedErr: domainerrors.ErrInvalidKnowledgeStoreID,
		},
		{
			name:        "empty filename",
			id:          uuid.New(),
			storeID:     uuid.New(),
			filename:    "",
			contentType: "application/pdf",
			expectedErr: domainerrors.ErrInvalidKnowledgeFilename,
		},
		{
			name:        "whitespace filename",
			id:          uuid.New(),
			storeID:     uuid.New(),
			filename:    "   ",
			contentType: "application/pdf",
			expectedErr: domainerrors.ErrInvalidKnowledgeFilename,
		},
		{
			name:        "empty content type",
			id:          uuid.New(),
			storeID:     uuid.New(),
			filename:    "policy.pdf",
			contentType: "",
			expectedErr: domainerrors.ErrInvalidKnowledgeContentType,
		},
		{
			name:        "whitespace content type",
			id:          uuid.New(),
			storeID:     uuid.New(),
			filename:    "policy.pdf",
			contentType: "   ",
			expectedErr: domainerrors.ErrInvalidKnowledgeContentType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			knowledge, err := entities.ReconstituteKnowledge(
				tt.id,
				tt.storeID,
				tt.filename,
				tt.contentType,
			)

			require.ErrorIs(t, err, tt.expectedErr)
			assert.Nil(t, knowledge)
		})
	}
}

func TestNewKnowledge_PreservesFilenameExactly(t *testing.T) {
	storeID := uuid.New()
	filename := "Store Returns Policy (2026).PDF"

	knowledge, err := entities.NewKnowledge(
		storeID,
		filename,
		"application/pdf",
	)

	require.NoError(t, err)
	assert.Equal(t, filename, knowledge.Filename())
	assert.Equal(
		t,
		"knowledge/"+storeID.String()+"/"+filename,
		knowledge.StorageReference(),
	)
}
