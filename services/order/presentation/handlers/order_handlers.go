package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/use_cases"
	presentationerrors "github.com/iheanyi-dev/ecommerce-backend/services/order/presentation/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/presentation/schemas"
)

const maxRequestBodyBytes = 1 << 20

type OrderHandlers struct {
	create         *use_cases.CreateOrder
	get            *use_cases.GetOrder
	list           *use_cases.ListOrders
	addShipping    *use_cases.AddShipping
	rejectShipping *use_cases.RejectShipping
	markPaid       *use_cases.MarkPaid
	cancel         *use_cases.CancelOrder
	markSent       *use_cases.MarkSent
	confirmReceipt *use_cases.ConfirmReceipt
}

func NewOrderHandlers(
	create *use_cases.CreateOrder,
	get *use_cases.GetOrder,
	list *use_cases.ListOrders,
	addShipping *use_cases.AddShipping,
	rejectShipping *use_cases.RejectShipping,
	markPaid *use_cases.MarkPaid,
	cancel *use_cases.CancelOrder,
	markSent *use_cases.MarkSent,
	confirmReceipt *use_cases.ConfirmReceipt,
) *OrderHandlers {
	return &OrderHandlers{
		create: create, get: get, list: list,
		addShipping: addShipping, rejectShipping: rejectShipping,
		markPaid: markPaid, cancel: cancel, markSent: markSent,
		confirmReceipt: confirmReceipt,
	}
}

func (h *OrderHandlers) CreateOrder(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	identity, ok := authenticatedIdentity(r)
	if !ok {
		presentationerrors.WriteError(w, presentationerrors.ErrAuthenticationRequired)
		return
	}

	var request schemas.CreateOrderRequest
	if !decodeJSON(w, r, &request) || !request.Validate() {
		presentationerrors.WriteError(w, presentationerrors.ErrInvalidRequestBody)
		return
	}

	orders, err := h.create.Execute(r.Context(), use_cases.CreateOrderInput{
		CartID: request.CartID, CustomerID: identity.UserID,
		DeliveryLocation: strings.TrimSpace(request.DeliveryLocation),
	})
	if err != nil {
		presentationerrors.WriteError(w, err)
		return
	}

	response := make([]schemas.OrderResponse, 0, len(orders))
	for _, order := range orders {
		response = append(response, schemas.FromDetails(&use_cases.OrderDetails{
			Order: order,
		}))
	}
	writeJSON(w, http.StatusCreated, map[string]any{"items": response})
}

func (h *OrderHandlers) GetOrder(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	identity, ok := authenticatedIdentity(r)
	if !ok {
		presentationerrors.WriteError(w, presentationerrors.ErrAuthenticationRequired)
		return
	}
	orderID, ok := pathUUID(r, "order_id", "id")
	if !ok {
		presentationerrors.WriteError(w, presentationerrors.ErrInvalidRequestBody)
		return
	}

	storeID, err := optionalStoreID(r)
	if err != nil {
		presentationerrors.WriteError(w, presentationerrors.ErrInvalidRequestBody)
		return
	}
	details, err := h.get.Execute(r.Context(), use_cases.GetOrderInput{
		OrderID: orderID, UserID: identity.UserID, StoreID: storeID,
	})
	if err != nil {
		presentationerrors.WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, schemas.FromDetails(details))
}

func (h *OrderHandlers) ListOrders(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	identity, ok := authenticatedIdentity(r)
	if !ok {
		presentationerrors.WriteError(w, presentationerrors.ErrAuthenticationRequired)
		return
	}

	page, err := queryInt(r, "page", 1)
	if err != nil {
		presentationerrors.WriteError(w, presentationerrors.ErrInvalidRequestBody)
		return
	}
	pageSize, err := queryInt(r, "page_size", use_cases.DefaultOrderPageSize)
	if err != nil {
		presentationerrors.WriteError(w, presentationerrors.ErrInvalidRequestBody)
		return
	}
	storeID, err := optionalStoreID(r)
	if err != nil {
		presentationerrors.WriteError(w, presentationerrors.ErrInvalidRequestBody)
		return
	}

	result, err := h.list.Execute(r.Context(), use_cases.ListOrdersInput{
		UserID: identity.UserID, StoreID: storeID, Page: page, PageSize: pageSize,
	})
	if err != nil {
		presentationerrors.WriteError(w, err)
		return
	}

	items := make([]schemas.OrderResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, schemas.FromDetails(item))
	}
	writeJSON(w, http.StatusOK, schemas.OrderPageResponse{
		Items: items, Page: result.Page, PageSize: result.PageSize,
		TotalItems: result.TotalItems, TotalPages: result.TotalPages,
	})
}

