package mocks

import (
	"context"

	"github.com/nusiss-capstone-project/task-mservice/server/repository/model"
	"github.com/stretchr/testify/mock"
)

type DataMetricDao struct {
	mock.Mock
}

func (m *DataMetricDao) List(ctx context.Context) ([]model.DataMetric, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.DataMetric), args.Error(1)
}

func (m *DataMetricDao) GetByCode(ctx context.Context, code string) (*model.DataMetric, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.DataMetric), args.Error(1)
}
