package kafka_test

import (
	"context"
	"testing"

	"github.com/nusiss-capstone-project/task-mservice/server/kafka"
)

func TestRegisterHandlerMultiple(t *testing.T) {
	topic := "test.topic.multi"
	kafka.RegisterHandler(topic, func(ctx context.Context, msg *kafka.Message) error {
		return nil
	})
	kafka.RegisterHandler(topic, func(ctx context.Context, msg *kafka.Message) error {
		return nil
	})

	handlers := kafka.HandlersForTopic(topic)
	if len(handlers) != 2 {
		t.Fatalf("handler count = %d, want 2", len(handlers))
	}
}
