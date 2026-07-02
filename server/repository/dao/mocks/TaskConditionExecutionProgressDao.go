package mocks

import (
	"context"
	"time"

	"github.com/nusiss-capstone-project/task-mservice/server/repository/model"
	"github.com/stretchr/testify/mock"
)

type TaskConditionExecutionProgressDao struct {
	mock.Mock
}

func (m *TaskConditionExecutionProgressDao) Create(ctx context.Context, progress *model.TaskConditionExecutionProgress) (int, error) {
	args := m.Called(ctx, progress)
	return args.Int(0), args.Error(1)
}

func (m *TaskConditionExecutionProgressDao) Update(ctx context.Context, progress *model.TaskConditionExecutionProgress) error {
	args := m.Called(ctx, progress)
	return args.Error(0)
}

func (m *TaskConditionExecutionProgressDao) ListInProgressByUserAndMetric(ctx context.Context, userID, metricID int) ([]model.TaskConditionExecutionProgress, error) {
	args := m.Called(ctx, userID, metricID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.TaskConditionExecutionProgress), args.Error(1)
}

func (m *TaskConditionExecutionProgressDao) ListByTaskExecutionProgressID(ctx context.Context, taskExecutionProgressID int) ([]model.TaskConditionExecutionProgress, error) {
	args := m.Called(ctx, taskExecutionProgressID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.TaskConditionExecutionProgress), args.Error(1)
}

func (m *TaskConditionExecutionProgressDao) UpdateIfStatusIn(ctx context.Context, id int, currentValue, newStatus string, eventTime time.Time, fromStatuses []string) (bool, error) {
	args := m.Called(ctx, id, currentValue, newStatus, eventTime, fromStatuses)
	return args.Bool(0), args.Error(1)
}
