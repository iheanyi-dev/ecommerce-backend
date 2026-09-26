package use_cases_test

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/ports"
	usecases "github.com/iheanyi-dev/ecommerce-backend/services/storage/application/use_cases"
)

func TestDeleteObjectUseCase_DeletesObject(t *testing.T) {
	storage := &deleteObjectStorage{}
	logger := &deleteObjectLogger{}
	metrics := &deleteObjectMetrics{}

	uc := usecases.NewDeleteObjectUseCase(storage, logger, metrics)

	err := uc.Execute(
		context.Background(),
		dto.DeleteObjectInput{
			Reference: "stores/store-id.png",
		},
	)

	require.NoError(t, err)
	require.True(t, storage.called)
	require.Equal(t, "stores/store-id.png", storage.reference)
	require.True(t, logger.successCalled)
	require.GreaterOrEqual(t, metrics.incrementCalls, 1)
	require.GreaterOrEqual(t, metrics.observeCalls, 1)
}

func TestDeleteObjectUseCase_StorageFailure(t *testing.T) {
	storageError := errors.New("storage backend unavailable")
	storage := &deleteObjectStorage{
		err: storageError,
	}
	logger := &deleteObjectLogger{}
	metrics := &deleteObjectMetrics{}

	uc := usecases.NewDeleteObjectUseCase(storage, logger, metrics)

	err := uc.Execute(
		context.Background(),
		dto.DeleteObjectInput{
			Reference: "knowledge/store-id/file.pdf",
		},
	)

	require.Error(t, err)
	require.ErrorIs(t, err, storageError)
	require.True(t, storage.called)
	require.False(t, logger.successCalled)
	require.GreaterOrEqual(t, metrics.incrementCalls, 1)
	require.GreaterOrEqual(t, metrics.observeCalls, 1)
}

func TestDeleteObjectUseCase_ObservabilityFailureDoesNotChangeSuccessfulResult(t *testing.T) {
	storage := &deleteObjectStorage{}
	logger := &deleteObjectLogger{
		err: errors.New("logger unavailable"),
	}
	metrics := &deleteObjectMetrics{
		err: errors.New("metrics unavailable"),
	}

	uc := usecases.NewDeleteObjectUseCase(storage, logger, metrics)

	err := uc.Execute(
		context.Background(),
		dto.DeleteObjectInput{
			Reference: "products/product-id.webp",
		},
	)

	require.NoError(t, err)
	require.True(t, storage.called)
}

type deleteObjectStorage struct {
	called    bool
	reference string
	err       error
}

func (f *deleteObjectStorage) PutObject(
	_ context.Context,
	_ string,
	_ io.Reader,
	_ string,
) error {
	return errors.New("unexpected PutObject call")
}

func (f *deleteObjectStorage) GetPublicObject(
	_ context.Context,
	_ string,
) (io.ReadCloser, string, error) {
	return nil, "", errors.New("unexpected GetPublicObject call")
}

func (f *deleteObjectStorage) GetPrivateObject(
	_ context.Context,
	_ string,
) (io.ReadCloser, string, error) {
	return nil, "", errors.New("unexpected GetPrivateObject call")
}

func (f *deleteObjectStorage) DeleteObject(
	_ context.Context,
	reference string,
) error {
	f.called = true
	f.reference = reference

	return f.err
}

type deleteObjectLogger struct {
	successCalled bool
	err           error
}

func (f *deleteObjectLogger) Log(
	_ context.Context,
	event ports.LogEvent,
) error {
	if event.Event == "storage.delete_object.succeeded" {
		f.successCalled = true
	}

	return f.err
}

type deleteObjectMetrics struct {
	incrementCalls int
	observeCalls   int
	err            error
}

func (f *deleteObjectMetrics) Increment(
	_ context.Context,
	_ ports.Metric,
) error {
	f.incrementCalls++
	return f.err
}

func (f *deleteObjectMetrics) Observe(
	_ context.Context,
	_ ports.Metric,
) error {
	f.observeCalls++
	return f.err
}
