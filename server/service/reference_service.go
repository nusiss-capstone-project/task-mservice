package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/nusiss-capstone-project/task-mservice/server/http/data"
	"github.com/nusiss-capstone-project/task-mservice/server/repository/dao"
)

type ReferenceService interface {
	ListDataMetrics(ctx context.Context) ([]data.DataMetricVO, error)
	ListMetricOperators(ctx context.Context) ([]data.MetricOperatorVO, error)
}

type ReferenceServiceImpl struct {
	dataMetricDao     dao.DataMetricDao
	metricOperatorDao dao.MetricOperatorDao
}

var (
	referenceServiceOnce sync.Once
	referenceServiceInst *ReferenceServiceImpl
)

func GetReferenceService() *ReferenceServiceImpl {
	referenceServiceOnce.Do(func() {
		referenceServiceInst = &ReferenceServiceImpl{
			dataMetricDao:     dao.GetDataMetricDao(),
			metricOperatorDao: dao.GetMetricOperatorDao(),
		}
	})
	return referenceServiceInst
}

func (s *ReferenceServiceImpl) ListDataMetrics(ctx context.Context) ([]data.DataMetricVO, error) {
	metrics, err := s.dataMetricDao.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list data metrics: %w", err)
	}
	result := make([]data.DataMetricVO, 0, len(metrics))
	for _, metric := range metrics {
		result = append(result, data.DataMetricVO{
			ID:   metric.ID,
			Code: metric.Code,
		})
	}
	return result, nil
}

func (s *ReferenceServiceImpl) ListMetricOperators(ctx context.Context) ([]data.MetricOperatorVO, error) {
	operators, err := s.metricOperatorDao.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list metric operators: %w", err)
	}
	result := make([]data.MetricOperatorVO, 0, len(operators))
	for _, op := range operators {
		result = append(result, data.MetricOperatorVO{
			ID:      op.ID,
			Code:    op.Code,
			Display: op.Display,
		})
	}
	return result, nil
}
