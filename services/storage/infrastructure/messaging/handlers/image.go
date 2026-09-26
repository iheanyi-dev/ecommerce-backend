package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel/propagation"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/infrastructure/messaging/events"
)

// ImageEventHandler processes Store image events received from Kafka.
//
// Kafka transport concerns remain in the messaging consumer. This handler
// translates validated Kafka events into application use-case inputs.
type ImageEventHandler struct {
	putObject    ports.PutObjectService
	deleteObject ports.DeleteObjectService
	tracer       ports.Tracer
}

// NewImageEventHandler constructs the Store image event handler.
func NewImageEventHandler(
	putObject ports.PutObjectService,
	deleteObject ports.DeleteObjectService,
	tracer ports.Tracer,
) *ImageEventHandler {
	return &ImageEventHandler{
		putObject:    putObject,
		deleteObject: deleteObject,
		tracer:       tracer,
	}
}

// HandleStoreImage processes a store.image.requested Kafka record.
func (h *ImageEventHandler) HandleStoreImage(
	ctx context.Context,
	record *kgo.Record,
) error {
	ctx = extractTraceContext(ctx, record)

	ctx, span := h.tracer.Start(
		ctx,
		"storage.kafka.store_image",
	)
	defer span.End()

	var event events.StoreImageRequested

	if err := json.Unmarshal(record.Value, &event); err != nil {
		return err
	}

	if event.StoreID == [16]byte{} {
		return errors.New("store image event: store ID is required")
	}

	if event.Reference == "" {
		return errors.New("store image event: reference is required")
	}

	if event.ContentType == "" {
		return errors.New("store image event: content type is required")
	}

	return h.putObject.Execute(ctx, dto.PutObjectInput{
		Reference:   event.Reference,
		Content:     bytes.NewReader(event.Content),
		ContentType: event.ContentType,
	})
}

// HandleStoreImageDeletion processes a store.image.deletion.requested Kafka
// record.
func (h *ImageEventHandler) HandleStoreImageDeletion(
	ctx context.Context,
	record *kgo.Record,
) error {
	ctx = extractTraceContext(ctx, record)

	ctx, span := h.tracer.Start(
		ctx,
		"storage.kafka.store_image_deletion",
	)
	defer span.End()

	var event events.StoreImageDeletionRequested

	if err := json.Unmarshal(record.Value, &event); err != nil {
		return err
	}

	if event.StoreID == [16]byte{} {
		return errors.New("store image deletion event: store ID is required")
	}

	if event.Reference == "" {
		return errors.New("store image deletion event: reference is required")
	}

	return h.deleteObject.Execute(ctx, dto.DeleteObjectInput{
		Reference: event.Reference,
	})
}

func extractTraceContext(
	ctx context.Context,
	record *kgo.Record,
) context.Context {
	carrier := propagation.MapCarrier{}

	for _, header := range record.Headers {
		carrier[header.Key] = string(header.Value)
	}

	return propagation.TraceContext{}.Extract(ctx, carrier)
}
