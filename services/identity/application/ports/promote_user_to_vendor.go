package ports

import "context"

// PromoteUserToVendorService promotes an existing user account to vendor.
type PromoteUserToVendorService interface {
	Execute(ctx context.Context, userID string) error
}
