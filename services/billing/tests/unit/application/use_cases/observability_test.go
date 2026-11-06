package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain"
)

func assertSuccessfulTelemetry(
	t *testing.T,
	logger *recordingLogger,
	metrics *recordingMetrics,
	operation string,
	metricPrefix string,
) {
	t.Helper()

	if len(logger.events) != 1 {
		t.Fatalf("expected 1 log event, got %d", len(logger.events))
	}

	if got, want := logger.events[0].Event, "billing."+operation+".succeeded"; got != want {
		t.Fatalf("expected event %q, got %q", want, got)
	}

	if logger.events[0].Operation != operation {
		t.Fatalf("expected operation %q, got %q", operation, logger.events[0].Operation)
	}

	if len(metrics.increments) != 2 {
		t.Fatalf("expected 2 counter increments, got %d", len(metrics.increments))
	}

	if metrics.increments[0].Name != metricPrefix+"_total" {
		t.Fatalf("unexpected start metric: %q", metrics.increments[0].Name)
	}

	if metrics.increments[1].Name != metricPrefix+"_success_total" {
		t.Fatalf("unexpected success metric: %q", metrics.increments[1].Name)
	}

	if len(metrics.observations) != 1 {
		t.Fatalf("expected 1 duration observation, got %d", len(metrics.observations))
	}

	if metrics.observations[0].Name != metricPrefix+"_duration_seconds" {
		t.Fatalf("unexpected duration metric: %q", metrics.observations[0].Name)
	}
}

func assertFailureTelemetry(
	t *testing.T,
	logger *recordingLogger,
	metrics *recordingMetrics,
	operation string,
	metricPrefix string,
	category string,
) {
	t.Helper()

	if len(logger.events) != 1 {
		t.Fatalf("expected 1 log event, got %d", len(logger.events))
	}

	if got, want := logger.events[0].Event, "billing."+operation+".failed"; got != want {
		t.Fatalf("expected event %q, got %q", want, got)
	}

	if logger.events[0].FailureCategory != category {
		t.Fatalf("expected failure category %q, got %q", category, logger.events[0].FailureCategory)
	}

	if len(metrics.increments) != 2 {
		t.Fatalf("expected 2 counter increments, got %d", len(metrics.increments))
	}

	if metrics.increments[0].Name != metricPrefix+"_total" {
		t.Fatalf("unexpected start metric: %q", metrics.increments[0].Name)
	}

	failureMetric := metrics.increments[1]
	if failureMetric.Name != metricPrefix+"_failure_total" {
		t.Fatalf("unexpected failure metric: %q", failureMetric.Name)
	}

	if failureMetric.Labels["failure_category"] != category {
		t.Fatalf("expected metric failure category %q, got %q", category, failureMetric.Labels["failure_category"])
	}

	if len(metrics.observations) != 1 {
		t.Fatalf("expected 1 duration observation, got %d", len(metrics.observations))
	}
}

