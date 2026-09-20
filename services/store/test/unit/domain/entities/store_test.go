package entities_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/entities"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/domain/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
)

func TestNewStore(t *testing.T) {
	ownerID, err := valueobjects.NewOwnerID(uuid.New())
	require.NoError(t, err)

	name, err := valueobjects.NewStoreName("My Store")
	require.NoError(t, err)

	slug, err := valueobjects.NewSlug("my-store")
	require.NoError(t, err)

	description, err := valueobjects.NewDescription("My store description")
	require.NoError(t, err)

	plan, err := valueobjects.NewPlan(valueobjects.PlanTypeBasic)
	require.NoError(t, err)

	before := time.Now().UTC()

	store, err := entities.NewStore(
		ownerID.Value(),
		name.Value(),
		slug.Value(),
		description.Value(),
		nil,
		string(plan.Type()),
	)

	after := time.Now().UTC()

	require.NoError(t, err)
	require.NotNil(t, store)

	assert.NotEqual(t, uuid.Nil, store.ID().Value())
	assert.Equal(t, ownerID, store.OwnerID())
	assert.Equal(t, name, store.Name())
	assert.Equal(t, slug, store.Slug())
	assert.Equal(t, description, store.Description())
	assert.Equal(t, valueobjects.StatusActive, store.Status())
	assert.Equal(t, plan.Type(), store.Plan().Type())
	assert.False(t, store.CreatedAt().Before(before))
	assert.False(t, store.CreatedAt().After(after))
	assert.Equal(t, store.CreatedAt(), store.UpdatedAt())
	assert.Nil(t, store.ImageReference())
}

func TestReconstituteStore(t *testing.T) {
	storeID := uuid.New()
	ownerID := uuid.New()
	createdAt := time.Date(2026, time.September, 1, 10, 30, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.September, 10, 15, 45, 0, 0, time.UTC)
	imageReference := storeID.String() + ".webp"
	store, err := entities.ReconstituteStore(
		storeID,
		ownerID,
		"Existing Store",
		"existing-store",
		"Existing description",
		&imageReference,
		"premium",
		"active",
		createdAt,
		updatedAt,
	)

	require.NoError(t, err)
	require.NotNil(t, store)
	require.NotNil(t, store.ImageReference())
	assert.Equal(t, imageReference, *store.ImageReference())
	assert.Equal(t, storeID, store.ID().Value())
	assert.Equal(t, ownerID, store.OwnerID().Value())
	assert.Equal(t, "Existing Store", store.Name().Value())
	assert.Equal(t, "existing-store", store.Slug().Value())
	assert.Equal(t, "Existing description", store.Description().Value())
	assert.Equal(t, valueobjects.PlanTypePremium, store.Plan().Type())
	assert.Equal(t, valueobjects.StatusActive, store.Status())
	assert.Equal(t, createdAt, store.CreatedAt())
	assert.Equal(t, updatedAt, store.UpdatedAt())
}

func TestReconstituteStore_RejectsInvalidPersistedValues(t *testing.T) {
	tests := []struct {
		name        string
		storeID     uuid.UUID
		ownerID     uuid.UUID
		nameValue   string
		slug        string
		description string
		status      string
		plan        string
	}{
		{
			name:      "invalid store id",
			storeID:   uuid.Nil,
			ownerID:   uuid.New(),
			nameValue: "Store",
			slug:      "store",
			status:    "active",
			plan:      "basic",
		},
		{
			name:      "invalid owner id",
			storeID:   uuid.New(),
			ownerID:   uuid.Nil,
			nameValue: "Store",
			slug:      "store",
			status:    "active",
			plan:      "basic",
		},
		{
			name:      "invalid name",
			storeID:   uuid.New(),
			ownerID:   uuid.New(),
			nameValue: "",
			slug:      "store",
			status:    "active",
			plan:      "basic",
		},
		{
			name:      "invalid slug",
			storeID:   uuid.New(),
			ownerID:   uuid.New(),
			nameValue: "Store",
			slug:      "invalid_slug",
			status:    "active",
			plan:      "basic",
		},
		{
			name:      "invalid status",
			storeID:   uuid.New(),
			ownerID:   uuid.New(),
			nameValue: "Store",
			slug:      "store",
			status:    "invalid",
			plan:      "basic",
		},
		{
			name:      "invalid plan",
			storeID:   uuid.New(),
			ownerID:   uuid.New(),
			nameValue: "Store",
			slug:      "store",
			status:    "active",
			plan:      "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := entities.ReconstituteStore(
				tt.storeID,
				tt.ownerID,
				tt.nameValue,
				tt.slug,
				tt.description,
				nil,
				tt.status,
				tt.plan,
				time.Now().UTC(),
				time.Now().UTC(),
			)

			require.Error(t, err)
			assert.Nil(t, store)
		})
	}
}

