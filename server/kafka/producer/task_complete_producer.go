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
	TaskID int                  `json:"task_id"`
	UserID int                  `json:"user_id"`
	Status TaskCompletionStatus `json:"status"`
}

// TaskCompleteProducer publishes task completion events.
type TaskCompleteProducer interface {
	PublishTaskCompleted(ctx context.Context, taskID, userID int, status TaskCompletionStatus) error
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

func (p *taskCompleteProducerImpl) PublishTaskCompleted(
	ctx context.Context,
	taskID, userID int,
	status TaskCompletionStatus,
) error {
	if err := validateTaskCompletedInput(taskID, userID, status); err != nil {
		return err
	}

	event := TaskCompletedEvent{
		TaskID: taskID,
		UserID: userID,
		Status: status,
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal task completed event: %w", err)
	}

	return p.producer.Publish(ctx, p.topic, []byte(strconv.Itoa(userID)), payload)
}

func validateTaskCompletedInput(taskID, userID int, status TaskCompletionStatus) error {
	if taskID <= 0 {
		return errors.New("task_id must be positive")
	}
	if userID <= 0 {
		return errors.New("user_id must be positive")
	}
	switch status {
	case TaskCompletionStatusCompleted, TaskCompletionStatusExpired:
		return nil
	default:
		return fmt.Errorf("invalid task completion status: %q", status)
	}
}
