package trace_test

import (
	"context"
	"testing"

	"github.com/nusiss-capstone-project/task-mservice/server/kafka/trace"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestHeadersFromContextAndExtract(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(tp)

	ctx, span := tp.Tracer("test").Start(context.Background(), "parent")
	defer span.End()

	headers := trace.HeadersFromContext(ctx)
	if len(headers) == 0 {
		t.Fatal("expected trace headers")
	}

	extracted := trace.ContextFromHeaders(context.Background(), headers)
	traceID, spanID := trace.IDs(extracted)
	if traceID == "" || spanID == "" {
		t.Fatalf("expected trace ids, got trace_id=%q span_id=%q", traceID, spanID)
	}
}

func TestStartConsume(t *testing.T) {
	ctx, span := trace.StartConsume(context.Background(), &kgo.Record{
		Topic:     "deposit.events",
		Partition: 1,
		Offset:    10,
	})
	trace.Finish(span, nil)
	traceID, spanID := trace.IDs(ctx)
	_ = traceID
	_ = spanID
}
