package producer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
)

const TaskCompletedTopic = "task.events.completed"

// TaskCompletionStatus represents the final state of a user task.
type TaskCompletionStatus string

const (
	TaskCompletionStatusCompleted TaskCompletionStatus = "completed"
	TaskCompletionStatusExpired   TaskCompletionStatus = "expired"
)

// TaskCompletedEvent is the payload for task completion messages.
type TaskCompletedEvent struct {
	TaskID              int                  `json:"task_id"`
	UserID              int                  `json:"user_id"`
	Status              TaskCompletionStatus `json:"status"`
	GroupID             int                  `json:"group_id"`
	CompletedTaskCount  int                  `json:"completed_task_count"`
	TotalTaskCount      int                  `json:"total_task_count"`
}

// TaskCompleteProducer publishes task completion events.
type TaskCompleteProducer interface {
	PublishTaskCompleted(ctx context.Context, event TaskCompletedEvent) error
}

type taskCompleteProducerImpl struct {
	producer KafkaProducer
	topic    string
}

var (
	taskCompleteProducerOnce sync.Once
	taskCompleteProducerInst TaskCompleteProducer
)

// GetTaskCompleteProducer returns the singleton task completion producer.
func GetTaskCompleteProducer() TaskCompleteProducer {
	taskCompleteProducerOnce.Do(func() {
		taskCompleteProducerInst = &taskCompleteProducerImpl{
			producer: GetKafkaProducer(),
			topic:    TaskCompletedTopic,
		}
	})
	return taskCompleteProducerInst
}

func (p *taskCompleteProducerImpl) PublishTaskCompleted(ctx context.Context, event TaskCompletedEvent) error {
	if err := validateTaskCompletedEvent(event); err != nil {
		return err
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal task completed event: %w", err)
	}

	return p.producer.Publish(ctx, p.topic, []byte(strconv.Itoa(event.UserID)), payload)
}

func validateTaskCompletedEvent(event TaskCompletedEvent) error {
	if event.TaskID <= 0 {
		return errors.New("task_id must be positive")
	}
	if event.UserID <= 0 {
		return errors.New("user_id must be positive")
	}
	if event.GroupID <= 0 {
		return errors.New("group_id must be positive")
	}
	if event.CompletedTaskCount < 0 {
		return errors.New("completed_task_count must be non-negative")
	}
	if event.TotalTaskCount < 0 {
		return errors.New("total_task_count must be non-negative")
	}
	switch event.Status {
	case TaskCompletionStatusCompleted, TaskCompletionStatusExpired:
		return nil
	default:
		return fmt.Errorf("invalid task completion status: %q", event.Status)
	}
}
