package handlers

import (
	"context"

	"github.com/nusiss-capstone-project/task-mservice/server/kafka"
	"github.com/nusiss-capstone-project/task-mservice/server/log"
)

// handleDepositEvent is a stub handler for deposit.events.
func handleDepositEvent(ctx context.Context, msg *kafka.Message) error {
	log.WithContext(ctx).Infow("deposit event stub handler",
		"topic", msg.Topic,
		"offset", msg.Offset,
	)
	return nil
}
