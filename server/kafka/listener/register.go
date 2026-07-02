package listener

import (
	"context"

	"github.com/nusiss-capstone-project/task-mservice/server/kafka"
)

const (
	TopicDepositEvents         = "deposit.events"
	TopicUserRegisteredEvents  = "user.events.registered"
	TopicUserKycCompleteEvents = "user.events.kyc_complete"
)

func init() {
	kafka.RegisterHandler(TopicDepositEvents, handleDepositEvent)
	kafka.RegisterHandler(TopicUserRegisteredEvents, handleUserEvent)
}

type KafkaHandler func(ctx context.Context, msg *kafka.Message) error
