package listener

import (
	"context"

	"github.com/nusiss-capstone-project/task-mservice/server/kafka"
)

const (
	TopicUserKycCompleteEvents         = "user.events.kyc_complete"
	TopicPaymentMethodAddedEvents      = "payment.payment_method.added"
	TopicAssetOrderPaymentResultEvents = "asset.order.payment_result"
	TopicDepositOrderPaymentResultEvents = "deposit.order.payment_result"
)

func init() {
	kafka.RegisterHandler(TopicUserKycCompleteEvents, handleUserKycCompleteEvent)
	kafka.RegisterHandler(TopicPaymentMethodAddedEvents, handlePaymentMethodAddedEvent)
	kafka.RegisterHandler(TopicAssetOrderPaymentResultEvents, handleAssetOrderPaymentResultEvent)
	kafka.RegisterHandler(TopicDepositOrderPaymentResultEvents, handleDepositOrderPaymentResultEvent)
}

type KafkaHandler func(ctx context.Context, msg *kafka.Message) error
