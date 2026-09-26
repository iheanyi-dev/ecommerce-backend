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

func TestPutObjectUseCase_StoresObject(t *testing.T) {
	storage := &putObjectStorage{}
	logger := &putObjectLogger{}
	metrics := &putObjectMetrics{}

	uc := usecases.NewPutObjectUseCase(
		storage,
		logger,
		metrics,
	)

	content := []byte("object-content")

	err := uc.Execute(
		context.Background(),
		dto.PutObjectInput{
			Reference:   "stores/store-id.png",
			Content:     bytes.NewReader(content),
			ContentType: "image/png",
		},
	)

	require.NoError(t, err)

	require.True(t, storage.called)
	require.Equal(t, "stores/store-id.png", storage.reference)
	require.Equal(t, content, storage.content)
	require.Equal(t, "image/png", storage.contentType)

	require.True(t, logger.successCalled)
	require.GreaterOrEqual(t, metrics.incrementCalls, 1)
	require.GreaterOrEqual(t, metrics.observeCalls, 1)
}

func TestPutObjectUseCase_StorageFailure(t *testing.T) {
	storageError := errors.New("storage backend unavailable")

	storage := &putObjectStorage{
		err: storageError,
	}

	logger := &putObjectLogger{}
	metrics := &putObjectMetrics{}

	uc := usecases.NewPutObjectUseCase(
		storage,
		logger,
		metrics,
	)

	err := uc.Execute(
		context.Background(),
		dto.PutObjectInput{
			Reference:   "stores/store-id.png",
			Content:     bytes.NewReader([]byte("object-content")),
			ContentType: "image/png",
		},
	)

	require.Error(t, err)
	require.ErrorIs(t, err, storageError)

	require.True(t, storage.called)
	require.False(t, logger.successCalled)
	require.GreaterOrEqual(t, metrics.incrementCalls, 1)
	require.GreaterOrEqual(t, metrics.observeCalls, 1)
}

func TestPutObjectUseCase_ObservabilityFailureDoesNotChangeSuccessfulResult(t *testing.T) {
	storage := &putObjectStorage{}
	logger := &putObjectLogger{
		err: errors.New("logger unavailable"),
	}
	metrics := &putObjectMetrics{
		err: errors.New("metrics unavailable"),
	}

	uc := usecases.NewPutObjectUseCase(
		storage,
		logger,
		metrics,
	)

	err := uc.Execute(
		context.Background(),
		dto.PutObjectInput{
			Reference:   "knowledge/store-id/file.pdf",
			Content:     bytes.NewReader([]byte("knowledge")),
			ContentType: "application/pdf",
		},
	)

	require.NoError(t, err)
	require.True(t, storage.called)
}

type putObjectStorage struct {
	called      bool
	reference   string
	content     []byte
	contentType string
	err         error
}

func (f *putObjectStorage) DeleteObject(
	_ context.Context,
	_ string,
) error {
	return errors.New("unexpected DeleteObject call")
}

func (f *putObjectStorage) GetPrivateObject(
	_ context.Context,
	_ string,
) (io.ReadCloser, string, error) {
	return nil, "", errors.New("unexpected GetPrivateObject call")
}

func (f *putObjectStorage) GetPublicObject(
	_ context.Context,
	_ string,
) (io.ReadCloser, string, error) {
	return nil, "", errors.New("unexpected GetPublicObject call")
}

func (f *putObjectStorage) PutObject(
	_ context.Context,
	reference string,
	content io.Reader,
	contentType string,
) error {
	f.called = true
	f.reference = reference
	f.contentType = contentType

	data, err := io.ReadAll(content)
	if err != nil {
		return err
	}

	f.content = data

	return f.err
}

type putObjectLogger struct {
	successCalled bool
	err           error
}

func (f *putObjectLogger) Log(
	_ context.Context,
	event ports.LogEvent,
) error {
	if event.Event == "storage.put_object.succeeded" {
		f.successCalled = true
	}

	return f.err
}

type putObjectMetrics struct {
	incrementCalls int
	observeCalls   int
	err            error
}

func (f *putObjectMetrics) Increment(
	_ context.Context,
	_ ports.Metric,
) error {
	f.incrementCalls++
	return f.err
}

func (f *putObjectMetrics) Observe(
	_ context.Context,
	_ ports.Metric,
) error {
	f.observeCalls++
	return f.err
}
