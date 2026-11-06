package grpc

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/proto/billing/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/value_objects"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	billingv1.UnimplementedBillingServiceServer

	createPlan *use_cases.CreatePlan
	updatePlan *use_cases.UpdatePlan
	deletePlan *use_cases.DeletePlanByID
	getPlan    *use_cases.GetPlanByID
	getPlans   *use_cases.GetPlans
	cancelSub  *use_cases.CancelSub
	getSub     *use_cases.GetSubByID
	getSubs    *use_cases.GetSubsByStoreID
	subscribe  *use_cases.Subscribe
}

func NewServer(
	createPlan *use_cases.CreatePlan,
	updatePlan *use_cases.UpdatePlan,
	deletePlan *use_cases.DeletePlanByID,
	getPlan *use_cases.GetPlanByID,
	getPlans *use_cases.GetPlans,
	cancelSub *use_cases.CancelSub,
	getSub *use_cases.GetSubByID,
	getSubs *use_cases.GetSubsByStoreID,
	subscribe *use_cases.Subscribe,
) *Server {
	return &Server{
		createPlan: createPlan,
		updatePlan: updatePlan,
		deletePlan: deletePlan,
		getPlan:    getPlan,
		getPlans:   getPlans,
		cancelSub:  cancelSub,
		getSub:     getSub,
		getSubs:    getSubs,
		subscribe:  subscribe,
	}
}

func (s *Server) CreatePlan(ctx context.Context, req *billingv1.CreatePlanRequest) (*billingv1.PlanResponse, error) {
	if req == nil || req.Price == nil || req.Capabilities == nil {
		return nil, status.Error(codes.InvalidArgument, "price and capabilities are required")
	}
	price, err := value_objects.NewMoney(req.Price.AmountMinor, req.Price.Currency)
	if err != nil {
		return nil, mapError(err)
	}
	input := use_cases.CreatePlanInput{
		Type: req.Type, Name: req.Name, Description: req.Description,
		Price: price, Duration: duration(req.DurationSeconds),
		Capabilities: entities.PlanCapabilities{MaxProducts: int(req.Capabilities.MaxProducts), AIEnabled: req.Capabilities.AiEnabled},
		Available:    req.Available, Promotion: promotionFromProto(req.Promotion),
	}
	plan, err := s.createPlan.Execute(ctx, input)
	if err != nil {
		return nil, mapError(err)
	}
	return &billingv1.PlanResponse{Plan: planToProto(plan)}, nil
}

func (s *Server) UpdatePlan(ctx context.Context, req *billingv1.UpdatePlanRequest) (*billingv1.PlanResponse, error) {
	if req == nil || req.Price == nil || req.Capabilities == nil {
		return nil, status.Error(codes.InvalidArgument, "id, price and capabilities are required")
	}
	id, err := parseID(req.Id)
	if err != nil {
		return nil, err
	}
	price, err := value_objects.NewMoney(req.Price.AmountMinor, req.Price.Currency)
	if err != nil {
		return nil, mapError(err)
	}
	input := use_cases.UpdatePlanInput{
		Type: req.Type, Name: req.Name, Description: req.Description,
		Price: price, Duration: duration(req.DurationSeconds),
		Capabilities: entities.PlanCapabilities{MaxProducts: int(req.Capabilities.MaxProducts), AIEnabled: req.Capabilities.AiEnabled},
		Available:    req.Available, Promotion: promotionFromProto(req.Promotion),
	}
	plan, err := s.updatePlan.Execute(ctx, id, input)
	if err != nil {
		return nil, mapError(err)
	}
	return &billingv1.PlanResponse{Plan: planToProto(plan)}, nil
}

func (s *Server) DeletePlan(ctx context.Context, req *billingv1.DeletePlanRequest) (*billingv1.Empty, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	id, err := parseID(req.Id)
	if err != nil {
		return nil, err
	}
	if err := s.deletePlan.Execute(ctx, id); err != nil {
		return nil, mapError(err)
	}
	return &billingv1.Empty{}, nil
}

