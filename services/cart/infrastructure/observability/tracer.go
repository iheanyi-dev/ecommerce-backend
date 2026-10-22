package observability

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/cart/application/ports"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Tracer struct {
	tracer trace.Tracer
}

func NewTracer() ports.Tracer {
	return &Tracer{
		tracer: otel.Tracer("github.com/iheanyi-dev/ecommerce-backend/services/cart"),
	}
}

func (t *Tracer) Start(ctx context.Context, name string) (context.Context, ports.Span) {
	ctx, span := t.tracer.Start(ctx, name)
	return ctx, &spanAdapter{span: span}
}

type spanAdapter struct {
	span trace.Span
}

func (s *spanAdapter) SetAttribute(key, value string) {
	s.span.SetAttributes(attribute.String(key, value))
}

func (s *spanAdapter) SetIntAttribute(key string, value int) {
	s.span.SetAttributes(attribute.Int(key, value))
}

func (s *spanAdapter) RecordError(err error) {
	if err != nil {
		s.span.RecordError(err)
	}
}

func (s *spanAdapter) SetErrorStatus(description string) {
	s.span.SetStatus(codes.Error, description)
}

func (s *spanAdapter) End() {
	s.span.End()
}
