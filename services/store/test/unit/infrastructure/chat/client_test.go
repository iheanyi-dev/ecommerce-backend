package chat

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	chatv1 "github.com/iheanyi-dev/ecommerce-backend/proto/chat/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	chatinfra "github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/chat"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/shared/config"
)

type testLifecycle struct {
	hooks []fx.Hook
}

func (l *testLifecycle) Append(hook fx.Hook) {
	l.hooks = append(l.hooks, hook)
}

func (l *testLifecycle) stop(t *testing.T) {
	t.Helper()
	for i := len(l.hooks) - 1; i >= 0; i-- {
		if l.hooks[i].OnStop != nil {
			require.NoError(t, l.hooks[i].OnStop(context.Background()))
		}
	}
}

type testChatServer struct {
	chatv1.UnimplementedChatServiceServer
	request  *chatv1.StreamMessageRequest
	metadata metadata.MD
	err      error
	received chan struct{}
}

func (s *testChatServer) StreamMessage(
	request *chatv1.StreamMessageRequest,
	stream grpc.ServerStreamingServer[chatv1.StreamMessageResponse],
) error {
	s.request = request
	s.metadata, _ = metadata.FromIncomingContext(stream.Context())
	if s.received != nil {
		close(s.received)
	}

	if s.err != nil {
		return s.err
	}

	return stream.Send(&chatv1.StreamMessageResponse{
		Type:           "token",
		Content:        "Hello",
		ConversationId: request.GetConversationId(),
	})
}

func newTestDispatcher(
	t *testing.T,
	server *testChatServer,
) (ports.ChatDispatcher, *testLifecycle) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer()
	chatv1.RegisterChatServiceServer(grpcServer, server)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	lifecycle := &testLifecycle{}
	cfg := &config.Config{
		ChatServiceAddr:   listener.Addr().String(),
		ChatServiceName:   "store",
		ChatServiceSecret: "store-chat-test-secret",
	}

	dispatcher, err := chatinfra.NewClient(lifecycle, cfg)
	require.NoError(t, err)

	t.Cleanup(func() {
		lifecycle.stop(t)
		grpcServer.Stop()
		_ = listener.Close()
	})

	return dispatcher, lifecycle
}

func TestStartChatForwardsRequestAndServiceAuthentication(t *testing.T) {
	server := &testChatServer{received: make(chan struct{})}
	dispatcher, _ := newTestDispatcher(t, server)

	storeID := uuid.New()
	userID := uuid.New()
	conversationID := uuid.New()

	stream, err := dispatcher.StartChat(
		context.Background(),
		ports.ChatRequest{
			StoreID:        storeID,
			UserID:         userID,
			ConversationID: &conversationID,
			Message:        "Find me a product",
		},
	)
	require.NoError(t, err)
	select {
	case <-server.received:
	case <-time.After(time.Second):
		t.Fatal("Chat server did not receive the request")
	}

	require.Equal(t, storeID.String(), server.request.GetStoreId())
	require.Equal(t, userID.String(), server.request.GetUserId())
	require.Equal(t, conversationID.String(), server.request.GetConversationId())
	require.Equal(t, "Find me a product", server.request.GetMessage())
	require.Equal(t, []string{"store"}, server.metadata.Get("x-service-name"))
	require.Equal(
		t,
		[]string{"store-chat-test-secret"},
		server.metadata.Get("x-service-secret"),
	)

	event, err := stream.Recv(context.Background())
	require.NoError(t, err)
	require.Equal(t, "token", event.Type)
	require.Equal(t, "Hello", event.Content)
	require.NotNil(t, event.ConversationID)
	require.Equal(t, conversationID, *event.ConversationID)
}

func TestStartChatAllowsOptionalUserAndConversationIDs(t *testing.T) {
	server := &testChatServer{}
	dispatcher, _ := newTestDispatcher(t, server)

	stream, err := dispatcher.StartChat(
		context.Background(),
		ports.ChatRequest{
			StoreID: uuid.New(),
			Message: "Hello",
		},
	)
	require.NoError(t, err)

	require.Empty(t, server.request.GetUserId())
	require.Empty(t, server.request.GetConversationId())

	event, err := stream.Recv(context.Background())
	require.NoError(t, err)
	require.Nil(t, event.ConversationID)
}

func TestStartChatPropagatesGRPCError(t *testing.T) {
	server := &testChatServer{
		err: status.Error(codes.PermissionDenied, "service not authorized"),
	}
	dispatcher, _ := newTestDispatcher(t, server)

	stream, err := dispatcher.StartChat(
		context.Background(),
		ports.ChatRequest{
			StoreID: uuid.New(),
			Message: "Hello",
		},
	)
	require.NoError(t, err)

	_, err = stream.Recv(context.Background())
	require.Error(t, err)
	require.Equal(t, codes.PermissionDenied, status.Code(err))
}
