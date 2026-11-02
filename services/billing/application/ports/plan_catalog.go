package ports

import "github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"

// PlanCatalog provides the plan definitions offered by Billing.
type PlanCatalog interface {
	AvailablePlans() []entities.Plan
}
