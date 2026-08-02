package mocks

import (
	"context"

	"github.com/nusiss-capstone-project/task-mservice/server/repository/model"
	"github.com/stretchr/testify/mock"
)

type TaskExecutionProgressDao struct {
	mock.Mock
}

func (m *TaskExecutionProgressDao) Create(ctx context.Context, progress *model.TaskExecutionProgress) (int, error) {
	args := m.Called(ctx, progress)
	return args.Int(0), args.Error(1)
}

func (m *TaskExecutionProgressDao) Update(ctx context.Context, progress *model.TaskExecutionProgress) error {
	args := m.Called(ctx, progress)
	return args.Error(0)
}

func (m *TaskExecutionProgressDao) GetByID(ctx context.Context, id int) (*model.TaskExecutionProgress, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.TaskExecutionProgress), args.Error(1)
}

func (m *TaskExecutionProgressDao) UpdateStatusIfIn(ctx context.Context, id int, newStatus string, fromStatuses []string) (bool, error) {
	args := m.Called(ctx, id, newStatus, fromStatuses)
	return args.Bool(0), args.Error(1)
}

func (m *TaskExecutionProgressDao) EnrollUserTask(ctx context.Context, userID, taskID int, conditions []model.TaskCondition) (int, []int, error) {
	args := m.Called(ctx, userID, taskID, conditions)
	if args.Get(1) == nil {
		return args.Int(0), nil, args.Error(2)
	}
	return args.Int(0), args.Get(1).([]int), args.Error(2)
}
