package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
)

func newStoreOutput(storeID uuid.UUID) dto.StoreOutput {
	now := time.Now().UTC()

	return dto.StoreOutput{
		ID:          mustStoreID(storeID),
		OwnerID:     mustOwnerID(uuid.New()),
		Name:        mustStoreName("My Store"),
		Slug:        mustSlug("my-store"),
		Description: mustDescription("My store description"),
		Status:      mustStatus(valueobjects.StatusActive),
		Plan:        mustPlan(valueobjects.PlanTypeBasic),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func assertBadRequest(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusBadRequest,
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

func assertErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder, status int, message string) {
	t.Helper()

	if recorder.Code != status {
		t.Fatalf("expected status %d, got %d", status, recorder.Code)
	}

	var response struct {
		Error string `json:"error"`
	}

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}

	if response.Error != message {
		t.Fatalf("expected error %q, got %q", message, response.Error)
	}
}
