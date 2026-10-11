package entities

import (
	"strings"

	"github.com/google/uuid"

	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/domain/errors"
)

// Knowledge represents metadata for a knowledge file belonging to a Store.
//
// Store owns this metadata. The file bytes and embedding data are owned by
// their respective infrastructure services.
type Knowledge struct {
	id               uuid.UUID
	storeID          uuid.UUID
	filename         string
	contentType      string
	storageReference string
}

// NewKnowledge creates knowledge metadata for a Store and generates its ID.
func NewKnowledge(
	storeID uuid.UUID,
	filename string,
	contentType string,
) (*Knowledge, error) {
	return ReconstituteKnowledge(
		uuid.New(),
		storeID,
		filename,
		contentType,
	)
}

// ReconstituteKnowledge restores knowledge metadata from persisted values.
func ReconstituteKnowledge(
	id uuid.UUID,
	storeID uuid.UUID,
	filename string,
	contentType string,
) (*Knowledge, error) {
	if id == uuid.Nil {
		return nil, domainerrors.ErrInvalidKnowledgeID
	}

	if storeID == uuid.Nil {
		return nil, domainerrors.ErrInvalidKnowledgeStoreID
	}

	if strings.TrimSpace(filename) == "" {
		return nil, domainerrors.ErrInvalidKnowledgeFilename
	}

	if strings.TrimSpace(contentType) == "" {
		return nil, domainerrors.ErrInvalidKnowledgeContentType
	}

	return &Knowledge{
		id:               id,
		storeID:          storeID,
		filename:         filename,
		contentType:      contentType,
		storageReference: "knowledge/" + storeID.String() + "/" + filename,
	}, nil
}

// ID returns the knowledge identifier.
func (k *Knowledge) ID() uuid.UUID {
	return k.id
}

// StoreID returns the owning Store identifier.
func (k *Knowledge) StoreID() uuid.UUID {
	return k.storeID
}

// Filename returns the original filename without modification.
func (k *Knowledge) Filename() string {
	return k.filename
}

// ContentType returns the file's media type.
func (k *Knowledge) ContentType() string {
	return k.contentType
}

// StorageReference returns the deterministic reference to the stored file.
func (k *Knowledge) StorageReference() string {
	return k.storageReference
}