func (h *OrderHandlers) AddShipping(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	identity, ok := authenticatedIdentity(r)
	if !ok {
		presentationerrors.WriteError(w, presentationerrors.ErrAuthenticationRequired)
		return
	}
	orderID, ok := pathUUID(r, "order_id", "id")
	if !ok {
		presentationerrors.WriteError(w, presentationerrors.ErrInvalidRequestBody)
		return
	}
	storeID, err := requiredStoreID(r)
	if err != nil {
		presentationerrors.WriteError(w, presentationerrors.ErrInvalidRequestBody)
		return
	}
	var request schemas.AddShippingRequest
	if !decodeJSON(w, r, &request) || !request.Validate() {
		presentationerrors.WriteError(w, presentationerrors.ErrInvalidRequestBody)
		return
	}
	_ = identity
	if err := h.addShipping.Execute(r.Context(), use_cases.AddShippingInput{
		OrderID: orderID, StoreID: storeID, ShippingAmount: request.ShippingAmount,
	}); err != nil {
		presentationerrors.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *OrderHandlers) RejectShipping(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	identity, ok := authenticatedIdentity(r)
	if !ok {
		presentationerrors.WriteError(w, presentationerrors.ErrAuthenticationRequired)
		return
	}
	orderID, ok := pathUUID(r, "order_id", "id")
	if !ok {
		presentationerrors.WriteError(w, presentationerrors.ErrInvalidRequestBody)
		return
	}
	if err := h.rejectShipping.Execute(r.Context(), use_cases.RejectShippingInput{
		OrderID: orderID, CustomerID: identity.UserID,
	}); err != nil {
		presentationerrors.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// MarkPaid is intended for a trusted, service-authenticated payment callback.
func (h *OrderHandlers) MarkPaid(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	orderID, ok := pathUUID(r, "order_id", "id")
	if !ok {
		presentationerrors.WriteError(w, presentationerrors.ErrInvalidRequestBody)
		return
	}
	if err := h.markPaid.Execute(r.Context(), use_cases.MarkPaidInput{OrderID: orderID}); err != nil {
		presentationerrors.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *OrderHandlers) CancelOrder(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	identity, ok := authenticatedIdentity(r)
	if !ok {
		presentationerrors.WriteError(w, presentationerrors.ErrAuthenticationRequired)
		return
	}
	orderID, ok := pathUUID(r, "order_id", "id")
	if !ok {
		presentationerrors.WriteError(w, presentationerrors.ErrInvalidRequestBody)
		return
	}
	if err := h.cancel.Execute(r.Context(), use_cases.CancelOrderInput{
		OrderID: orderID, ActorID: identity.UserID,
	}); err != nil {
		presentationerrors.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *OrderHandlers) MarkSent(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if _, ok := authenticatedIdentity(r); !ok {
		presentationerrors.WriteError(w, presentationerrors.ErrAuthenticationRequired)
		return
	}
	orderID, ok := pathUUID(r, "order_id", "id")
	if !ok {
		presentationerrors.WriteError(w, presentationerrors.ErrInvalidRequestBody)
		return
	}
	storeID, err := requiredStoreID(r)
	if err != nil {
		presentationerrors.WriteError(w, presentationerrors.ErrInvalidRequestBody)
		return
	}
	if err := h.markSent.Execute(r.Context(), use_cases.MarkSentInput{
		OrderID: orderID, StoreID: storeID,
	}); err != nil {
		presentationerrors.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *OrderHandlers) ConfirmReceipt(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	identity, ok := authenticatedIdentity(r)
	if !ok {
		presentationerrors.WriteError(w, presentationerrors.ErrAuthenticationRequired)
		return
	}
	orderID, ok := pathUUID(r, "order_id", "id")
	if !ok {
		presentationerrors.WriteError(w, presentationerrors.ErrInvalidRequestBody)
		return
	}
	if err := h.confirmReceipt.Execute(r.Context(), use_cases.ConfirmReceiptInput{
		OrderID: orderID, CustomerID: identity.UserID,
	}); err != nil {
		presentationerrors.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func authenticatedIdentity(r *http.Request) (ports.AuthenticatedIdentity, bool) {
	identity, ok := ports.AuthenticatedIdentityFromContext(r.Context())
	return identity, ok && identity.UserID != uuid.Nil && strings.TrimSpace(identity.Role) != ""
}

func pathUUID(r *http.Request, names ...string) (uuid.UUID, bool) {
	for _, name := range names {
		value := strings.TrimSpace(r.PathValue(name))
		if value == "" {
			continue
		}
		id, err := uuid.Parse(value)
		return id, err == nil && id != uuid.Nil
	}
	return uuid.Nil, false
}

func optionalStoreID(r *http.Request) (uuid.UUID, error) {
	value := strings.TrimSpace(r.PathValue("store_id"))
	if value == "" {
		value = strings.TrimSpace(r.URL.Query().Get("store_id"))
	}
	if value == "" {
		return uuid.Nil, nil
	}
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, errors.New("invalid store ID")
	}
	return id, nil
}

func requiredStoreID(r *http.Request) (uuid.UUID, error) {
	id, err := optionalStoreID(r)
	if err != nil {
		return uuid.Nil, err
	}
	if id == uuid.Nil {
		return uuid.Nil, errors.New("store ID is required")
	}
	return id, nil
}

func queryInt(r *http.Request, key string, defaultValue int) (int, error) {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 0, errors.New("invalid " + key)
	}
	return parsed, nil
}

func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}
	presentationerrors.WriteError(w, presentationerrors.ErrMethodNotAllowed)
	return false
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) bool {
	if r.Body == nil {
		return false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
