package log

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestWithContextExtractsTraceIDs(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)

	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	defer span.End()

	l := WithContext(ctx)
	if l == Logger {
		t.Fatal("expected logger with trace fields")
	}
	traceID, spanID := traceIDs(ctx)
	if traceID == "" || spanID == "" {
		t.Fatal("expected valid trace context in test")
	}
}
