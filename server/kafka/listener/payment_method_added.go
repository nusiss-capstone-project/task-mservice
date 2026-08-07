package listener

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/nusiss-capstone-project/task-mservice/server/kafka"
	"github.com/nusiss-capstone-project/task-mservice/server/log"
)

type paymentMethodAddedEvent struct {
	UserID    int64  `json:"user_id"`
	EventType string `json:"event_type"`
	EventTime int64  `json:"event_time"`
	Provider  string `json:"provider"`
}

func (e *paymentMethodAddedEvent) Validate() error {
	if e.UserID <= 0 {
		return errors.New("user_id is required")
	}
	if e.EventTime <= 0 {
		return errors.New("event_time is required")
	}
	return nil
}

func handlePaymentMethodAddedEvent(ctx context.Context, msg *kafka.Message) error {
	log.WithContext(ctx).Infow("payment method added event",
		"topic", msg.Topic,
		"offset", msg.Offset,
	)
	var event paymentMethodAddedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("unmarshal payment method added event: %w", err)
	}
	if err := event.Validate(); err != nil {
		return fmt.Errorf("invalid payment method added event: %w", err)
	}
	return applyMetricEvent(
		ctx,
		int(event.UserID),
		metricPaymentMethodAdded,
		"true",
		eventTimeFromUnixSeconds(event.EventTime),
		"",
	)
}