func TestPlanUseCasesRecordSuccessfulTelemetry(t *testing.T) {
	ctx := context.Background()

	t.Run("create plan", func(t *testing.T) {
		repository := &fakePlanRepository{}
		logger, metrics, observability := newRecordingObservability()

		_, err := use_cases.NewCreatePlan(repository, observability).Execute(
			ctx,
			validCreatePlanInput(),
		)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		assertSuccessfulTelemetry(t, logger, metrics, "create_plan", "billing_create_plan")
	})

	t.Run("delete plan", func(t *testing.T) {
		repository := &fakePlanRepository{}
		logger, metrics, observability := newRecordingObservability()

		err := use_cases.NewDeletePlanByID(repository, observability).Execute(ctx, uuid.New())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		assertSuccessfulTelemetry(t, logger, metrics, "delete_plan_by_id", "billing_delete_plan_by_id")
	})

	t.Run("get plan by id", func(t *testing.T) {
		repository := &fakePlanRepository{plan: validPlan()}
		logger, metrics, observability := newRecordingObservability()

		_, err := use_cases.NewGetPlanByID(repository, observability).Execute(ctx, repository.plan.ID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		assertSuccessfulTelemetry(t, logger, metrics, "get_plan_by_id", "billing_get_plan_by_id")
	})

	t.Run("get plans", func(t *testing.T) {
		logger, metrics, observability := newRecordingObservability()

		_, err := use_cases.NewGetPlans(&fakePlanRepository{}, observability).Execute(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		assertSuccessfulTelemetry(t, logger, metrics, "get_plans", "billing_get_plans")
	})

	t.Run("update plan", func(t *testing.T) {
		repository := &fakePlanRepository{plan: validPlan()}
		logger, metrics, observability := newRecordingObservability()

		_, err := use_cases.NewUpdatePlan(repository, observability).Execute(
			ctx,
			repository.plan.ID,
			validUpdatePlanInput(),
		)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		assertSuccessfulTelemetry(t, logger, metrics, "update_plan", "billing_update_plan")
	})
}

func TestPlanUseCasesRecordFailureTelemetry(t *testing.T) {
	ctx := context.Background()

	t.Run("create plan validation failure", func(t *testing.T) {
		logger, metrics, observability := newRecordingObservability()

		_, err := use_cases.NewCreatePlan(&fakePlanRepository{}, observability).Execute(
			ctx,
			use_cases.CreatePlanInput{},
		)
		if err == nil {
			t.Fatal("expected validation error")
		}

		assertFailureTelemetry(t, logger, metrics, "create_plan", "billing_create_plan", "validation")
	})

	t.Run("delete plan invalid input", func(t *testing.T) {
		logger, metrics, observability := newRecordingObservability()

		err := use_cases.NewDeletePlanByID(&fakePlanRepository{}, observability).Execute(ctx, uuid.Nil)
		if !errors.Is(err, domain.ErrInvalidPlan) {
			t.Fatalf("expected invalid plan error, got %v", err)
		}

		assertFailureTelemetry(t, logger, metrics, "delete_plan_by_id", "billing_delete_plan_by_id", "invalid_input")
	})

	t.Run("get plan invalid input", func(t *testing.T) {
		logger, metrics, observability := newRecordingObservability()

		_, err := use_cases.NewGetPlanByID(&fakePlanRepository{}, observability).Execute(ctx, uuid.Nil)
		if !errors.Is(err, domain.ErrInvalidPlan) {
			t.Fatalf("expected invalid plan error, got %v", err)
		}

		assertFailureTelemetry(t, logger, metrics, "get_plan_by_id", "billing_get_plan_by_id", "invalid_input")
	})

	t.Run("get plans repository failure", func(t *testing.T) {
		repositoryError := errors.New("database unavailable")
		repository := &fakePlanRepository{getAllErr: repositoryError}
		logger, metrics, observability := newRecordingObservability()

		_, err := use_cases.NewGetPlans(repository, observability).Execute(ctx)
		if !errors.Is(err, repositoryError) {
			t.Fatalf("expected repository error, got %v", err)
		}

		assertFailureTelemetry(t, logger, metrics, "get_plans", "billing_get_plans", "retrieval")
	})

	t.Run("update plan invalid input", func(t *testing.T) {
		logger, metrics, observability := newRecordingObservability()

		_, err := use_cases.NewUpdatePlan(&fakePlanRepository{}, observability).Execute(
			ctx,
			uuid.Nil,
			validUpdatePlanInput(),
		)
		if !errors.Is(err, domain.ErrInvalidPlan) {
			t.Fatalf("expected invalid plan error, got %v", err)
		}

		assertFailureTelemetry(t, logger, metrics, "update_plan", "billing_update_plan", "invalid_input")
	})
}

func TestPlanUseCasesIgnoreObservabilityFailures(t *testing.T) {
	ctx := context.Background()
	observability := newFailingObservability()

	t.Run("create plan", func(t *testing.T) {
		_, err := use_cases.NewCreatePlan(&fakePlanRepository{}, observability).Execute(
			ctx,
			validCreatePlanInput(),
		)
		if err != nil {
			t.Fatalf("observability failure changed successful result: %v", err)
		}
	})

	t.Run("delete plan", func(t *testing.T) {
		err := use_cases.NewDeletePlanByID(&fakePlanRepository{}, observability).Execute(ctx, uuid.New())
		if err != nil {
			t.Fatalf("observability failure changed successful result: %v", err)
		}
	})

	t.Run("get plan by id", func(t *testing.T) {
		repository := &fakePlanRepository{plan: validPlan()}
		_, err := use_cases.NewGetPlanByID(repository, observability).Execute(ctx, repository.plan.ID)
		if err != nil {
			t.Fatalf("observability failure changed successful result: %v", err)
		}
	})

	t.Run("get plans", func(t *testing.T) {
		_, err := use_cases.NewGetPlans(&fakePlanRepository{}, observability).Execute(ctx)
		if err != nil {
			t.Fatalf("observability failure changed successful result: %v", err)
		}
	})

	t.Run("update plan", func(t *testing.T) {
		repository := &fakePlanRepository{plan: validPlan()}
		_, err := use_cases.NewUpdatePlan(repository, observability).Execute(
			ctx,
			repository.plan.ID,
			validUpdatePlanInput(),
		)
		if err != nil {
			t.Fatalf("observability failure changed successful result: %v", err)
		}
	})
}

// Compile-time checks for the observability test doubles.
var (
	_ ports.Logger  = (*recordingLogger)(nil)
	_ ports.Metrics = (*recordingMetrics)(nil)
)
