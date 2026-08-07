package listener

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/nusiss-capstone-project/task-mservice/server/kafka"
)

func TestHandleUserKycCompleteEvent_SkipNonPassed(t *testing.T) {
	body, _ := json.Marshal(userKYCCompleteEvent{
		UserID: 1, KYCStatus: "PENDING", EventTime: time.Now().UTC(),
	})
	err := handleUserKycCompleteEvent(context.Background(), &kafka.Message{Topic: TopicUserKycCompleteEvents, Value: body})
	if err != nil {
		t.Fatalf("expected skip without error, got %v", err)
	}
}

func TestHandleAssetOrderPaymentResultEvent_SkipNonSuccess(t *testing.T) {
	body, _ := json.Marshal(orderPaymentResultEvent{
		UserID: 1, OrderID: 2, Status: "failed", PaymentAmount: "10", EventTime: 1710000000,
	})
	err := handleAssetOrderPaymentResultEvent(context.Background(), &kafka.Message{
		Topic: TopicAssetOrderPaymentResultEvents, Value: body,
	})
	if err != nil {
		t.Fatalf("expected skip without error, got %v", err)
	}
}

func TestOrderPaymentResultEventValidate(t *testing.T) {
	e := orderPaymentResultEvent{}
	if err := e.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}
