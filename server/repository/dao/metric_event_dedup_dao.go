package dao

import (
	"context"
	"errors"
	"sync"

	"github.com/nusiss-capstone-project/task-mservice/server/log"
	"github.com/nusiss-capstone-project/task-mservice/server/repository"
	"github.com/nusiss-capstone-project/task-mservice/server/repository/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MetricEventDedupDao interface {
	// TryClaim inserts (metricCode, bizID). Returns false when the pair already exists.
	TryClaim(ctx context.Context, metricCode, bizID string) (bool, error)
	// Release deletes a claim so a failed update can be retried.
	Release(ctx context.Context, metricCode, bizID string) error
}

type metricEventDedupDaoImpl struct {
	db *gorm.DB
}

var (
	metricEventDedupOnce sync.Once
	metricEventDedupDao  MetricEventDedupDao
)

func GetMetricEventDedupDao() MetricEventDedupDao {
	metricEventDedupOnce.Do(func() {
		metricEventDedupDao = &metricEventDedupDaoImpl{db: repository.DB}
	})
	return metricEventDedupDao
}

func (d *metricEventDedupDaoImpl) TryClaim(ctx context.Context, metricCode, bizID string) (bool, error) {
	if metricCode == "" || bizID == "" {
		return false, errors.New("metric_code and biz_id are required")
	}
	row := model.MetricEventDedup{MetricCode: metricCode, BizID: bizID}
	ret := d.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
	if ret.Error != nil {
		log.WithContext(ctx).Errorf("claim metric event %s/%s: %v", metricCode, bizID, ret.Error)
		return false, ret.Error
	}
	return ret.RowsAffected > 0, nil
}

func (d *metricEventDedupDaoImpl) Release(ctx context.Context, metricCode, bizID string) error {
	if metricCode == "" || bizID == "" {
		return errors.New("metric_code and biz_id are required")
	}
	ret := d.db.WithContext(ctx).
		Where("metric_code = ? AND biz_id = ?", metricCode, bizID).
		Delete(&model.MetricEventDedup{})
	if ret.Error != nil {
		log.WithContext(ctx).Errorf("release metric event %s/%s: %v", metricCode, bizID, ret.Error)
		return ret.Error
	}
	return nil
}
