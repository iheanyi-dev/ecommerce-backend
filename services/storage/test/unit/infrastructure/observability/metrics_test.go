package observability_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/infrastructure/observability"
)

func TestMetrics_IncrementRecordsCounter(t *testing.T) {
	metrics := observability.NewMetrics()

	err := metrics.Increment(
		context.Background(),
		ports.Metric{
			Name:  "storage.put_object.succeeded",
			Value: 1,
			Labels: map[string]string{
				"operation": "put_object",
			},
		},
	)

	require.NoError(t, err)
	require.Equal(
		t,
		float64(1),
		metrics.CounterValue(
			"storage.put_object.succeeded",
			map[string]string{
				"operation": "put_object",
			},
		),
	)
}

func TestMetrics_ObserveRecordsMeasurement(t *testing.T) {
	metrics := observability.NewMetrics()

	err := metrics.Observe(
		context.Background(),
		ports.Metric{
			Name:  "storage.put_object.duration",
			Value: 42,
			Labels: map[string]string{
				"operation": "put_object",
			},
		},
	)

	require.NoError(t, err)

	require.Equal(
		t,
		[]float64{42},
		metrics.ObservationValues(
			"storage.put_object.duration",
			map[string]string{
				"operation": "put_object",
			},
		),
	)
}

func TestMetrics_EquivalentLabelMapsProduceSameMetric(t *testing.T) {
	metrics := observability.NewMetrics()

	err := metrics.Increment(
		context.Background(),
		ports.Metric{
			Name: "storage.operation",
			Labels: map[string]string{
				"operation": "put",
				"result":    "success",
			},
		},
	)
	require.NoError(t, err)

	require.Equal(
		t,
		float64(1),
		metrics.CounterValue(
			"storage.operation",
			map[string]string{
				"result":    "success",
				"operation": "put",
			},
		),
	)
}

func TestMetrics_RejectsEmptyMetricName(t *testing.T) {
	metrics := observability.NewMetrics()

	err := metrics.Increment(
		context.Background(),
		ports.Metric{},
	)

	require.ErrorIs(t, err, observability.ErrEmptyMetricName)
}
