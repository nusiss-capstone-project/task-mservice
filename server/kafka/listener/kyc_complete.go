package listener

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nusiss-capstone-project/task-mservice/server/kafka"
	"github.com/nusiss-capstone-project/task-mservice/server/log"
)

type userKYCCompleteEvent struct {
	UserID       int       `json:"user_id"`
	KYCStatus    string    `json:"kyc_status"`
	KYCUpdatedAt time.Time `json:"kyc_updated_at"`
	EventTime    time.Time `json:"event_time"`
}

func (e *userKYCCompleteEvent) Validate() error {
	if e.UserID <= 0 {
		return errors.New("user_id is required")
	}
	if e.KYCStatus == "" {
		return errors.New("kyc_status is required")
	}
	return nil
}

func (e *userKYCCompleteEvent) resolveEventTime() time.Time {
	if !e.EventTime.IsZero() {
		return e.EventTime.UTC()
	}
	return e.KYCUpdatedAt.UTC()
}

func handleUserKycCompleteEvent(ctx context.Context, msg *kafka.Message) error {
	log.WithContext(ctx).Infow("kyc complete event",
		"topic", msg.Topic,
		"offset", msg.Offset,
	)
	var event userKYCCompleteEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("unmarshal kyc complete event: %w", err)
	}
	if err := event.Validate(); err != nil {
		return fmt.Errorf("invalid kyc complete event: %w", err)
	}
	if event.KYCStatus != kycStatusPassed {
		log.WithContext(ctx).Infow("skip kyc event, status not PASSED",
			"user_id", event.UserID,
			"kyc_status", event.KYCStatus,
		)
		return nil
	}
	eventTime := event.resolveEventTime()
	if eventTime.IsZero() {
		return errors.New("event_time is required")
	}
	return applyMetricEvent(ctx, event.UserID, metricKycPassed, "true", eventTime, "")
}
