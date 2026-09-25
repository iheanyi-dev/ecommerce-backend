package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/handlers"
)

type mockCreateStoreService struct {
	executeFunc func(
		ctx context.Context,
		input dto.CreateStoreInput,
	) (dto.CreateStoreOutput, error)
}

func (m *mockCreateStoreService) Execute(
	ctx context.Context,
	input dto.CreateStoreInput,
) (dto.CreateStoreOutput, error) {
	return m.executeFunc(ctx, input)
}

// TestCreateStoreHandler_Success verifies the complete successful
// presentation boundary.
//
// The handler must:
//   - translate multipart/form-data into the application DTO
//   - preserve the trusted authenticated identity in the request context
//   - invoke the application service
//   - translate the application result into the public JSON response schema
//
// HTTP-specific multipart types must never cross into the application layer.
func TestCreateStoreHandler_Success(t *testing.T) {
	storeID := uuid.New()
	ownerID := uuid.New()
	createdAt := time.Now().UTC()
	updatedAt := createdAt
	imageContent := []byte("fake-image-content")
	imageReference := "stores/" + storeID.String() + ".png"

	service := &mockCreateStoreService{
		executeFunc: func(
			ctx context.Context,
			input dto.CreateStoreInput,
		) (dto.CreateStoreOutput, error) {
			assertCreateStoreInput(t, ctx, input, ownerID, imageContent)

			return dto.CreateStoreOutput{
				ID:             mustStoreID(storeID),
				OwnerID:        mustOwnerID(ownerID),
				Name:           mustStoreName("My Store"),
				Slug:           mustSlug("my-store"),
				Description:    mustDescription("My store description"),
				ImageReference: &imageReference,
				Status:         mustStatus(valueobjects.StatusActive),
				Plan:           mustPlan(valueobjects.PlanTypeBasic),
				CreatedAt:      createdAt,
				UpdatedAt:      updatedAt,
			}, nil
		},
	}

	request := newCreateStoreMultipartRequest(
		t,
		"store.png",
		"image/png",
		imageContent,
	)

	request = request.WithContext(
		ports.WithAuthenticatedIdentity(
			request.Context(),
			ports.AuthenticatedIdentity{
				UserID: ownerID,
				Role:   "user",
			},
		),
	)

	recorder := httptest.NewRecorder()

	handlers.NewCreateStoreHandler(service).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusCreated,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf(
			"expected Content-Type %q, got %q",
			"application/json",
			contentType,
		)
	}

	var response struct {
		ID             string    `json:"id"`
		OwnerID        string    `json:"owner_id"`
		Name           string    `json:"name"`
		Slug           string    `json:"slug"`
		Description    string    `json:"description"`
		ImageReference *string   `json:"image_reference"`
		Status         string    `json:"status"`
		Plan           string    `json:"plan"`
		CreatedAt      time.Time `json:"created_at"`
		UpdatedAt      time.Time `json:"updated_at"`
	}

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}

	if response.ID != storeID.String() {
		t.Errorf("expected ID %q, got %q", storeID, response.ID)
	}

	if response.OwnerID != ownerID.String() {
		t.Errorf("expected owner ID %q, got %q", ownerID, response.OwnerID)
	}

	if response.Name != "My Store" {
		t.Errorf("expected name %q, got %q", "My Store", response.Name)
	}

	if response.Slug != "my-store" {
		t.Errorf("expected slug %q, got %q", "my-store", response.Slug)
	}

	if response.Description != "My store description" {
		t.Errorf(
			"expected description %q, got %q",
			"My store description",
			response.Description,
		)
	}

	if response.ImageReference == nil {
		t.Fatal("expected image reference")
	}

	if *response.ImageReference != imageReference {
		t.Errorf(
			"expected image reference %q, got %q",
			imageReference,
			*response.ImageReference,
		)
	}

	if response.Status != string(valueobjects.StatusActive) {
		t.Errorf(
			"expected status %q, got %q",
			valueobjects.StatusActive,
			response.Status,
		)
	}

	if response.Plan != string(valueobjects.PlanTypeBasic) {
		t.Errorf(
			"expected plan %q, got %q",
			valueobjects.PlanTypeBasic,
			response.Plan,
		)
	}

	if !response.CreatedAt.Equal(createdAt) {
		t.Errorf(
			"expected created_at %v, got %v",
			createdAt,
			response.CreatedAt,
		)
	}

	if !response.UpdatedAt.Equal(updatedAt) {
		t.Errorf(
			"expected updated_at %v, got %v",
			updatedAt,
			response.UpdatedAt,
		)
	}
}

