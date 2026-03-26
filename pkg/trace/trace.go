package trace

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	Enabled     bool
	ServiceName string
	Endpoint    string
	SampleRate  float64
}

type TracerProvider struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
}

func NewTracerProvider(cfg *Config) (*TracerProvider, error) {
	if !cfg.Enabled {
		return &TracerProvider{}, nil
	}

	var exporter sdktrace.SpanExporter
	var err error

	if cfg.Endpoint == "" {
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
	} else if cfg.Endpoint == "jaeger" {
		exporter, err = jaeger.New(jaeger.WithAgentEndpoint())
	} else {
		exporter, err = otlptracegrpc.New(context.Background(),
			otlptracegrpc.WithEndpoint(cfg.Endpoint),
			otlptracegrpc.WithInsecure(),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	sampleRate := cfg.SampleRate
	if sampleRate <= 0 || sampleRate > 1 {
		sampleRate = 1.0
	}

	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion("v1.0.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(sampleRate))),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &TracerProvider{
		provider: tp,
		tracer:   tp.Tracer(cfg.ServiceName),
	}, nil
}

func (tp *TracerProvider) Tracer() trace.Tracer {
	if tp.tracer == nil {
		return otel.Tracer("default")
	}
	return tp.tracer
}

func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	if tp.provider != nil {
		return tp.provider.Shutdown(ctx)
	}
	return nil
}

func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return otel.Tracer("").Start(ctx, name, opts...)
}

func AddSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

func RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
}

func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

func NewSpanContext() trace.SpanContext {
	return trace.NewSpanContext(trace.SpanContextConfig{})
}

type TraceID string

func GetTraceID(ctx context.Context) TraceID {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasTraceID() {
		return TraceID(span.SpanContext().TraceID().String())
	}
	return ""
}

type Span interface {
	End(opts ...trace.SpanEndOption)
	RecordError(err error)
	AddEvent(name string, opts ...trace.EventOption)
	SetAttributes(attrs ...attribute.KeyValue)
	SetStatus(code codes.Code, description string)
	Tracer() trace.Tracer
}

const (
	CodeUnset = codes.Unset
	CodeOk    = codes.Ok
	CodeError = codes.Error
)

type wrappedSpan struct {
	span trace.Span
}

func (s *wrappedSpan) End(opts ...trace.SpanEndOption) {
	s.span.End(opts...)
}

func (s *wrappedSpan) RecordError(err error) {
	s.span.RecordError(err)
}

func (s *wrappedSpan) AddEvent(name string, opts ...trace.EventOption) {
	s.span.AddEvent(name, opts...)
}

func (s *wrappedSpan) SetAttributes(attrs ...attribute.KeyValue) {
	s.span.SetAttributes(attrs...)
}

func (s *wrappedSpan) SetStatus(code codes.Code, description string) {
	s.span.SetStatus(code, description)
}

func (s *wrappedSpan) Tracer() trace.Tracer {
	return s.span.TracerProvider().Tracer("")
}

func WrapSpan(span trace.Span) Span {
	return &wrappedSpan{span: span}
}
