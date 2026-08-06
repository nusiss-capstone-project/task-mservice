package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type MetricEventDedupDao struct {
	mock.Mock
}

func (m *MetricEventDedupDao) TryClaim(ctx context.Context, metricCode, bizID string) (bool, error) {
	args := m.Called(ctx, metricCode, bizID)
	return args.Bool(0), args.Error(1)
}

func (m *MetricEventDedupDao) Release(ctx context.Context, metricCode, bizID string) error {
	args := m.Called(ctx, metricCode, bizID)
	return args.Error(0)
}
