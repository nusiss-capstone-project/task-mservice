package listener

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/nusiss-capstone-project/task-mservice/server/kafka"
	"github.com/nusiss-capstone-project/task-mservice/server/log"
)

// orderPaymentResultEvent covers asset.order.payment_result (trade metrics).
type orderPaymentResultEvent struct {
	UserID        int64  `json:"user_id"`
	OrderID       int64  `json:"order_id"`
	OrderNo       string `json:"order_no"`
	PaymentID     string `json:"payment_id"`
	AssetID       int64  `json:"asset_id"`
	AssetSymbol   string `json:"asset_symbol"`
	Quantity      string `json:"quantity"`
	UnitPrice     string `json:"unit_price"`
	PaymentAmount string `json:"payment_amount"`
	PayCurrency   string `json:"pay_currency"`
	Status        string `json:"status"`
	EventTime     int64  `json:"event_time"`
}

func (e *orderPaymentResultEvent) Validate() error {
	if e.UserID <= 0 {
		return errors.New("user_id is required")
	}
	if e.OrderID <= 0 {
		return errors.New("order_id is required")
	}
	if e.EventTime <= 0 {
		return errors.New("event_time is required")
	}
	return nil
}

func handleAssetOrderPaymentResultEvent(ctx context.Context, msg *kafka.Message) error {
	log.WithContext(ctx).Infow("asset order payment result event",
		"topic", msg.Topic,
		"offset", msg.Offset,
	)
	var event orderPaymentResultEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("unmarshal order payment result event: %w", err)
	}
	if err := event.Validate(); err != nil {
		return fmt.Errorf("invalid order payment result event: %w", err)
	}
	if event.Status != orderPaymentStatusOK {
		log.WithContext(ctx).Infow("skip order payment event, status not pay_succeed",
			"user_id", event.UserID,
			"order_id", event.OrderID,
			"status", event.Status,
		)
		return nil
	}
	if event.PaymentAmount == "" {
		return errors.New("payment_amount is required")
	}

	eventTime := eventTimeFromUnixSeconds(event.EventTime)
	bizID := orderBizID(event.OrderID)
	userID := int(event.UserID)

	updates := []struct {
		code  string
		value string
	}{
		{metricTradeCountAccum, "1"},
		{metricTradeAmount, event.PaymentAmount},
		{metricTradeAmountAccum, event.PaymentAmount},
	}
	for _, u := range updates {
		if err := applyMetricEvent(ctx, userID, u.code, u.value, eventTime, bizID); err != nil {
			return err
		}
	}
	return nil
}
