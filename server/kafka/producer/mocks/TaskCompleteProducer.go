package mocks

import (
	"context"

	"github.com/nusiss-capstone-project/task-mservice/server/kafka/producer"
	"github.com/stretchr/testify/mock"
)

type TaskCompleteProducer struct {
	mock.Mock
}

func (m *TaskCompleteProducer) PublishTaskCompleted(ctx context.Context, taskID, userID int, status producer.TaskCompletionStatus) error {
	args := m.Called(ctx, taskID, userID, status)
	return args.Error(0)
}
