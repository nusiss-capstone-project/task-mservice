package mocks

import (
	"context"

	"github.com/nusiss-capstone-project/task-mservice/server/repository/model"
	"github.com/stretchr/testify/mock"
)

type TaskGroupDao struct {
	mock.Mock
}

func (m *TaskGroupDao) Save(ctx context.Context, group *model.TaskGroup) (int, error) {
	args := m.Called(ctx, group)
	return args.Int(0), args.Error(1)
}

func (m *TaskGroupDao) GetByID(ctx context.Context, id int) (*model.TaskGroup, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.TaskGroup), args.Error(1)
}

func (m *TaskGroupDao) List(ctx context.Context) ([]model.TaskGroup, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.TaskGroup), args.Error(1)
}

func (m *TaskGroupDao) UpdateStatus(ctx context.Context, id int, status string) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}
