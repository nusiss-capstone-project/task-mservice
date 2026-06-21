package mocks

import (
	"context"

	"github.com/nusiss-capstone-project/task-mservice/server/repository/model"
	"github.com/stretchr/testify/mock"
)

type TaskConditionDao struct {
	mock.Mock
}

func (m *TaskConditionDao) ReplaceByTaskID(ctx context.Context, taskID int, conditions []model.TaskCondition) error {
	args := m.Called(ctx, taskID, conditions)
	return args.Error(0)
}

func (m *TaskConditionDao) ListByTaskID(ctx context.Context, taskID int) ([]model.TaskCondition, error) {
	args := m.Called(ctx, taskID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.TaskCondition), args.Error(1)
}
