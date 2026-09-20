package entities

import (
	"time"

	"github.com/google/uuid"

	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/domain/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
)

// Store is the aggregate root for the Store domain.
type Store struct {
	id             valueobjects.StoreID
	ownerID        valueobjects.OwnerID
	name           valueobjects.StoreName
	slug           valueobjects.Slug
	description    valueobjects.Description
	imageReference *string
	status         valueobjects.Status
	plan           valueobjects.Plan
	createdAt      time.Time
	updatedAt      time.Time
}

// NewStore creates a new Store aggregate.
func NewStore(
	ownerID uuid.UUID,
	name string,
	slug string,
	description string,
	imageReference *string,
	planType string,
) (*Store, error) {
	storeID, err := valueobjects.NewStoreID(uuid.New())
	if err != nil {
		return nil, err
	}

	owner, err := valueobjects.NewOwnerID(ownerID)
	if err != nil {
		return nil, err
	}

	storeName, err := valueobjects.NewStoreName(name)
	if err != nil {
		return nil, err
	}

	storeSlug, err := valueobjects.NewSlug(slug)
	if err != nil {
		return nil, err
	}

	storeDescription, err := valueobjects.NewDescription(description)
	if err != nil {
		return nil, err
	}

	plan, err := valueobjects.NewPlan(valueobjects.PlanType(planType))
	if err != nil {
		return nil, err
	}
	imageReference = imageReference

	now := time.Now().UTC()

	return &Store{
		id:             storeID,
		ownerID:        owner,
		name:           storeName,
		slug:           storeSlug,
		description:    storeDescription,
		imageReference: imageReference,
		status:         valueobjects.StatusActive,
		plan:           plan,
		createdAt:      now,
		updatedAt:      now,
	}, nil
}

// ReconstituteStore restores a Store aggregate from persisted primitive values.
func ReconstituteStore(
	storeID uuid.UUID,
	ownerID uuid.UUID,
	name string,
	slug string,
	description string,
	imageReference *string,
	planType string,
	status string,
	createdAt time.Time,
	updatedAt time.Time,
) (*Store, error) {
	id, err := valueobjects.NewStoreID(storeID)
	if err != nil {
		return nil, err
	}

	owner, err := valueobjects.NewOwnerID(ownerID)
	if err != nil {
		return nil, err
	}

	storeName, err := valueobjects.NewStoreName(name)
	if err != nil {
		return nil, err
	}

	storeSlug, err := valueobjects.NewSlug(slug)
	if err != nil {
		return nil, err
	}

	storeDescription, err := valueobjects.NewDescription(description)
	if err != nil {
		return nil, err
	}

	plan, err := valueobjects.NewPlan(valueobjects.PlanType(planType))
	if err != nil {
		return nil, err
	}

	storeStatus, err := valueobjects.NewStatus(valueobjects.Status(status))
	if err != nil {
		return nil, err
	}

	if createdAt.IsZero() || updatedAt.IsZero() {
		return nil, domainerrors.ErrInvalidStoreTimestamps
	}

	return &Store{
		id:             id,
		ownerID:        owner,
		name:           storeName,
		slug:           storeSlug,
		description:    storeDescription,
		imageReference: imageReference,
		status:         storeStatus,
		plan:           plan,
		createdAt:      createdAt,
		updatedAt:      updatedAt,
	}, nil
}

// ID returns the Store identifier.
func (s *Store) ID() valueobjects.StoreID {
	return s.id
}

// OwnerID returns the Store owner identifier.
func (s *Store) OwnerID() valueobjects.OwnerID {
	return s.ownerID
}

// Name returns the Store name.
func (s *Store) Name() valueobjects.StoreName {
	return s.name
}

// Slug returns the Store slug.
func (s *Store) Slug() valueobjects.Slug {
	return s.slug
}

// Description returns the Store description.
func (s *Store) Description() valueobjects.Description {
	return s.description
}

// ImageReference returns the persisted reference to the Store image.
// A nil value means the Store has no image.
func (s *Store) ImageReference() *string {
	return s.imageReference
}

// SetImageReference assigns the Store image reference.
func (s *Store) SetImageReference(reference string) {
	s.imageReference = &reference
	s.touch()
}

// Status returns the Store lifecycle status.
func (s *Store) Status() valueobjects.Status {
	return s.status
}

// Plan returns the Store plan.
func (s *Store) Plan() valueobjects.Plan {
	return s.plan
}

// CreatedAt returns the Store creation timestamp.
func (s *Store) CreatedAt() time.Time {
	return s.createdAt
}

// UpdatedAt returns the Store last-modified timestamp.
func (s *Store) UpdatedAt() time.Time {
	return s.updatedAt
}

// ChangeName changes the Store name.
func (s *Store) ChangeName(name string) error {
	newName, err := valueobjects.NewStoreName(name)
	if err != nil {
		return err
	}

	if s.name.Value() == newName.Value() {
		return nil
	}

	s.name = newName
	s.touch()

	return nil
}

// ChangeSlug changes the Store slug.
func (s *Store) ChangeSlug(slug string) error {
	newSlug, err := valueobjects.NewSlug(slug)
	if err != nil {
		return err
	}

	if s.slug.Value() == newSlug.Value() {
		return nil
	}

	s.slug = newSlug
	s.touch()

	return nil
}

// ChangeDescription changes the Store description.
func (s *Store) ChangeDescription(description string) error {
	newDescription, err := valueobjects.NewDescription(description)
	if err != nil {
		return err
	}

	if s.description.Value() == newDescription.Value() {
		return nil
	}

	s.description = newDescription
	s.touch()

	return nil
}

// Activate changes an inactive Store to active.
func (s *Store) Activate() error {
	if !s.status.CanTransitionTo(valueobjects.StatusActive) {
		return domainerrors.ErrInvalidStatusTransition
	}

	if s.status == valueobjects.StatusActive {
		return nil
	}

	s.status = valueobjects.StatusActive
	s.touch()

	return nil
}

// Deactivate changes an active Store to inactive.
func (s *Store) Deactivate() error {
	if !s.status.CanTransitionTo(valueobjects.StatusInactive) {
		return domainerrors.ErrInvalidStatusTransition
	}

	if s.status == valueobjects.StatusInactive {
		return nil
	}

	s.status = valueobjects.StatusInactive
	s.touch()

	return nil
}

// ChangePlan changes the Store plan.
//
// Product count is intentionally not part of the Store aggregate.
// The application layer obtains the current count from the Product
// boundary and validates it against the target Plan before calling this
// method.
func (s *Store) ChangePlan(newPlan valueobjects.Plan) error {
	if newPlan == nil {
		return domainerrors.ErrInvalidPlanType
	}

	if s.plan.Type() == newPlan.Type() {
		return nil
	}

	s.plan = newPlan
	s.touch()

	return nil
}

// IsOwnedBy determines whether the supplied user owns the Store.
func (s *Store) IsOwnedBy(ownerID uuid.UUID) bool {
	return s.ownerID.Value() == ownerID
}

// touch updates the aggregate modification timestamp.
func (s *Store) touch() {
	s.updatedAt = time.Now().UTC()
}