// TestCreateStoreHandler_SuccessWithoutImage verifies that image upload is
// genuinely optional at the HTTP boundary.
func TestCreateStoreHandler_SuccessWithoutImage(t *testing.T) {
	ownerID := uuid.New()

	service := &mockCreateStoreService{
		executeFunc: func(
			ctx context.Context,
			input dto.CreateStoreInput,
		) (dto.CreateStoreOutput, error) {
			if input.Image != nil {
				t.Fatal("expected image input to be nil when no image was uploaded")
			}

			return dto.CreateStoreOutput{
				ID:          mustStoreID(uuid.New()),
				OwnerID:     mustOwnerID(ownerID),
				Name:        mustStoreName("My Store"),
				Slug:        mustSlug("my-store"),
				Description: mustDescription("My store description"),
				Status:      mustStatus(valueobjects.StatusActive),
				Plan:        mustPlan(valueobjects.PlanTypeBasic),
				CreatedAt:   time.Now().UTC(),
				UpdatedAt:   time.Now().UTC(),
			}, nil
		},
	}

	var body strings.Builder
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("name", "My Store"); err != nil {
		t.Fatalf("failed to write name field: %v", err)
	}

	if err := writer.WriteField("slug", "my-store"); err != nil {
		t.Fatalf("failed to write slug field: %v", err)
	}

	if err := writer.WriteField("description", "My store description"); err != nil {
		t.Fatalf("failed to write description field: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/stores",
		strings.NewReader(body.String()),
	)

	request.Header.Set("Content-Type", writer.FormDataContentType())
	request = request.WithContext(
		ports.WithAuthenticatedIdentity(
			request.Context(),
			ports.AuthenticatedIdentity{
				UserID: ownerID,
				Role:   "user",
			},
		),
	)

	recorder := httptest.NewRecorder()

	handlers.NewCreateStoreHandler(service).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusCreated,
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

// TestCreateStoreHandler_ApplicationError verifies that application errors
// are translated by the centralized Store presentation error translator.
func TestCreateStoreHandler_ApplicationError(t *testing.T) {
	service := &mockCreateStoreService{
		executeFunc: func(
			ctx context.Context,
			input dto.CreateStoreInput,
		) (dto.CreateStoreOutput, error) {
			return dto.CreateStoreOutput{}, application_errors.ErrStoreSlugAlreadyExists
		},
	}

	request := newCreateStoreMultipartRequest(
		t,
		"",
		"",
		nil,
	)

	recorder := httptest.NewRecorder()

	handlers.NewCreateStoreHandler(service).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusConflict,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var response struct {
		Error string `json:"error"`
	}

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if response.Error != "store slug already exists" {
		t.Errorf(
			"expected error %q, got %q",
			"store slug already exists",
			response.Error,
		)
	}
}

// TestCreateStoreHandler_MethodNotAllowed verifies that unsupported HTTP
// methods are rejected before the application layer is invoked.
func TestCreateStoreHandler_MethodNotAllowed(t *testing.T) {
	handler := handlers.NewCreateStoreHandler(
		&mockCreateStoreService{
			executeFunc: func(
				ctx context.Context,
				input dto.CreateStoreInput,
			) (dto.CreateStoreOutput, error) {
				t.Fatal("application service must not be called for an invalid HTTP method")
				return dto.CreateStoreOutput{}, nil
			},
		},
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/stores",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusMethodNotAllowed,
			recorder.Code,
		)
	}
}

// TestCreateStoreHandler_InvalidMultipartRequest verifies that malformed
// multipart input is rejected before reaching the application layer.
func TestCreateStoreHandler_InvalidMultipartRequest(t *testing.T) {
	handler := handlers.NewCreateStoreHandler(
		&mockCreateStoreService{
			executeFunc: func(
				ctx context.Context,
				input dto.CreateStoreInput,
			) (dto.CreateStoreOutput, error) {
				t.Fatal("application service must not be called for malformed multipart input")
				return dto.CreateStoreOutput{}, nil
			},
		},
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/stores",
		strings.NewReader("this is not valid multipart data"),
	)

	request.Header.Set(
		"Content-Type",
		"multipart/form-data; boundary=invalid-boundary",
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusBadRequest,
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

// assertCreateStoreInput verifies the translation performed by the handler
// without coupling the test to HTTP implementation details.
func assertCreateStoreInput(
	t *testing.T,
	ctx context.Context,
	input dto.CreateStoreInput,
	ownerID uuid.UUID,
	imageContent []byte,
) {
	t.Helper()

	if input.Name != "My Store" {
		t.Errorf("expected name %q, got %q", "My Store", input.Name)
	}

	if input.Slug != "my-store" {
		t.Errorf("expected slug %q, got %q", "my-store", input.Slug)
	}

	if input.Description != "My store description" {
		t.Errorf(
			"expected description %q, got %q",
			"My store description",
			input.Description,
		)
	}

	if input.Image == nil {
		t.Fatal("expected image input")
	}

	if string(input.Image.Content) != string(imageContent) {
		t.Errorf(
			"expected image content %q, got %q",
			string(imageContent),
			string(input.Image.Content),
		)
	}

	if input.Image.Filename != "store.png" {
		t.Errorf(
			"expected filename %q, got %q",
			"store.png",
			input.Image.Filename,
		)
	}

	if input.Image.ContentType != "image/png" {
		t.Errorf(
			"expected content type %q, got %q",
			"image/png",
			input.Image.ContentType,
		)
	}

	identity, ok := ports.AuthenticatedIdentityFromContext(ctx)
	if !ok {
		t.Fatal("expected authenticated identity in request context")
	}

	if identity.UserID != ownerID {
		t.Errorf(
			"expected authenticated user ID %q, got %q",
			ownerID,
			identity.UserID,
		)
	}

	if identity.Role != "user" {
		t.Errorf(
			"expected authenticated role %q, got %q",
			"user",
			identity.Role,
		)
	}
}

// newCreateStoreMultipartRequest creates a valid multipart Store request.
// Supplying an empty filename creates a request without an image.
func newCreateStoreMultipartRequest(
	t *testing.T,
	filename string,
	contentType string,
	imageContent []byte,
) *http.Request {
	t.Helper()

	var body strings.Builder
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("name", "My Store"); err != nil {
		t.Fatalf("failed to write name field: %v", err)
	}

	if err := writer.WriteField("slug", "my-store"); err != nil {
		t.Fatalf("failed to write slug field: %v", err)
	}

	if err := writer.WriteField("description", "My store description"); err != nil {
		t.Fatalf("failed to write description field: %v", err)
	}

	if filename != "" {
		header := make(textproto.MIMEHeader)
		header.Set(
			"Content-Disposition",
			`form-data; name="image"; filename="`+filename+`"`,
		)
		header.Set("Content-Type", contentType)

		part, err := writer.CreatePart(header)
		if err != nil {
			t.Fatalf("failed to create multipart image part: %v", err)
		}

		if _, err := part.Write(imageContent); err != nil {
			t.Fatalf("failed to write image content: %v", err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/stores",
		strings.NewReader(body.String()),
	)

	request.Header.Set("Content-Type", writer.FormDataContentType())

	return request
}

func mustStoreID(value uuid.UUID) valueobjects.StoreID {
	result, err := valueobjects.NewStoreID(value)
	if err != nil {
		panic(err)
	}
	return result
}

func mustOwnerID(value uuid.UUID) valueobjects.OwnerID {
	result, err := valueobjects.NewOwnerID(value)
	if err != nil {
		panic(err)
	}
	return result
}

func mustStoreName(value string) valueobjects.StoreName {
	result, err := valueobjects.NewStoreName(value)
	if err != nil {
		panic(err)
	}
	return result
}

func mustSlug(value string) valueobjects.Slug {
	result, err := valueobjects.NewSlug(value)
	if err != nil {
		panic(err)
	}
	return result
}

func mustDescription(value string) valueobjects.Description {
	result, err := valueobjects.NewDescription(value)
	if err != nil {
		panic(err)
	}
	return result
}

func mustStatus(value valueobjects.Status) valueobjects.Status {
	result, err := valueobjects.NewStatus(value)
	if err != nil {
		panic(err)
	}
	return result
}

func mustPlan(value valueobjects.PlanType) valueobjects.Plan {
	result, err := valueobjects.NewPlan(value)
	if err != nil {
		panic(err)
	}
	return result
}

// Keep the standard errors package imported while allowing future wrapped
// application-error assertions to use errors.Is in this test package.
var _ = errors.Is
