package use_cases_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/ports"
	usecases "github.com/iheanyi-dev/ecommerce-backend/services/storage/application/use_cases"
)

func TestGetPublicObjectUseCase_ReturnsObject(t *testing.T) {
	storage := &getPublicObjectStorage{
		content:     []byte("public-object"),
		contentType: "image/png",
	}
	logger := &getPublicObjectLogger{}
	metrics := &getPublicObjectMetrics{}

	uc := usecases.NewGetPublicObjectUseCase(storage, logger, metrics)

	output, err := uc.Execute(
		context.Background(),
		dto.GetPublicObjectInput{
			Reference: "stores/store-id.png",
		},
	)

	require.NoError(t, err)
	require.NotNil(t, output.Content)
	require.Equal(t, "image/png", output.ContentType)
	require.Equal(t, "stores/store-id.png", storage.reference)

	content, readErr := io.ReadAll(output.Content)
	require.NoError(t, readErr)
	require.NoError(t, output.Content.Close())
	require.Equal(t, []byte("public-object"), content)

	require.True(t, logger.successCalled)
	require.GreaterOrEqual(t, metrics.incrementCalls, 1)
	require.GreaterOrEqual(t, metrics.observeCalls, 1)
}

func TestGetPublicObjectUseCase_StorageFailure(t *testing.T) {
	storageError := errors.New("storage backend unavailable")
	storage := &getPublicObjectStorage{
		err: storageError,
	}
	logger := &getPublicObjectLogger{}
	metrics := &getPublicObjectMetrics{}

	uc := usecases.NewGetPublicObjectUseCase(storage, logger, metrics)

	output, err := uc.Execute(
		context.Background(),
		dto.GetPublicObjectInput{
			Reference: "stores/store-id.png",
		},
	)

	require.Error(t, err)
	require.ErrorIs(t, err, storageError)
	require.Nil(t, output.Content)
	require.False(t, logger.successCalled)
	require.GreaterOrEqual(t, metrics.incrementCalls, 1)
	require.GreaterOrEqual(t, metrics.observeCalls, 1)
}

func TestGetPublicObjectUseCase_ObservabilityFailureDoesNotChangeSuccessfulResult(t *testing.T) {
	storage := &getPublicObjectStorage{
		content:     []byte("public-object"),
		contentType: "image/webp",
	}
	logger := &getPublicObjectLogger{
		err: errors.New("logger unavailable"),
	}
	metrics := &getPublicObjectMetrics{
		err: errors.New("metrics unavailable"),
	}

	uc := usecases.NewGetPublicObjectUseCase(storage, logger, metrics)

	output, err := uc.Execute(
		context.Background(),
		dto.GetPublicObjectInput{
			Reference: "products/product-id.webp",
		},
	)

	require.NoError(t, err)
	require.NotNil(t, output.Content)

	content, readErr := io.ReadAll(output.Content)
	require.NoError(t, readErr)
	require.NoError(t, output.Content.Close())
	require.Equal(t, []byte("public-object"), content)
}

type getPublicObjectStorage struct {
	reference   string
	content     []byte
	contentType string
	err         error
}

func (f *getPublicObjectStorage) PutObject(
	_ context.Context,
	_ string,
	_ io.Reader,
	_ string,
) error {
	return errors.New("unexpected PutObject call")
}

func (f *getPublicObjectStorage) DeleteObject(
	_ context.Context,
	_ string,
) error {
	return errors.New("unexpected DeleteObject call")
}

func (f *getPublicObjectStorage) GetPrivateObject(
	_ context.Context,
	_ string,
) (io.ReadCloser, string, error) {
	return nil, "", errors.New("unexpected GetPrivateObject call")
}

func (f *getPublicObjectStorage) GetPublicObject(
	_ context.Context,
	reference string,
) (io.ReadCloser, string, error) {
	f.reference = reference

	if f.err != nil {
		return nil, "", f.err
	}

	return io.NopCloser(bytes.NewReader(f.content)), f.contentType, nil
}

type getPublicObjectLogger struct {
	successCalled bool
	err           error
}

func (f *getPublicObjectLogger) Log(
	_ context.Context,
	event ports.LogEvent,
) error {
	if event.Event == "storage.get_public_object.succeeded" {
		f.successCalled = true
	}

	return f.err
}

type getPublicObjectMetrics struct {
	incrementCalls int
	observeCalls   int
	err            error
}

func (f *getPublicObjectMetrics) Increment(
	_ context.Context,
	_ ports.Metric,
) error {
	f.incrementCalls++
	return f.err
}

func (f *getPublicObjectMetrics) Observe(
	_ context.Context,
	_ ports.Metric,
) error {
	f.observeCalls++
	return f.err
}
