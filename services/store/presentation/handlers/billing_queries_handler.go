package handlers

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/schemas"
)

type ListBillingPlansHandler struct {
	service ports.ListBillingPlansService
}

func NewListBillingPlansHandler(service ports.ListBillingPlansService) *ListBillingPlansHandler {
	return &ListBillingPlansHandler{service: service}
}

func (h *ListBillingPlansHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		presentation_errors.WriteError(w, presentation_errors.ErrMethodNotAllowed)
		return
	}

	result, err := h.service.Execute(r.Context())
	if err != nil {
		presentation_errors.WriteError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, schemas.NewListBillingPlansResponse(result))
}

type GetBillingPlanHandler struct {
	service ports.GetBillingPlanService
}

func NewGetBillingPlanHandler(service ports.GetBillingPlanService) *GetBillingPlanHandler {
	return &GetBillingPlanHandler{service: service}
}

func (h *GetBillingPlanHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		presentation_errors.WriteError(w, presentation_errors.ErrMethodNotAllowed)
		return
	}

	planID, err := uuid.Parse(r.PathValue("id"))
	if err != nil || planID == uuid.Nil {
		presentation_errors.WriteError(w, presentation_errors.ErrInvalidRequestBody)
		return
	}

	result, err := h.service.Execute(r.Context(), dto.GetBillingPlanInput{PlanID: planID})
	if err != nil {
		presentation_errors.WriteError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, schemas.NewBillingPlanResponse(result))
}

type ListStoreSubscriptionsHandler struct {
	service ports.ListStoreSubscriptionsService
}

func NewListStoreSubscriptionsHandler(
	service ports.ListStoreSubscriptionsService,
) *ListStoreSubscriptionsHandler {
	return &ListStoreSubscriptionsHandler{service: service}
}

func (h *ListStoreSubscriptionsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		presentation_errors.WriteError(w, presentation_errors.ErrMethodNotAllowed)
		return
	}

	storeID, err := uuid.Parse(r.PathValue("id"))
	if err != nil || storeID == uuid.Nil {
		presentation_errors.WriteError(w, presentation_errors.ErrInvalidRequestBody)
		return
	}

	result, err := h.service.Execute(
		r.Context(),
		dto.ListStoreSubscriptionsInput{StoreID: storeID},
	)
	if err != nil {
		presentation_errors.WriteError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, schemas.NewListStoreSubscriptionsResponse(result))
}

type GetStoreSubscriptionHandler struct {
	service ports.GetStoreSubscriptionService
}

func NewGetStoreSubscriptionHandler(
	service ports.GetStoreSubscriptionService,
) *GetStoreSubscriptionHandler {
	return &GetStoreSubscriptionHandler{service: service}
}

func (h *GetStoreSubscriptionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		presentation_errors.WriteError(w, presentation_errors.ErrMethodNotAllowed)
		return
	}

	storeID, storeErr := uuid.Parse(r.PathValue("id"))
	subscriptionID, subscriptionErr := uuid.Parse(r.PathValue("subscriptionID"))
	if storeErr != nil || storeID == uuid.Nil ||
		subscriptionErr != nil || subscriptionID == uuid.Nil {
		presentation_errors.WriteError(w, presentation_errors.ErrInvalidRequestBody)
		return
	}

	result, err := h.service.Execute(
		r.Context(),
		dto.GetStoreSubscriptionInput{
			StoreID:        storeID,
			SubscriptionID: subscriptionID,
		},
	)
	if err != nil {
		presentation_errors.WriteError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, schemas.NewBillingSubscriptionResponse(result))
}
