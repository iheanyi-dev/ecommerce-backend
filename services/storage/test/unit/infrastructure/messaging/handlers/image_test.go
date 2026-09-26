package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/infrastructure/messaging/events"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/infrastructure/messaging/handlers"
)

func TestImageEventHandler_HandleStoreImage(t *testing.T) {
	storeID := uuid.New()

	t.Run("processes valid event", func(t *testing.T) {
		putObject := &putObjectService{}
		deleteObject := &deleteObjectService{}
		tracer := &testTracer{}

		handler := handlers.NewImageEventHandler(
			putObject,
			deleteObject,
			tracer,
		)

		record := newRecord(t, events.StoreImageRequested{
			StoreID:     storeID,
			Reference:   "stores/store-id.png",
			ContentType: "image/png",
			Content:     []byte("image-content"),
		})

		err := handler.HandleStoreImage(
			context.Background(),
			record,
		)

		require.NoError(t, err)
		require.True(t, putObject.called)
		require.Equal(t, "stores/store-id.png", putObject.input.Reference)
		require.Equal(t, "image/png", putObject.input.ContentType)
		require.Equal(t, []byte("image-content"), putObject.content)
		require.False(t, deleteObject.called)
		require.Equal(t, "storage.kafka.store_image", tracer.spanName)
		require.True(t, tracer.spanEnded)
	})

	t.Run("rejects invalid JSON", func(t *testing.T) {
		putObject := &putObjectService{}
		handler := handlers.NewImageEventHandler(
			putObject,
			&deleteObjectService{},
			&testTracer{},
		)

		record := &kgo.Record{
			Value: []byte(`{"invalid"`),
		}

		err := handler.HandleStoreImage(
			context.Background(),
			record,
		)

		require.Error(t, err)
		require.False(t, putObject.called)
	})

	t.Run("rejects missing store ID", func(t *testing.T) {
		putObject := &putObjectService{}
		handler := handlers.NewImageEventHandler(
			putObject,
			&deleteObjectService{},
			&testTracer{},
		)

		record := newRecord(t, events.StoreImageRequested{
			Reference:   "stores/store-id.png",
			ContentType: "image/png",
			Content:     []byte("image-content"),
		})

		err := handler.HandleStoreImage(
			context.Background(),
			record,
		)

		require.EqualError(
			t,
			err,
			"store image event: store ID is required",
		)
		require.False(t, putObject.called)
	})

	t.Run("rejects missing reference", func(t *testing.T) {
		putObject := &putObjectService{}
		handler := handlers.NewImageEventHandler(
			putObject,
			&deleteObjectService{},
			&testTracer{},
		)

		record := newRecord(t, events.StoreImageRequested{
			StoreID:     storeID,
			ContentType: "image/png",
			Content:     []byte("image-content"),
		})

		err := handler.HandleStoreImage(
			context.Background(),
			record,
		)

		require.EqualError(
			t,
			err,
			"store image event: reference is required",
		)
		require.False(t, putObject.called)
	})

	t.Run("rejects missing content type", func(t *testing.T) {
		putObject := &putObjectService{}
		handler := handlers.NewImageEventHandler(
			putObject,
			&deleteObjectService{},
			&testTracer{},
		)

		record := newRecord(t, events.StoreImageRequested{
			StoreID:   storeID,
			Reference: "stores/store-id.png",
			Content:   []byte("image-content"),
		})

		err := handler.HandleStoreImage(
			context.Background(),
			record,
		)

		require.EqualError(
			t,
			err,
			"store image event: content type is required",
		)
		require.False(t, putObject.called)
	})

	t.Run("propagates application error", func(t *testing.T) {
		expectedErr := errors.New("put object failed")

		putObject := &putObjectService{
			err: expectedErr,
		}

		handler := handlers.NewImageEventHandler(
			putObject,
			&deleteObjectService{},
			&testTracer{},
		)

		record := newRecord(t, events.StoreImageRequested{
			StoreID:     storeID,
			Reference:   "stores/store-id.png",
			ContentType: "image/png",
			Content:     []byte("image-content"),
		})

		err := handler.HandleStoreImage(
			context.Background(),
			record,
		)

		require.ErrorIs(t, err, expectedErr)
	})
}

