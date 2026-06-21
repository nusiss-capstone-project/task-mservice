package dao

import (
	"context"
	"sync"

	"github.com/nusiss-capstone-project/task-mservice/server/log"
	"github.com/nusiss-capstone-project/task-mservice/server/repository"
	"github.com/nusiss-capstone-project/task-mservice/server/repository/model"
	"gorm.io/gorm"
)

type DataMetricDao interface {
	List(ctx context.Context) ([]model.DataMetric, error)
}

type DataMetricDaoImpl struct {
	db *gorm.DB
}

var (
	dataMetricOnce sync.Once
	dataMetricDao  *DataMetricDaoImpl
)

func GetDataMetricDao() *DataMetricDaoImpl {
	dataMetricOnce.Do(func() {
		dataMetricDao = &DataMetricDaoImpl{db: repository.DB}
	})
	return dataMetricDao
}

func (d *DataMetricDaoImpl) List(ctx context.Context) ([]model.DataMetric, error) {
	var metrics []model.DataMetric
	ret := d.db.WithContext(ctx).Order("id ASC").Find(&metrics)
	if ret.Error != nil {
		log.Logger.Errorf("failed to list data metrics: %v", ret.Error)
		return nil, ret.Error
	}
	return metrics, nil
}
