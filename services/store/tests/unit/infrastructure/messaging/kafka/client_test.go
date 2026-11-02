package kafka_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	"github.com/twmb/franz-go/pkg/kgo"

	kafkainfra "github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/messaging/kafka"
)

type fakeProducer struct {
	record *kgo.Record
	err    error
}

func (f *fakeProducer) ProduceSync(
	_ context.Context,
	records ...*kgo.Record,
) kgo.ProduceResults {
	if len(records) > 0 {
		f.record = records[0]
	}

	return kgo.ProduceResults{
		{
			Record: f.record,
			Err:    f.err,
		},
	}
}

func (f *fakeProducer) Close() {}

func TestClientPublishInjectsTraceContext(t *testing.T) {
	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SpanID:     [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
		TraceFlags: trace.FlagsSampled,
	})

	ctx := trace.ContextWithSpanContext(context.Background(), spanContext)

	producer := &fakeProducer{}
	client := kafkainfra.NewClientWithProducer(producer)

	err := client.Publish(
		ctx,
		"store.image.requested",
		"store-123",
		map[string]string{
			"reference": "stores/store-123.png",
		},
	)

	require.NoError(t, err)
	require.NotNil(t, producer.record)

	require.Equal(t, "store.image.requested", producer.record.Topic)
	require.Equal(t, []byte("store-123"), producer.record.Key)

	require.Contains(t, headerKeys(producer.record.Headers), "traceparent")

	var traceparent string

	for _, header := range producer.record.Headers {
		if header.Key == "traceparent" {
			traceparent = string(header.Value)
			break
		}
	}

	require.Equal(
		t,
		"00-0102030405060708090a0b0c0d0e0f10-0102030405060708-01",
		traceparent,
	)
}

func TestClientPublishPropagatesProducerError(t *testing.T) {
	expectedErr := errors.New("producer failed")

	producer := &fakeProducer{
		err: expectedErr,
	}

	client := kafkainfra.NewClientWithProducer(producer)

	err := client.Publish(
		context.Background(),
		"store.image.requested",
		"store-123",
		map[string]string{
			"reference": "stores/store-123.png",
		},
	)

	require.ErrorIs(t, err, expectedErr)
}

func TestClientPublishMarshalsValue(t *testing.T) {
	producer := &fakeProducer{}
	client := kafkainfra.NewClientWithProducer(producer)

	err := client.Publish(
		context.Background(),
		"store.image.requested",
		"store-123",
		map[string]string{
			"reference":   "stores/store-123.png",
			"contentType": "image/png",
		},
	)

	require.NoError(t, err)
	require.JSONEq(
		t,
		`{"reference":"stores/store-123.png","contentType":"image/png"}`,
		string(producer.record.Value),
	)
}

func TestClientPublishWithoutTraceContextDoesNotCreateTraceHeader(t *testing.T) {
	producer := &fakeProducer{}
	client := kafkainfra.NewClientWithProducer(producer)

	err := client.Publish(
		context.Background(),
		"store.image.requested",
		"store-123",
		map[string]string{
			"reference": "stores/store-123.png",
		},
	)

	require.NoError(t, err)
	require.NotNil(t, producer.record)
	require.NotContains(t, headerKeys(producer.record.Headers), "traceparent")
}

func headerKeys(headers []kgo.RecordHeader) []string {
	keys := make([]string, 0, len(headers))

	for _, header := range headers {
		keys = append(keys, header.Key)
	}

	return keys
}