func TestStore_Deactivate(t *testing.T) {
	store := newTestStore(t)
	originalUpdatedAt := store.UpdatedAt()

	time.Sleep(time.Millisecond)

	err := store.Deactivate()

	require.NoError(t, err)
	assert.Equal(t, valueobjects.StatusInactive, store.Status())
	assert.True(t, store.UpdatedAt().After(originalUpdatedAt))
}

func TestStore_Activate(t *testing.T) {
	store := newTestStore(t)

	require.NoError(t, store.Deactivate())
	originalUpdatedAt := store.UpdatedAt()

	time.Sleep(time.Millisecond)

	err := store.Activate()

	require.NoError(t, err)
	assert.Equal(t, valueobjects.StatusActive, store.Status())
	assert.True(t, store.UpdatedAt().After(originalUpdatedAt))
}

func TestStore_Deactivate_WhenAlreadyInactive(t *testing.T) {
	store := newTestStore(t)

	require.NoError(t, store.Deactivate())
	originalUpdatedAt := store.UpdatedAt()

	err := store.Deactivate()

	require.NoError(t, err)
	assert.Equal(t, valueobjects.StatusInactive, store.Status())
	assert.Equal(t, originalUpdatedAt, store.UpdatedAt())
}

func TestStore_Activate_WhenAlreadyActive(t *testing.T) {
	store := newTestStore(t)
	originalUpdatedAt := store.UpdatedAt()

	err := store.Activate()

	require.NoError(t, err)
	assert.Equal(t, valueobjects.StatusActive, store.Status())
	assert.Equal(t, originalUpdatedAt, store.UpdatedAt())
}

func newTestStore(t *testing.T) *entities.Store {
	t.Helper()

	ownerID := uuid.New()

	store, err := entities.NewStore(
		ownerID,
		"My Store",
		"my-store",
		"",
		nil,
		"basic",
	)

	require.NoError(t, err)

	return store
}

func TestStore_ChangeName(t *testing.T) {
	store, err := entities.NewStore(
		uuid.New(),
		"Original Store",
		"original-store",
		"Original description",
		nil,
		"basic",
	)
	require.NoError(t, err)

	originalUpdatedAt := store.UpdatedAt()
	time.Sleep(time.Millisecond)

	err = store.ChangeName("Updated Store")

	require.NoError(t, err)
	assert.Equal(t, "Updated Store", store.Name().Value())
	assert.True(t, store.UpdatedAt().After(originalUpdatedAt))
}

func TestStore_ChangeName_RejectsInvalidValue(t *testing.T) {
	store, err := entities.NewStore(
		uuid.New(),
		"Original Store",
		"original-store",
		"Original description",
		nil,
		"basic",
	)
	require.NoError(t, err)

	err = store.ChangeName("")

	require.ErrorIs(t, err, domainerrors.ErrInvalidStoreName)
	assert.Equal(t, "Original Store", store.Name().Value())
}

func TestStore_ChangeSlug(t *testing.T) {
	store, err := entities.NewStore(
		uuid.New(),
		"Original Store",
		"original-store",
		"Original description",
		nil,
		"basic",
	)
	require.NoError(t, err)

	originalUpdatedAt := store.UpdatedAt()
	time.Sleep(time.Millisecond)

	err = store.ChangeSlug("updated-store")

	require.NoError(t, err)
	assert.Equal(t, "updated-store", store.Slug().Value())
	assert.True(t, store.UpdatedAt().After(originalUpdatedAt))
}

func TestStore_ChangeSlug_RejectsInvalidValue(t *testing.T) {
	store, err := entities.NewStore(
		uuid.New(),
		"Original Store",
		"original-store",
		"Original description",
		nil,
		"basic",
	)
	require.NoError(t, err)

	err = store.ChangeSlug("invalid_slug")

	require.ErrorIs(t, err, domainerrors.ErrInvalidStoreSlug)
	assert.Equal(t, "original-store", store.Slug().Value())
}

