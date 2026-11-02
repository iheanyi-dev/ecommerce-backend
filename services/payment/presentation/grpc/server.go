package grpcserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/proto/payment/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/enums"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/value_objects"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/infrastructure"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	paymentv1.UnimplementedPaymentServiceServer
	useCases *infrastructure.UseCases
}

func NewServer(useCases *infrastructure.UseCases) *Server {
	return &Server{useCases: useCases}
}

func Register(server interface {
	RegisterService(*grpc.ServiceDesc, interface{})
}, paymentServer *Server) {
	paymentv1.RegisterPaymentServiceServer(server.(grpc.ServiceRegistrar), paymentServer)
}

func (s *Server) InitiatePayment(
	ctx context.Context,
	req *paymentv1.InitiatePaymentRequest,
) (*paymentv1.InitiatePaymentResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	amount, err := value_objects.NewMoney(req.GetAmountMinor(), req.GetCurrency())
	if err != nil {
		return nil, mapError(err)
	}

	result, err := s.useCases.InitiatePayment.Execute(ctx, use_cases.InitiatePaymentInput{
		PayerID:         req.GetPayerId(),
		Purpose:         enums.PaymentPurpose(req.GetPurpose()),
		BusinessRefType: req.GetBusinessReferenceType(),
		BusinessRefID:   req.GetBusinessReferenceId(),
		Amount:          amount,
		CustomerEmail:   req.GetCustomerEmail(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return &paymentv1.InitiatePaymentResponse{
		PaymentId:     result.PaymentID,
		AttemptId:     result.AttemptID,
		PaymentStatus: string(result.PaymentStatus),
		AttemptStatus: string(result.AttemptStatus),
		Provider:      result.Provider,
		CheckoutUrl:   result.CheckoutURL,
		AmountMinor:   result.Amount,
		Currency:      result.Currency,
	}, nil
}

func (s *Server) VerifyPayment(
	ctx context.Context,
	req *paymentv1.VerifyPaymentRequest,
) (*paymentv1.PaymentDetails, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	result, err := s.useCases.VerifyPayment.Execute(ctx, req.GetPaymentId())
	if err != nil {
		return nil, mapError(err)
	}
	return paymentDetailsToProto(result), nil
}

func (s *Server) GetPayment(
	ctx context.Context,
	req *paymentv1.GetPaymentRequest,
) (*paymentv1.PaymentDetails, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	result, err := s.useCases.GetPayment.Execute(ctx, req.GetPaymentId())
	if err != nil {
		return nil, mapError(err)
	}
	return paymentDetailsToProto(result), nil
}

func (s *Server) ListPayments(
	ctx context.Context,
	req *paymentv1.ListPaymentsRequest,
) (*paymentv1.ListPaymentsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	result, err := s.useCases.ListPayments.Execute(ctx, use_cases.ListPaymentsByPayerIDInput{
		PayerID:  req.GetPayerId(),
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	})
	if err != nil {
		return nil, mapError(err)
	}

	response := &paymentv1.ListPaymentsResponse{
		Page:       int32(result.Page),
		PageSize:   int32(result.PageSize),
		Total:      result.Total,
		TotalPages: int32(result.TotalPages),
	}
	for _, item := range result.Items {
		response.Items = append(response.Items, paymentDetailsToProto(item))
	}
	return response, nil
}

func (s *Server) InitiatePayout(
	ctx context.Context,
	req *paymentv1.InitiatePayoutRequest,
) (*paymentv1.PayoutOperationResponse, error) {
	if req == nil || req.GetBeneficiary() == nil {
		return nil, status.Error(codes.InvalidArgument, "request and beneficiary are required")
	}

	reference, err := value_objects.NewBusinessReference(
		req.GetBusinessReferenceType(),
		req.GetBusinessReferenceId(),
	)
	if err != nil {
		return nil, mapError(err)
	}
	amount, err := value_objects.NewMoney(req.GetAmountMinor(), req.GetCurrency())
	if err != nil {
		return nil, mapError(err)
	}

	b := req.GetBeneficiary()
	result, err := s.useCases.InitiatePayout.Execute(ctx, ports.PayoutRequest{
		BusinessReference: reference,
		StoreID:           req.GetStoreId(),
		OrderID:           req.GetOrderId(),
		Reason:            req.GetReason(),
		Amount:            amount,
		Beneficiary: entities.PayoutBeneficiary{
			AccountName:   b.GetAccountName(),
			AccountNumber: b.GetAccountNumber(),
			BankCode:      b.GetBankCode(),
			BankName:      b.GetBankName(),
			CountryCode:   b.GetCountryCode(),
		},
	})
	if err != nil {
		return nil, mapError(err)
	}
	return payoutOperationToProto(result), nil
}

func (s *Server) VerifyPayout(
	ctx context.Context,
	req *paymentv1.VerifyPayoutRequest,
) (*paymentv1.PayoutOperationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	result, err := s.useCases.VerifyPayout.Execute(ctx, req.GetPayoutId())
	if err != nil {
		return nil, mapError(err)
	}
	return payoutOperationToProto(result), nil
}

func (s *Server) GetPayout(
	ctx context.Context,
	req *paymentv1.GetPayoutRequest,
) (*paymentv1.PayoutDetails, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	result, err := s.useCases.GetPayout.Execute(ctx, req.GetPayoutId(), req.GetStoreId())
	if err != nil {
		return nil, mapError(err)
	}
	return payoutToProto(result), nil
}

func (s *Server) ListPayouts(
	ctx context.Context,
	req *paymentv1.ListPayoutsRequest,
) (*paymentv1.ListPayoutsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	result, err := s.useCases.ListPayouts.Execute(
		ctx,
		req.GetStoreId(),
		int(req.GetLimit()),
		int(req.GetOffset()),
	)
	if err != nil {
		return nil, mapError(err)
	}

	response := &paymentv1.ListPayoutsResponse{
		Total:  result.Total,
		Limit:  int32(result.Limit),
		Offset: int32(result.Offset),
	}
	for _, payout := range result.Payouts {
		response.Payouts = append(response.Payouts, payoutToProto(payout))
	}
	return response, nil
}

func (s *Server) RetryPayout(
	ctx context.Context,
	req *paymentv1.RetryPayoutRequest,
) (*paymentv1.PayoutOperationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	result, err := s.useCases.RetryPayout.Execute(ctx, req.GetPayoutId(), req.GetStoreId())
	if err != nil {
		return nil, mapError(err)
	}
	return payoutOperationToProto(result), nil
}

func (s *Server) HandleWebhook(
	ctx context.Context,
	req *paymentv1.HandleWebhookRequest,
) (*paymentv1.HandleWebhookResponse, error) {
	if req == nil || len(req.GetBody()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "webhook body is required")
	}
	result, err := s.useCases.HandleWebhook.Execute(ctx, req.GetBody(), req.GetHeaders())
	if err != nil {
		return nil, mapError(err)
	}
	return &paymentv1.HandleWebhookResponse{
		EventId:      result.EventID,
		EventType:    result.EventType,
		ResourceType: result.ResourceType,
		ResourceId:   result.ResourceID,
	}, nil
}

func paymentDetailsToProto(details *use_cases.PaymentDetails) *paymentv1.PaymentDetails {
	if details == nil {
		return nil
	}
	result := &paymentv1.PaymentDetails{
		Id:                    details.ID,
		Reference:             details.Reference,
		PayerId:               details.PayerID,
		Purpose:               string(details.Purpose),
		BusinessReferenceType: details.BusinessReferenceType,
		BusinessReferenceId:   details.BusinessReferenceID,
		AmountMinor:           details.Amount,
		Currency:              details.Currency,
		Status:                string(details.Status),
	}
	if details.Attempt != nil {
		attempt := details.Attempt
		result.Attempt = &paymentv1.PaymentAttemptDetails{
			Id:                attempt.ID,
			PaymentId:         attempt.PaymentID,
			Provider:          attempt.Provider,
			ProviderReference: attempt.ProviderReference,
			Status:            string(attempt.Status),
			FailureReason:     attempt.FailureReason,
			CheckoutUrl:       attempt.CheckoutURL,
			MetadataJson:      json.RawMessage(attempt.Metadata),
		}
		switch expiry := attempt.ExpiresAt.(type) {
		case time.Time:
			result.Attempt.ExpiresAt = timestamppb.New(expiry)
		case *time.Time:
			if expiry != nil {
				result.Attempt.ExpiresAt = timestamppb.New(*expiry)
			}
		}
	}
	return result
}

func payoutOperationToProto(result *use_cases.InitiatePayoutOutput) *paymentv1.PayoutOperationResponse {
	if result == nil {
		return nil
	}
	return &paymentv1.PayoutOperationResponse{
		PayoutId:      result.PayoutID,
		AttemptId:     result.AttemptID,
		Reference:     result.Reference,
		Status:        string(result.Status),
		AttemptStatus: string(result.AttemptStatus),
		Provider:      result.Provider,
		AmountMinor:   result.Amount,
		Currency:      result.Currency,
	}
}

func payoutToProto(payout *entities.Payout) *paymentv1.PayoutDetails {
	if payout == nil {
		return nil
	}
	reference := payout.BusinessReference()
	amount := payout.Amount()
	return &paymentv1.PayoutDetails{
		Id:                    payout.ID(),
		Reference:             payout.Reference(),
		BusinessReferenceType: reference.Type(),
		BusinessReferenceId:   reference.ID(),
		Reason:                payout.Reason(),
		StoreId:               payout.StoreID(),
		OrderId:               payout.OrderID(),
		AmountMinor:           amount.Amount(),
		Currency:              amount.Currency(),
		Status:                string(payout.Status()),
		CreatedAt:             timestamppb.New(payout.CreatedAt()),
		UpdatedAt:             timestamppb.New(payout.UpdatedAt()),
	}
}

func mapError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, application.ErrPaymentNotFound),
		errors.Is(err, application.ErrPaymentAttemptNotFound),
		errors.Is(err, application.ErrPayoutNotFound),
		errors.Is(err, application.ErrPayoutAttemptNotFound):
		return status.Error(codes.NotFound, err.Error())

	case errors.Is(err, application.ErrPaymentAlreadyExists),
		errors.Is(err, application.ErrPayoutAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())

	case errors.Is(err, application.ErrIdempotencyConflict),
		errors.Is(err, application.ErrPayoutIdempotencyConflict):
		return status.Error(codes.Aborted, err.Error())

	case errors.Is(err, application.ErrAttemptRequiresReconciliation),
		errors.Is(err, application.ErrPayoutRetryNotAllowed):
		return status.Error(codes.FailedPrecondition, err.Error())

	case errors.Is(err, application.ErrInvalidPagination),
		errors.Is(err, domainerrors.ErrInvalidPayment),
		errors.Is(err, domainerrors.ErrInvalidPaymentTransition),
		errors.Is(err, domainerrors.ErrInvalidPaymentAttempt),
		errors.Is(err, domainerrors.ErrInvalidPaymentAttemptTransition),
		errors.Is(err, domainerrors.ErrInvalidMoney),
		errors.Is(err, domainerrors.ErrInvalidBusinessReference),
		errors.Is(err, domainerrors.ErrInvalidPayout),
		errors.Is(err, domainerrors.ErrInvalidPayoutTransition),
		errors.Is(err, domainerrors.ErrInvalidPayoutAttempt),
		errors.Is(err, domainerrors.ErrInvalidPayoutAttemptTransition):
		return status.Error(codes.InvalidArgument, err.Error())

	case errors.Is(err, application.ErrInvalidProviderResponse):
		return status.Error(codes.Unavailable, err.Error())

	default:
		_ = fmt.Sprintf("%v", err)
		return status.Error(codes.Internal, "payment operation failed")
	}
}
