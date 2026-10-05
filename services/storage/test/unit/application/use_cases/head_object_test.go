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

func TestHeadObjectUseCase_ReturnsContentType(t *testing.T) {
	storage := &headObjectStorage{
		contentType: "image/png",
	}
	logger := &headObjectLogger{}
	metrics := &headObjectMetrics{}

	uc := usecases.NewHeadObjectUseCase(storage, logger, metrics)

	output, err := uc.Execute(
		context.Background(),
		dto.HeadObjectInput{
			Reference: "stores/store-id.png",
		},
	)

	require.NoError(t, err)
	require.Equal(t, "image/png", output.ContentType)
	require.True(t, storage.called)
	require.Equal(t, "stores/store-id.png", storage.reference)
	require.True(t, logger.successCalled)
	require.GreaterOrEqual(t, metrics.incrementCalls, 1)
	require.GreaterOrEqual(t, metrics.observeCalls, 1)
}

func TestHeadObjectUseCase_StorageFailure(t *testing.T) {
	storageError := errors.New("storage backend unavailable")
	storage := &headObjectStorage{
		err: storageError,
	}
	logger := &headObjectLogger{}
	metrics := &headObjectMetrics{}

	uc := usecases.NewHeadObjectUseCase(storage, logger, metrics)

	_, err := uc.Execute(
		context.Background(),
		dto.HeadObjectInput{
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

func TestHeadObjectUseCase_ObservabilityFailureDoesNotChangeSuccessfulResult(t *testing.T) {
	storage := &headObjectStorage{
		contentType: "application/pdf",
	}
	logger := &headObjectLogger{
		err: errors.New("logger unavailable"),
	}
	metrics := &headObjectMetrics{
		err: errors.New("metrics unavailable"),
	}

	uc := usecases.NewHeadObjectUseCase(storage, logger, metrics)

	output, err := uc.Execute(
		context.Background(),
		dto.HeadObjectInput{
			Reference: "knowledge/store-id/file.pdf",
		},
	)

	require.NoError(t, err)
	require.Equal(t, "application/pdf", output.ContentType)
	require.True(t, storage.called)
}

type headObjectStorage struct {
	called      bool
	reference   string
	contentType string
	err         error
}

func (f *headObjectStorage) PutObject(
	_ context.Context,
	_ string,
	_ io.Reader,
	_ string,
) error {
	return errors.New("unexpected PutObject call")
}

func (f *headObjectStorage) GetPublicObject(
	_ context.Context,
	_ string,
) (io.ReadCloser, string, error) {
	return nil, "", errors.New("unexpected GetPublicObject call")
}

func (f *headObjectStorage) GetPrivateObject(
	_ context.Context,
	_ string,
) (io.ReadCloser, string, error) {
	return nil, "", errors.New("unexpected GetPrivateObject call")
}

func (f *headObjectStorage) HeadObject(
	_ context.Context,
	reference string,
) (string, error) {
	f.called = true
	f.reference = reference

	return f.contentType, f.err
}

func (f *headObjectStorage) DeleteObject(
	_ context.Context,
	_ string,
) error {
	return errors.New("unexpected DeleteObject call")
}

type headObjectLogger struct {
	successCalled bool
	err           error
}

func (f *headObjectLogger) Log(
	_ context.Context,
	event ports.LogEvent,
) error {
	if event.Event == "storage.head_object.succeeded" {
		f.successCalled = true
	}

	return f.err
}

type headObjectMetrics struct {
	incrementCalls int
	observeCalls   int
	err            error
}

func (f *headObjectMetrics) Increment(
	_ context.Context,
	_ ports.Metric,
) error {
	f.incrementCalls++
	return f.err
}

func (f *headObjectMetrics) Observe(
	_ context.Context,
	_ ports.Metric,
) error {
	f.observeCalls++
	return f.err
}
