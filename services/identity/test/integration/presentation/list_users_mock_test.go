package presentation_test

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
)

// mockRouterListUsersService is a lightweight test double used by
// integration-test router helpers.
//
// The integration tests for registration, login, refresh, and logout
// are not testing the List Users use case itself. However, the production
// router now requires a ListUsersHandler dependency, so those tests need
// a valid application service behind that handler.
//
// This mock keeps those unrelated integration tests isolated from the
// List Users application and database logic.
type mockRouterListUsersService struct {
	result *dto.ListUsersResult
	err    error
}

// Execute implements ports.ListUsersService.
//
// The method intentionally returns the configured result and error without
// performing any business logic because the surrounding integration tests
// are concerned with their respective authentication flows.
func (m *mockRouterListUsersService) Execute(
	ctx context.Context,
	limit int,
	offset int,
) (*dto.ListUsersResult, error) {
	return m.result, m.err
}

// Compile-time verification that the mock satisfies the application port.
var _ ports.ListUsersService = (*mockRouterListUsersService)(nil)

// mockIntegrationUpdateUserStatusService is a lightweight test double used
// by integration-test router helpers.
//
// The integration tests for registration, login, refresh, and logout are not
// testing administrative status management. They only need a valid
// UpdateUserStatusHandler dependency so the production router can be built.
type mockIntegrationUpdateUserStatusService struct {
	result dto.UpdateUserStatusResult
	err    error
}

// Execute implements ports.UpdateUserStatusService.
func (m *mockIntegrationUpdateUserStatusService) Execute(
	ctx context.Context,
	userID string,
	command dto.UpdateUserStatusCommand,
) (dto.UpdateUserStatusResult, error) {
	return m.result, m.err
}

// Compile-time verification that the mock satisfies the application port.
var _ ports.UpdateUserStatusService = (*mockIntegrationUpdateUserStatusService)(nil)
