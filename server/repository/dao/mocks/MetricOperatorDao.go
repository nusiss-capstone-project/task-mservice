package mocks

import (
	"context"

	"github.com/nusiss-capstone-project/task-mservice/server/repository/model"
	"github.com/stretchr/testify/mock"
)

type MetricOperatorDao struct {
	mock.Mock
}

func (m *MetricOperatorDao) List(ctx context.Context) ([]model.MetricOperator, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.MetricOperator), args.Error(1)
}
