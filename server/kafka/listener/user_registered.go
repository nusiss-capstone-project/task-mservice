package listener

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nusiss-capstone-project/task-mservice/server/kafka"
	"github.com/nusiss-capstone-project/task-mservice/server/log"
	"github.com/nusiss-capstone-project/task-mservice/server/repository/dao"
	"github.com/nusiss-capstone-project/task-mservice/server/service"
)

const metricCode = "user_registered"

// handleUserEvent is a stub handler for user.events.
func handleUserEvent(ctx context.Context, msg *kafka.Message) error {
	log.WithContext(ctx).Infow("user event stub handler",
		"topic", msg.Topic,
		"offset", msg.Offset,
	)
	var event userRegisteredEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal user registered event: %w", err)
	}
	if err := event.Validate(); err != nil {
		return fmt.Errorf("invalid user registered event: %w", err)
	}
	metric, err := dao.GetDataMetricDao().GetByCode(ctx, metricCode)
	if err != nil {
		return fmt.Errorf("get data metric %s: %w", metricCode, err)
	}
	if metric == nil {
		return fmt.Errorf("data metric not found: %s", metricCode)
	}
	if err := service.GetUserTaskProgressService().UpdateUserTaskProgress(
		ctx, event.UserID, metric.ID, "true", time.Unix(int64(event.RegisterTime), 0),
	); err != nil {
		return fmt.Errorf("update user task progress: %w", err)
	}
	return nil
}

type userRegisteredEvent struct {
	UserID       int       `json:"user_id"`
	RegisterTime int       `json:"register_time"`
	Channel      string    `json:"channel"`
	InviterID    int       `json:"inviter_id"`
	EventTime    time.Time `json:"event_time"`
}

func (e *userRegisteredEvent) Validate() error {
	if e.UserID <= 0 {
		return errors.New("user_id is required")
	}
	return nil
}
