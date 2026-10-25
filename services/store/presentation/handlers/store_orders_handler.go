package handlers

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/errors"
)

type ListStoreOrdersHandler struct {
	service ports.ListStoreOrdersService
}

func NewListStoreOrdersHandler(s ports.ListStoreOrdersService) *ListStoreOrdersHandler {
	return &ListStoreOrdersHandler{service: s}
}

func (h *ListStoreOrdersHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		presentation_errors.WriteError(w, presentation_errors.ErrMethodNotAllowed)
		return
	}
	page, err := queryInt(r, "page", 1)
	if err != nil {
		presentation_errors.WriteError(w, presentation_errors.ErrInvalidRequestBody)
		return
	}
	pageSize, err := queryInt(r, "page_size", 20)
	if err != nil {
		presentation_errors.WriteError(w, presentation_errors.ErrInvalidRequestBody)
		return
	}
	result, err := h.service.Execute(r.Context(), page, pageSize)
	if err != nil {
		presentation_errors.WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

type GetStoreOrderHandler struct {
	service ports.GetStoreOrderService
}

func NewGetStoreOrderHandler(s ports.GetStoreOrderService) *GetStoreOrderHandler {
	return &GetStoreOrderHandler{service: s}
}

func (h *GetStoreOrderHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		presentation_errors.WriteError(w, presentation_errors.ErrMethodNotAllowed)
		return
	}
	id, err := uuid.Parse(r.PathValue("orderID"))
	if err != nil || id == uuid.Nil {
		presentation_errors.WriteError(w, presentation_errors.ErrInvalidRequestBody)
		return
	}
	result, err := h.service.Execute(r.Context(), id)
	if err != nil {
		presentation_errors.WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

type AddShippingHandler struct {
	service ports.AddShippingService
}

func NewAddShippingHandler(s ports.AddShippingService) *AddShippingHandler {
	return &AddShippingHandler{service: s}
}

func (h *AddShippingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		presentation_errors.WriteError(w, presentation_errors.ErrMethodNotAllowed)
		return
	}
	id, err := parseOrderID(r)
	if err != nil {
		presentation_errors.WriteError(w, presentation_errors.ErrInvalidRequestBody)
		return
	}
	var body struct {
		ShippingAmount int64 `json:"shipping_amount"`
	}
	if decodeJSON(r, &body) != nil || body.ShippingAmount < 0 {
		presentation_errors.WriteError(w, presentation_errors.ErrInvalidRequestBody)
		return
	}
	if err := h.service.Execute(r.Context(), id, body.ShippingAmount); err != nil {
		presentation_errors.WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

type MarkOrderSentHandler struct {
	service ports.MarkOrderSentService
}

func NewMarkOrderSentHandler(s ports.MarkOrderSentService) *MarkOrderSentHandler {
	return &MarkOrderSentHandler{service: s}
}

func (h *MarkOrderSentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		presentation_errors.WriteError(w, presentation_errors.ErrMethodNotAllowed)
		return
	}
	id, err := parseOrderID(r)
	if err != nil {
		presentation_errors.WriteError(w, presentation_errors.ErrInvalidRequestBody)
		return
	}
	if err := h.service.Execute(r.Context(), id); err != nil {
		presentation_errors.WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

type CancelStoreOrderHandler struct {
	service ports.CancelStoreOrderService
}

func NewCancelStoreOrderHandler(s ports.CancelStoreOrderService) *CancelStoreOrderHandler {
	return &CancelStoreOrderHandler{service: s}
}

func (h *CancelStoreOrderHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		presentation_errors.WriteError(w, presentation_errors.ErrMethodNotAllowed)
		return
	}
	id, err := parseOrderID(r)
	if err != nil {
		presentation_errors.WriteError(w, presentation_errors.ErrInvalidRequestBody)
		return
	}
	if err := h.service.Execute(r.Context(), id); err != nil {
		presentation_errors.WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func parseOrderID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(r.PathValue("orderID"))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, http.ErrNoCookie
	}
	return id, nil
}

func queryInt(r *http.Request, key string, fallback int) (int, error) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return n, nil
}
