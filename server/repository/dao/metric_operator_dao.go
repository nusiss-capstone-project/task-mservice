package dao

import (
	"context"
	"sync"

	"github.com/nusiss-capstone-project/task-mservice/server/log"
	"github.com/nusiss-capstone-project/task-mservice/server/repository"
	"github.com/nusiss-capstone-project/task-mservice/server/repository/model"
	"gorm.io/gorm"
)

type MetricOperatorDao interface {
	List(ctx context.Context) ([]model.MetricOperator, error)
}

type MetricOperatorDaoImpl struct {
	db *gorm.DB
}

var (
	metricOperatorOnce sync.Once
	metricOperatorDao  *MetricOperatorDaoImpl
)

func GetMetricOperatorDao() *MetricOperatorDaoImpl {
	metricOperatorOnce.Do(func() {
		metricOperatorDao = &MetricOperatorDaoImpl{db: repository.DB}
	})
	return metricOperatorDao
}

func (d *MetricOperatorDaoImpl) List(ctx context.Context) ([]model.MetricOperator, error) {
	var operators []model.MetricOperator
	ret := d.db.WithContext(ctx).Order("id ASC").Find(&operators)
	if ret.Error != nil {
		log.Logger.Errorf("failed to list metric operators: %v", ret.Error)
		return nil, ret.Error
	}
	return operators, nil
}