func TestStore_ChangeDescription(t *testing.T) {
	store, err := entities.NewStore(
		uuid.New(),
		"Original Store",
		"original-store",
		"Original description",
		nil,
		"basic",
	)
	require.NoError(t, err)

	originalUpdatedAt := store.UpdatedAt()
	time.Sleep(time.Millisecond)

	err = store.ChangeDescription("Updated description")

	require.NoError(t, err)
	assert.Equal(t, "Updated description", store.Description().Value())
	assert.True(t, store.UpdatedAt().After(originalUpdatedAt))
}

func TestStore_ChangeDescription_RejectsInvalidValue(t *testing.T) {
	store, err := entities.NewStore(
		uuid.New(),
		"Original Store",
		"original-store",
		"Original description",
		nil,
		"basic",
	)
	require.NoError(t, err)

	err = store.ChangeDescription(string(make([]rune, 501)))

	require.ErrorIs(t, err, domainerrors.ErrInvalidStoreDescription)
	assert.Equal(t, "Original description", store.Description().Value())
}

func TestStore_Activate_IsIdempotent(t *testing.T) {
	store, err := entities.NewStore(
		uuid.New(),
		"Store",
		"store",
		"Description",
		nil,
		"basic",
	)
	require.NoError(t, err)

	originalUpdatedAt := store.UpdatedAt()
	time.Sleep(time.Millisecond)

	err = store.Activate()

	require.NoError(t, err)
	assert.Equal(t, originalUpdatedAt, store.UpdatedAt())
}

func TestStore_Deactivate_IsIdempotent(t *testing.T) {
	store, err := entities.NewStore(
		uuid.New(),
		"Store",
		"store",
		"Description",
		nil,
		"basic",
	)
	require.NoError(t, err)

	err = store.Deactivate()
	require.NoError(t, err)

	originalUpdatedAt := store.UpdatedAt()
	time.Sleep(time.Millisecond)

	err = store.Deactivate()

	require.NoError(t, err)
	assert.Equal(t, originalUpdatedAt, store.UpdatedAt())
}

func TestStore_ChangePlan(t *testing.T) {
	store, err := entities.NewStore(
		uuid.New(),
		"Store",
		"store",
		"Description",
		nil,
		"basic",
	)
	require.NoError(t, err)

	newPlan, err := valueobjects.NewPlan(valueobjects.PlanTypePremium)
	require.NoError(t, err)

	originalUpdatedAt := store.UpdatedAt()
	time.Sleep(time.Millisecond)

	err = store.ChangePlan(newPlan)

	require.NoError(t, err)
	assert.Equal(t, valueobjects.PlanTypePremium, store.Plan().Type())
	assert.True(t, store.UpdatedAt().After(originalUpdatedAt))
}

func TestStore_ChangePlan_IsIdempotent(t *testing.T) {
	store, err := entities.NewStore(
		uuid.New(),
		"Store",
		"store",
		"Description",
		nil,
		"basic",
	)
	require.NoError(t, err)

	originalUpdatedAt := store.UpdatedAt()
	time.Sleep(time.Millisecond)

	currentPlan, err := valueobjects.NewPlan(valueobjects.PlanTypeBasic)
	require.NoError(t, err)

	err = store.ChangePlan(currentPlan)

	require.NoError(t, err)
	assert.Equal(t, valueobjects.PlanTypeBasic, store.Plan().Type())
	assert.Equal(t, originalUpdatedAt, store.UpdatedAt())
}

func TestStore_IsOwnedBy(t *testing.T) {
	ownerID := uuid.New()

	store, err := entities.NewStore(
		ownerID,
		"Store",
		"store",
		"Description",
		nil,
		"basic",
	)
	require.NoError(t, err)

	assert.True(t, store.IsOwnedBy(ownerID))
	assert.False(t, store.IsOwnedBy(uuid.New()))
}
func TestStore_SetImageReference(t *testing.T) {
	store := newTestStore(t)

	originalUpdatedAt := store.UpdatedAt()
	time.Sleep(time.Millisecond)

	reference := store.ID().Value().String() + ".webp"

	store.SetImageReference(reference)

	require.NotNil(t, store.ImageReference())
	assert.Equal(t, reference, *store.ImageReference())
	assert.True(t, store.UpdatedAt().After(originalUpdatedAt))
}
