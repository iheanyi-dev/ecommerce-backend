package handlers

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/cart/application/use_cases"
	presentationerrors "github.com/iheanyi-dev/ecommerce-backend/services/cart/presentation/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/cart/presentation/schemas"
)

// NewGetMyCartHandler handles GET /api/v1/cart.
func NewGetMyCartHandler(service *use_cases.GetMyCart) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, presentationerrors.ErrMethodNotAllowed)
			return
		}

		id, err := customerID(r)
		if err != nil {
			writeError(w, err)
			return
		}

		cart, err := service.Execute(r.Context(), id)
		if err != nil {
			writeError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, schemas.NewCartResponse(cart))
	})
}

// NewAddItemToCartHandler handles POST /api/v1/cart/items.
func NewAddItemToCartHandler(service *use_cases.AddItemToCart) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, presentationerrors.ErrMethodNotAllowed)
			return
		}

		customer, err := customerID(r)
		if err != nil {
			writeError(w, err)
			return
		}

		var body schemas.AddItemRequest
		if err := decodeJSON(w, r, &body); err != nil {
			writeError(w, err)
			return
		}

		productID, err := uuid.Parse(body.ProductID)
		if err != nil || productID == uuid.Nil {
			writeError(w, presentationerrors.ErrInvalidRequestBody)
			return
		}

		storeID, err := uuid.Parse(body.StoreID)
		if err != nil || storeID == uuid.Nil {
			writeError(w, presentationerrors.ErrInvalidRequestBody)
			return
		}

		cart, err := service.Execute(r.Context(), use_cases.AddItemToCartInput{
			CustomerID: customer,
			ProductID:  productID,
			StoreID:    storeID,
			Quantity:   body.Quantity,
		})
		if err != nil {
			writeError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, schemas.NewCartResponse(cart))
	})
}

// NewSetCartItemQuantityHandler handles PATCH /api/v1/cart/items/{itemID}.
func NewSetCartItemQuantityHandler(service *use_cases.SetCartItemQuantity) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			writeError(w, presentationerrors.ErrMethodNotAllowed)
			return
		}

		customer, err := customerID(r)
		if err != nil {
			writeError(w, err)
			return
		}

		itemID, err := parsePathUUID(r, "itemID")
		if err != nil {
			writeError(w, err)
			return
		}

		var body schemas.SetItemQuantityRequest
		if err := decodeJSON(w, r, &body); err != nil {
			writeError(w, err)
			return
		}

		cart, err := service.Execute(r.Context(), use_cases.SetCartItemQuantityInput{
			CustomerID: customer,
			ItemID:     itemID,
			Quantity:   body.Quantity,
		})
		if err != nil {
			writeError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, schemas.NewCartResponse(cart))
	})
}

// NewRemoveCartItemHandler handles DELETE /api/v1/cart/items/{itemID}.
func NewRemoveCartItemHandler(service *use_cases.RemoveCartItem) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			writeError(w, presentationerrors.ErrMethodNotAllowed)
			return
		}

		customer, err := customerID(r)
		if err != nil {
			writeError(w, err)
			return
		}

		itemID, err := parsePathUUID(r, "itemID")
		if err != nil {
			writeError(w, err)
			return
		}

		if err := service.Execute(r.Context(), use_cases.RemoveCartItemInput{
			CustomerID: customer,
			ItemID:     itemID,
		}); err != nil {
			writeError(w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})
}

// NewClearCartHandler handles DELETE /api/v1/cart.
func NewClearCartHandler(service *use_cases.ClearCart) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			writeError(w, presentationerrors.ErrMethodNotAllowed)
			return
		}

		customer, err := customerID(r)
		if err != nil {
			writeError(w, err)
			return
		}

		if err := service.Execute(r.Context(), customer); err != nil {
			writeError(w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})
}

// NewCheckoutCartHandler handles POST /api/v1/cart/checkout.
// This endpoint reconciles the cart; it does not create an order.
func NewCheckoutCartHandler(service *use_cases.CheckoutCart) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, presentationerrors.ErrMethodNotAllowed)
			return
		}

		customer, err := customerID(r)
		if err != nil {
			writeError(w, err)
			return
		}

		output, err := service.Execute(r.Context(), customer)
		if err != nil {
			writeError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, schemas.NewCheckoutResponse(output))
	})
}