func TestImageEventHandler_HandleStoreImageDeletion(t *testing.T) {
	storeID := uuid.New()

	t.Run("processes valid event", func(t *testing.T) {
		putObject := &putObjectService{}
		deleteObject := &deleteObjectService{}
		tracer := &testTracer{}

		handler := handlers.NewImageEventHandler(
			putObject,
			deleteObject,
			tracer,
		)

		record := newRecord(t, events.StoreImageDeletionRequested{
			StoreID:   storeID,
			Reference: "stores/store-id.png",
		})

		err := handler.HandleStoreImageDeletion(
			context.Background(),
			record,
		)

		require.NoError(t, err)
		require.True(t, deleteObject.called)
		require.Equal(
			t,
			"stores/store-id.png",
			deleteObject.input.Reference,
		)
		require.False(t, putObject.called)
		require.Equal(
			t,
			"storage.kafka.store_image_deletion",
			tracer.spanName,
		)
		require.True(t, tracer.spanEnded)
	})

	t.Run("rejects invalid JSON", func(t *testing.T) {
		deleteObject := &deleteObjectService{}

		handler := handlers.NewImageEventHandler(
			&putObjectService{},
			deleteObject,
			&testTracer{},
		)

		record := &kgo.Record{
			Value: []byte(`{"invalid"`),
		}

		err := handler.HandleStoreImageDeletion(
			context.Background(),
			record,
		)

		require.Error(t, err)
		require.False(t, deleteObject.called)
	})

	t.Run("rejects missing store ID", func(t *testing.T) {
		deleteObject := &deleteObjectService{}

		handler := handlers.NewImageEventHandler(
			&putObjectService{},
			deleteObject,
			&testTracer{},
		)

		record := newRecord(t, events.StoreImageDeletionRequested{
			Reference: "stores/store-id.png",
		})

		err := handler.HandleStoreImageDeletion(
			context.Background(),
			record,
		)

		require.EqualError(
			t,
			err,
			"store image deletion event: store ID is required",
		)
		require.False(t, deleteObject.called)
	})

	t.Run("rejects missing reference", func(t *testing.T) {
		deleteObject := &deleteObjectService{}

		handler := handlers.NewImageEventHandler(
			&putObjectService{},
			deleteObject,
			&testTracer{},
		)

		record := newRecord(t, events.StoreImageDeletionRequested{
			StoreID: storeID,
		})

		err := handler.HandleStoreImageDeletion(
			context.Background(),
			record,
		)

		require.EqualError(
			t,
			err,
			"store image deletion event: reference is required",
		)
		require.False(t, deleteObject.called)
	})

	t.Run("propagates application error", func(t *testing.T) {
		expectedErr := errors.New("delete object failed")

		deleteObject := &deleteObjectService{
			err: expectedErr,
		}

		handler := handlers.NewImageEventHandler(
			&putObjectService{},
			deleteObject,
			&testTracer{},
		)

		record := newRecord(t, events.StoreImageDeletionRequested{
			StoreID:   storeID,
			Reference: "stores/store-id.png",
		})

		err := handler.HandleStoreImageDeletion(
			context.Background(),
			record,
		)

		require.ErrorIs(t, err, expectedErr)
	})
}

type putObjectService struct {
	called  bool
	input   dto.PutObjectInput
	content []byte
	err     error
}

func (f *putObjectService) Execute(
	_ context.Context,
	input dto.PutObjectInput,
) error {
	f.called = true
	f.input = input

	if input.Content != nil {
		data, err := ioReadAll(input.Content)
		if err != nil {
			return err
		}

		f.content = data
	}

	return f.err
}

type deleteObjectService struct {
	called bool
	input  dto.DeleteObjectInput
	err    error
}

func (f *deleteObjectService) Execute(
	_ context.Context,
	input dto.DeleteObjectInput,
) error {
	f.called = true
	f.input = input

	return f.err
}

type testTracer struct {
	spanName  string
	spanEnded bool
}

func (t *testTracer) Start(
	ctx context.Context,
	name string,
) (context.Context, ports.Span) {
	t.spanName = name

	return ctx, &testSpan{
		parent: t,
	}
}

type testSpan struct {
	parent *testTracer
}

func (s *testSpan) End() {
	s.parent.spanEnded = true
}

func newRecord(
	t *testing.T,
	event any,
) *kgo.Record {
	t.Helper()

	value, err := json.Marshal(event)
	require.NoError(t, err)

	return &kgo.Record{
		Value: value,
	}
}

// Kept local to the test so the test doubles do not depend on additional
// production helpers.
func ioReadAll(reader interface{ Read([]byte) (int, error) }) ([]byte, error) {
	var result []byte
	buffer := make([]byte, 1024)

	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			result = append(result, buffer[:n]...)
		}

		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, err
			}

			if err.Error() == "EOF" {
				return result, nil
			}

			return result, err
		}
	}
}
