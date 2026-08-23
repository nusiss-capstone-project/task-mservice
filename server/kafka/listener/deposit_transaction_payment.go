package listener

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/nusiss-capstone-project/task-mservice/server/kafka"
	"github.com/nusiss-capstone-project/task-mservice/server/log"
)

// depositOrderPaymentResultEvent is a provisional schema (order_id/status/currency/amount).
type depositPaymentResultEvent struct {
	UserID        int64  `json:"user_id"`
	TransactionID int64  `json:"transaction_id"`
	Status        string `json:"status"`
	Currency      string `json:"currency"`
	Amount        string `json:"amount"`
	EventTime     int64  `json:"event_time"`
}

func (e *depositPaymentResultEvent) Validate() error {
	if e.UserID <= 0 {
		return errors.New("user_id is required")
	}
	if e.TransactionID <= 0 {
		return errors.New("order_id is required")
	}
	if e.EventTime <= 0 {
		return errors.New("event_time is required")
	}
	return nil
}

func handleDepositOrderPaymentResultEvent(ctx context.Context, msg *kafka.Message) error {
	log.WithContext(ctx).Infow("deposit order payment result event",
		"topic", msg.Topic,
		"offset", msg.Offset,
	)
	var event depositPaymentResultEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("unmarshal deposit order payment result event: %w", err)
	}
	if err := event.Validate(); err != nil {
		return fmt.Errorf("invalid deposit order payment result event: %w", err)
	}
	if event.Status != transactionPaymentStatusOK {
		log.WithContext(ctx).Infow("skip deposit payment event, status not SUCCEEDED",
			"user_id", event.UserID,
			"transaction_id", event.TransactionID,
			"status", event.Status,
		)
		return nil
	}
	if event.Amount == "" {
		return errors.New("amount is required")
	}

	eventTime := eventTimeFromUnixSeconds(event.EventTime)
	bizID := orderBizID(event.TransactionID)
	userID := int(event.UserID)

	updates := []struct {
		code  string
		value string
	}{
		{metricDepositAmount, event.Amount},
		{metricDepositAmountAccum, event.Amount},
	}
	for _, u := range updates {
		if err := applyMetricEvent(ctx, userID, u.code, u.value, eventTime, bizID); err != nil {
			return err
		}
	}
	return nil
}
