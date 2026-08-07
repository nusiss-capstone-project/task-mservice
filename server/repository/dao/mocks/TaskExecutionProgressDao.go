package mocks

import (
	"context"

	"github.com/nusiss-capstone-project/task-mservice/server/repository/dao"
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

func (m *TaskExecutionProgressDao) EnrollUserTasks(ctx context.Context, items []dao.EnrollProgressItem) error {
	args := m.Called(ctx, items)
	return args.Error(0)
}

func (m *TaskExecutionProgressDao) CountByUserGroupAndStatus(ctx context.Context, userID, groupID int, status string) (int, error) {
	args := m.Called(ctx, userID, groupID, status)
	return args.Int(0), args.Error(1)
}

func (m *TaskExecutionProgressDao) ListByUserAndGroupID(ctx context.Context, userID, groupID int) ([]model.TaskExecutionProgress, error) {
	args := m.Called(ctx, userID, groupID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.TaskExecutionProgress), args.Error(1)
}
