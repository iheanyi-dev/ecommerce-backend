package use_cases

import (
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
)

// ListAvailablePlans returns the plans Billing currently offers.
// It does not access Store's database or determine Store capabilities.
type ListAvailablePlans struct {
	catalog ports.PlanCatalog
}

// NewListAvailablePlans constructs the catalogue use case.
func NewListAvailablePlans(catalog ports.PlanCatalog) *ListAvailablePlans {
	return &ListAvailablePlans{catalog: catalog}
}

// Execute returns the currently available commercial plan definitions.
func (useCase *ListAvailablePlans) Execute() ([]entities.Plan, error) {
	if useCase == nil || useCase.catalog == nil {
		return nil, application.ErrPlanCatalogUnavailable
	}

	plans := useCase.catalog.AvailablePlans()
	available := make([]entities.Plan, 0, len(plans))

	for _, plan := range plans {
		if !plan.Available {
			continue
		}

		if strings.TrimSpace(plan.Name) == "" {
			return nil, application.ErrInvalidPlanCatalogueEntry
		}

		if err := plan.Validate(); err != nil {
			return nil, err
		}

		available = append(available, plan)
	}

	return available, nil
}
