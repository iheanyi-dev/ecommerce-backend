package kafka_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/messaging/kafka"
)

const testSubscriptionTopic = "billing.subscription.events"

func TestSubscriptionEventConsumer_HandleRecord(t *testing.T) {
	eventID := uuid.New()
	subscriptionID := uuid.New()
	storeID := uuid.New()

	payload, err := json.Marshal(map[string]string{
		"event_id":        eventID.String(),
		"subscription_id": subscriptionID.String(),
		"store_id":        storeID.String(),
		"plan_type":       "premium",
		"status":          "active",
	})
	require.NoError(t, err)

	service := &updateAfterSubscriptionFake{}
	consumer := kafka.NewSubscriptionEventConsumerForTest(testSubscriptionTopic, service)

	err = consumer.HandleRecordForTest(context.Background(), &kgo.Record{
		Topic: testSubscriptionTopic,
		Value: payload,
	})

	require.NoError(t, err)
	require.True(t, service.called)
	assert.Equal(t, eventID, service.input.EventID)
	assert.Equal(t, subscriptionID, service.input.SubscriptionID)
	assert.Equal(t, storeID, service.input.StoreID)
	assert.Equal(t, "premium", service.input.PlanType)
	assert.Equal(t, "active", service.input.Status)
}

func TestSubscriptionEventConsumer_RejectsMalformedPayload(t *testing.T) {
	consumer := kafka.NewSubscriptionEventConsumerForTest(testSubscriptionTopic, &updateAfterSubscriptionFake{})

	err := consumer.HandleRecordForTest(context.Background(), &kgo.Record{
		Topic: testSubscriptionTopic,
		Value: []byte("{"),
	})

	require.Error(t, err)
}

func TestSubscriptionEventConsumer_RejectsInvalidIdentifiers(t *testing.T) {
	payload, err := json.Marshal(map[string]string{
		"event_id":        "invalid",
		"subscription_id": uuid.NewString(),
		"store_id":        uuid.NewString(),
		"plan_type":       "premium",
		"status":          "active",
	})
	require.NoError(t, err)

	consumer := kafka.NewSubscriptionEventConsumerForTest(testSubscriptionTopic, &updateAfterSubscriptionFake{})

	err = consumer.HandleRecordForTest(context.Background(), &kgo.Record{
		Topic: testSubscriptionTopic,
		Value: payload,
	})

	require.Error(t, err)
}

func TestSubscriptionEventConsumer_RejectsUnexpectedTopic(t *testing.T) {
	consumer := kafka.NewSubscriptionEventConsumerForTest(testSubscriptionTopic, &updateAfterSubscriptionFake{})

	err := consumer.HandleRecordForTest(context.Background(), &kgo.Record{
		Topic: "another.topic",
		Value: []byte("{}"),
	})

	require.Error(t, err)
}

func TestSubscriptionEventConsumer_PropagatesApplicationFailure(t *testing.T) {
	payload, err := json.Marshal(map[string]string{
		"event_id":        uuid.NewString(),
		"subscription_id": uuid.NewString(),
		"store_id":        uuid.NewString(),
		"plan_type":       "premium",
		"status":          "active",
	})
	require.NoError(t, err)

	service := &updateAfterSubscriptionFake{
		err: errors.New("repository unavailable"),
	}
	consumer := kafka.NewSubscriptionEventConsumerForTest(testSubscriptionTopic, service)

	err = consumer.HandleRecordForTest(context.Background(), &kgo.Record{
		Topic: testSubscriptionTopic,
		Value: payload,
	})

	require.Error(t, err)
	assert.True(t, service.called)
}

type updateAfterSubscriptionFake struct {
	input  dto.UpdateAfterSubscriptionInput
	called bool
	err    error
}

func (f *updateAfterSubscriptionFake) Execute(
	_ context.Context,
	input dto.UpdateAfterSubscriptionInput,
) (dto.StoreOutput, error) {
	f.called = true
	f.input = input
	return dto.StoreOutput{}, f.err
}

var _ ports.UpdateAfterSubscriptionService = (*updateAfterSubscriptionFake)(nil)
