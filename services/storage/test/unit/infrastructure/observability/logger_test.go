package observability_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/infrastructure/observability"
)

func TestNewLogger_RejectsNilWriter(t *testing.T) {
	logger, err := observability.NewLogger(nil)

	require.ErrorIs(t, err, observability.ErrNilWriter)
	require.Nil(t, logger)
}

func TestLogger_LogWritesStructuredJSON(t *testing.T) {
	var output bytes.Buffer

	logger, err := observability.NewLogger(&output)
	require.NoError(t, err)

	err = logger.Log(
		context.Background(),
		ports.LogEvent{
			Event:           "storage.put_object.succeeded",
			Operation:       "put_object",
			FailureCategory: "",
		},
	)

	require.NoError(t, err)

	var entry map[string]any
	require.NoError(t, json.Unmarshal(output.Bytes(), &entry))

	require.Equal(t, "storage.put_object.succeeded", entry["event"])
	require.Equal(t, "put_object", entry["operation"])
}

func TestLogger_LogIncludesFailureCategory(t *testing.T) {
	var output bytes.Buffer

	logger, err := observability.NewLogger(&output)
	require.NoError(t, err)

	err = logger.Log(
		context.Background(),
		ports.LogEvent{
			Event:           "storage.delete_object.failed",
			Operation:       "delete_object",
			FailureCategory: "storage_backend",
		},
	)

	require.NoError(t, err)

	var entry map[string]any
	require.NoError(t, json.Unmarshal(output.Bytes(), &entry))

	require.Equal(t, "storage_backend", entry["failure_category"])
}