func (s *Server) GetPlan(ctx context.Context, req *billingv1.GetPlanRequest) (*billingv1.PlanResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	id, err := parseID(req.Id)
	if err != nil {
		return nil, err
	}
	plan, err := s.getPlan.Execute(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	return &billingv1.PlanResponse{Plan: planToProto(plan)}, nil
}

func (s *Server) ListPlans(ctx context.Context, _ *billingv1.Empty) (*billingv1.ListPlansResponse, error) {
	plans, err := s.getPlans.Execute(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	result := &billingv1.ListPlansResponse{Plans: make([]*billingv1.Plan, 0, len(plans))}
	for _, plan := range plans {
		result.Plans = append(result.Plans, planToProto(plan))
	}
	return result, nil
}

func (s *Server) Subscribe(ctx context.Context, req *billingv1.SubscribeRequest) (*billingv1.SubscribeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	storeID, err := parseID(req.StoreId)
	if err != nil {
		return nil, err
	}
	payerID, err := parseID(req.PayerId)
	if err != nil {
		return nil, err
	}
	planID, err := parseID(req.PlanId)
	if err != nil {
		return nil, err
	}
	result, err := s.subscribe.Execute(ctx, use_cases.SubscribeInput{
		StoreID: storeID, PayerID: payerID, PlanID: planID, CustomerEmail: req.CustomerEmail,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &billingv1.SubscribeResponse{
		Subscription: subscriptionToProto(result.Subscription),
		PaymentId:    result.Payment.PaymentID,
		CheckoutUrl:  result.Payment.CheckoutURL,
	}, nil
}

func (s *Server) CancelSubscription(ctx context.Context, req *billingv1.CancelSubscriptionRequest) (*billingv1.SubscriptionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	storeID, err := parseID(req.StoreId)
	if err != nil {
		return nil, err
	}
	subscriptionID, err := parseID(req.SubscriptionId)
	if err != nil {
		return nil, err
	}
	subscription, err := s.cancelSub.Execute(ctx, use_cases.CancelSubInput{
		StoreID: storeID, SubscriptionID: subscriptionID,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &billingv1.SubscriptionResponse{Subscription: subscriptionToProto(subscription)}, nil
}

func (s *Server) GetSubscription(ctx context.Context, req *billingv1.GetSubscriptionRequest) (*billingv1.SubscriptionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	id, err := parseID(req.SubscriptionId)
	if err != nil {
		return nil, err
	}
	result, err := s.getSub.Execute(ctx, use_cases.GetSubByIDInput{SubscriptionID: id})
	if err != nil {
		return nil, mapError(err)
	}
	return &billingv1.SubscriptionResponse{Subscription: subscriptionToProto(result.Subscription)}, nil
}

func (s *Server) ListStoreSubscriptions(ctx context.Context, req *billingv1.ListStoreSubscriptionsRequest) (*billingv1.ListStoreSubscriptionsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	storeID, err := parseID(req.StoreId)
	if err != nil {
		return nil, err
	}
	result, err := s.getSubs.Execute(ctx, use_cases.GetSubsByStoreIDInput{StoreID: storeID})
	if err != nil {
		return nil, mapError(err)
	}
	response := &billingv1.ListStoreSubscriptionsResponse{
		Subscriptions: make([]*billingv1.Subscription, 0, len(result.Subscriptions)),
	}
	for _, subscription := range result.Subscriptions {
		response.Subscriptions = append(response.Subscriptions, subscriptionToProto(subscription))
	}
	return response, nil
}

func parseID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, status.Error(codes.InvalidArgument, "a valid UUID is required")
	}
	return id, nil
}

func duration(seconds int64) time.Duration {
	return time.Duration(seconds) * time.Second
}

func promotionFromProto(p *billingv1.Promotion) *entities.Promotion {
	if p == nil {
		return nil
	}
	result := &entities.Promotion{Name: p.Name, DiscountAmountMinor: p.DiscountAmountMinor}
	if p.StartsAt != nil {
		result.StartsAt = p.StartsAt.AsTime()
	}
	if p.EndsAt != nil {
		result.EndsAt = p.EndsAt.AsTime()
	}
	return result
}

func planToProto(plan entities.Plan) *billingv1.Plan {
	result := &billingv1.Plan{
		Id: plan.ID.String(), Type: plan.Type, Name: plan.Name, Description: plan.Description,
		Price:           &billingv1.Money{AmountMinor: plan.Price.AmountMinor, Currency: plan.Price.Currency},
		DurationSeconds: int64(plan.Duration / time.Second),
		Capabilities:    &billingv1.PlanCapabilities{MaxProducts: int32(plan.Capabilities.MaxProducts), AiEnabled: plan.Capabilities.AIEnabled},
		Available:       plan.Available,
		CreatedAt:       timestamppb.New(plan.CreatedAt), UpdatedAt: timestamppb.New(plan.UpdatedAt),
	}
	if plan.Promotion != nil {
		result.Promotion = &billingv1.Promotion{
			Name: plan.Promotion.Name, DiscountAmountMinor: plan.Promotion.DiscountAmountMinor,
			StartsAt: timestamppb.New(plan.Promotion.StartsAt), EndsAt: timestamppb.New(plan.Promotion.EndsAt),
		}
	}
	return result
}

func subscriptionToProto(subscription entities.Subscription) *billingv1.Subscription {
	result := &billingv1.Subscription{
		Id: subscription.ID.String(), StoreId: subscription.StoreID.String(), PlanId: subscription.PlanID.String(),
		Price:           &billingv1.Money{AmountMinor: subscription.Price.AmountMinor, Currency: subscription.Price.Currency},
		DurationSeconds: int64(subscription.Duration / time.Second), Status: string(subscription.Status),
		CreatedAt: timestamppb.New(subscription.CreatedAt), UpdatedAt: timestamppb.New(subscription.UpdatedAt),
	}
	if !subscription.StartedAt.IsZero() {
		result.StartedAt = timestamppb.New(subscription.StartedAt)
	}
	if !subscription.ExpiresAt.IsZero() {
		result.ExpiresAt = timestamppb.New(subscription.ExpiresAt)
	}
	return result
}

func mapError(err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request canceled")
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "request deadline exceeded")
	case errors.Is(err, application.ErrPlanNotFound), errors.Is(err, application.ErrSubscriptionNotFound):
		return status.Error(codes.NotFound, "resource not found")
	case errors.Is(err, application.ErrPlanAlreadyExists), errors.Is(err, application.ErrSubscriptionConflict),
		errors.Is(err, application.ErrActiveSubscriptionExists), errors.Is(err, application.ErrSubscriptionAlreadyPending):
		return status.Error(codes.AlreadyExists, "resource conflict")
	case errors.Is(err, application.ErrPlanInUse):
		return status.Error(codes.FailedPrecondition, "plan is in use")
	case errors.Is(err, application.ErrPlanUnavailable):
		return status.Error(codes.FailedPrecondition, "plan is unavailable")
	case errors.Is(err, application.ErrPaymentUnavailable), errors.Is(err, application.ErrStoreNotifierUnavailable):
		return status.Error(codes.Unavailable, "dependency unavailable")
	case errors.Is(err, application.ErrPlanRepositoryUnavailable), errors.Is(err, application.ErrSubscriptionRepositoryUnavailable):
		return status.Error(codes.Unavailable, "repository unavailable")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
