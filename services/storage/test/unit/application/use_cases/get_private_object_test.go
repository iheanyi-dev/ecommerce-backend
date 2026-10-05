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

func TestGetPrivateObjectUseCase_ReturnsObject(t *testing.T) {
	storage := &getPrivateObjectStorage{
		content:     []byte("private-object"),
		contentType: "application/pdf",
	}
	logger := &getPrivateObjectLogger{}
	metrics := &getPrivateObjectMetrics{}

	uc := usecases.NewGetPrivateObjectUseCase(storage, logger, metrics)

	output, err := uc.Execute(
		context.Background(),
		dto.GetPrivateObjectInput{
			Reference: "knowledge/store-id/manual.pdf",
		},
	)

	require.NoError(t, err)
	require.NotNil(t, output.Content)
	require.Equal(t, "application/pdf", output.ContentType)
	require.Equal(t, "knowledge/store-id/manual.pdf", storage.reference)

	content, readErr := io.ReadAll(output.Content)
	require.NoError(t, readErr)
	require.NoError(t, output.Content.Close())
	require.Equal(t, []byte("private-object"), content)

	require.True(t, logger.successCalled)
	require.GreaterOrEqual(t, metrics.incrementCalls, 1)
	require.GreaterOrEqual(t, metrics.observeCalls, 1)
}

func TestGetPrivateObjectUseCase_StorageFailure(t *testing.T) {
	storageError := errors.New("storage backend unavailable")
	storage := &getPrivateObjectStorage{
		err: storageError,
	}
	logger := &getPrivateObjectLogger{}
	metrics := &getPrivateObjectMetrics{}

	uc := usecases.NewGetPrivateObjectUseCase(storage, logger, metrics)

	output, err := uc.Execute(
		context.Background(),
		dto.GetPrivateObjectInput{
			Reference: "knowledge/store-id/manual.pdf",
		},
	)

	require.Error(t, err)
	require.ErrorIs(t, err, storageError)
	require.Nil(t, output.Content)
	require.False(t, logger.successCalled)
	require.GreaterOrEqual(t, metrics.incrementCalls, 1)
	require.GreaterOrEqual(t, metrics.observeCalls, 1)
}

func TestGetPrivateObjectUseCase_ObservabilityFailureDoesNotChangeSuccessfulResult(t *testing.T) {
	storage := &getPrivateObjectStorage{
		content:     []byte("private-object"),
		contentType: "application/octet-stream",
	}
	logger := &getPrivateObjectLogger{
		err: errors.New("logger unavailable"),
	}
	metrics := &getPrivateObjectMetrics{
		err: errors.New("metrics unavailable"),
	}

	uc := usecases.NewGetPrivateObjectUseCase(storage, logger, metrics)

	output, err := uc.Execute(
		context.Background(),
		dto.GetPrivateObjectInput{
			Reference: "knowledge/store-id/data.bin",
		},
	)

	require.NoError(t, err)
	require.NotNil(t, output.Content)

	content, readErr := io.ReadAll(output.Content)
	require.NoError(t, readErr)
	require.NoError(t, output.Content.Close())
	require.Equal(t, []byte("private-object"), content)
}

type getPrivateObjectStorage struct {
	reference   string
	content     []byte
	contentType string
	err         error
}

func (f *getPrivateObjectStorage) PutObject(
	_ context.Context,
	_ string,
	_ io.Reader,
	_ string,
) error {
	return errors.New("unexpected PutObject call")
}

func (f *getPrivateObjectStorage) GetPublicObject(
	_ context.Context,
	_ string,
) (io.ReadCloser, string, error) {
	return nil, "", errors.New("unexpected GetPublicObject call")
}

func (f *getPrivateObjectStorage) HeadObject(
	_ context.Context,
	_ string,
) (string, error) {
	return "", errors.New("unexpected HeadObject call")
}

func (f *getPrivateObjectStorage) DeleteObject(
	_ context.Context,
	_ string,
) error {
	return errors.New("unexpected DeleteObject call")
}

func (f *getPrivateObjectStorage) GetPrivateObject(
	_ context.Context,
	reference string,
) (io.ReadCloser, string, error) {
	f.reference = reference

	if f.err != nil {
		return nil, "", f.err
	}

	return io.NopCloser(bytes.NewReader(f.content)), f.contentType, nil
}

type getPrivateObjectLogger struct {
	successCalled bool
	err           error
}

func (f *getPrivateObjectLogger) Log(
	_ context.Context,
	event ports.LogEvent,
) error {
	if event.Event == "storage.get_private_object.succeeded" {
		f.successCalled = true
	}

	return f.err
}

type getPrivateObjectMetrics struct {
	incrementCalls int
	observeCalls   int
	err            error
}

func (f *getPrivateObjectMetrics) Increment(
	_ context.Context,
	_ ports.Metric,
) error {
	f.incrementCalls++
	return f.err
}

func (f *getPrivateObjectMetrics) Observe(
	_ context.Context,
	_ ports.Metric,
) error {
	f.observeCalls++
	return f.err
}
