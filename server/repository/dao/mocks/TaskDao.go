package mocks

import (
	"context"

	"github.com/nusiss-capstone-project/task-mservice/server/repository/model"
	"github.com/stretchr/testify/mock"
)

type TaskDao struct {
	mock.Mock
}

func (m *TaskDao) Save(ctx context.Context, task *model.Task) (int, error) {
	args := m.Called(ctx, task)
	return args.Int(0), args.Error(1)
}

func (m *TaskDao) GetByID(ctx context.Context, id int) (*model.Task, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Task), args.Error(1)
}

func (m *TaskDao) ListByGroupID(ctx context.Context, groupID int) ([]model.Task, error) {
	args := m.Called(ctx, groupID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Task), args.Error(1)
}

func (m *TaskDao) UpdateStatus(ctx context.Context, id int, status string) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}
