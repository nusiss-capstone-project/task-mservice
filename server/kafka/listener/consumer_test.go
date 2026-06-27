package consumer

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nusiss-capstone-project/task-mservice/server/config"
	"github.com/nusiss-capstone-project/task-mservice/server/kafka"
)

func TestValidateConsumerConfig(t *testing.T) {
	if err := validateConfig(nil); err == nil {
		t.Fatal("expected error for nil config")
	}
	if err := validateConfig(&config.KafkaConfig{}); err == nil {
		t.Fatal("expected error for empty config")
	}
}

func TestInvokeHandlersParallel(t *testing.T) {
	var count int32
	msg := &kafka.Message{Topic: "parallel.topic", Offset: 1}
	handlers := []kafka.Handler{
		func(ctx context.Context, msg *kafka.Message) error {
			atomic.AddInt32(&count, 1)
			time.Sleep(20 * time.Millisecond)
			return nil
		},
		func(ctx context.Context, msg *kafka.Message) error {
			atomic.AddInt32(&count, 1)
			return nil
		},
	}

	start := time.Now()
	if err := invokeHandlersParallel(context.Background(), handlers, msg); err != nil {
		t.Fatalf("invokeHandlersParallel() error = %v", err)
	}
	if atomic.LoadInt32(&count) != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
	if elapsed := time.Since(start); elapsed < 15*time.Millisecond {
		t.Fatalf("expected parallel execution, elapsed = %v", elapsed)
	}
}

func TestInvokeHandlersParallelError(t *testing.T) {
	handlers := []kafka.Handler{
		func(ctx context.Context, msg *kafka.Message) error { return nil },
		func(ctx context.Context, msg *kafka.Message) error { return errors.New("handler failed") },
	}
	err := invokeHandlersParallel(context.Background(), handlers, &kafka.Message{Topic: "err.topic"})
	if err == nil {
		t.Fatal("expected error")
	}
}
