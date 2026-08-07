package mocks

import (
	"context"

	"github.com/nusiss-capstone-project/task-mservice/server/kafka/producer"
	"github.com/stretchr/testify/mock"
)

type TaskCompleteProducer struct {
	mock.Mock
}

func (m *TaskCompleteProducer) PublishTaskCompleted(ctx context.Context, event producer.TaskCompletedEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}
