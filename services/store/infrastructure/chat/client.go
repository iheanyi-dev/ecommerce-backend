package chat

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	chatv1 "github.com/iheanyi-dev/ecommerce-backend/proto/chat/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/shared/config"
)

func NewClient(
	lc fx.Lifecycle,
	cfg *config.Config,
) (ports.ChatDispatcher, error) {
	if cfg.ChatServiceAddr == "" {
		return nil, fmt.Errorf("CHAT_SERVICE_ADDR is required")
	}
	if cfg.ChatServiceName == "" {
		return nil, fmt.Errorf("CHAT_SERVICE_NAME is required")
	}
	if cfg.ChatServiceSecret == "" {
		return nil, fmt.Errorf("CHAT_SERVICE_SECRET is required")
	}

	conn, err := grpc.NewClient(
		cfg.ChatServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return conn.Close()
		},
	})

	return newDispatcher(
		chatv1.NewChatServiceClient(conn),
		cfg.ChatServiceName,
		cfg.ChatServiceSecret,
	), nil
}

type dispatcher struct {
	client        chatv1.ChatServiceClient
	serviceName   string
	serviceSecret string
}

func newDispatcher(
	client chatv1.ChatServiceClient,
	serviceName string,
	serviceSecret string,
) ports.ChatDispatcher {
	return &dispatcher{
		client:        client,
		serviceName:   serviceName,
		serviceSecret: serviceSecret,
	}
}

func (d *dispatcher) StartChat(
	ctx context.Context,
	request ports.ChatRequest,
) (ports.ChatStream, error) {
	conversationID := ""
	if request.ConversationID != nil {
		conversationID = request.ConversationID.String()
	}

	userID := ""
	if request.UserID != uuid.Nil {
		userID = request.UserID.String()
	}

	ctx = metadata.AppendToOutgoingContext(
		ctx,
		"x-service-name", d.serviceName,
		"x-service-secret", d.serviceSecret,
	)

	stream, err := d.client.StreamMessage(
		ctx,
		&chatv1.StreamMessageRequest{
			StoreId:        request.StoreID.String(),
			UserId:         userID,
			ConversationId: conversationID,
			Message:        request.Message,
		},
	)
	if err != nil {
		return nil, err
	}

	return &chatStream{stream: stream}, nil
}

type chatStream struct {
	stream grpc.ServerStreamingClient[chatv1.StreamMessageResponse]
}

func (s *chatStream) Recv(ctx context.Context) (ports.ChatEvent, error) {
	_ = ctx

	response, err := s.stream.Recv()
	if err != nil {
		return ports.ChatEvent{}, err
	}

	var conversationID *uuid.UUID
	if response.GetConversationId() != "" {
		id, err := uuid.Parse(response.GetConversationId())
		if err != nil {
			return ports.ChatEvent{}, err
		}
		conversationID = &id
	}

	return ports.ChatEvent{
		Type:           response.GetType(),
		Content:        response.GetContent(),
		ConversationID: conversationID,
	}, nil
}
