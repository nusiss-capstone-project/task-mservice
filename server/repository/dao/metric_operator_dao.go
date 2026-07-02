package dao

import (
	"context"
	"sync"
	"time"

	"github.com/nusiss-capstone-project/task-mservice/server/log"
	"github.com/nusiss-capstone-project/task-mservice/server/repository"
	"github.com/nusiss-capstone-project/task-mservice/server/repository/model"
	"gorm.io/gorm"
)

const metricOperatorCacheRefreshInterval = time.Minute

type MetricOperatorDao interface {
	List(ctx context.Context) ([]model.MetricOperator, error)
	GetByID(ctx context.Context, id int) (*model.MetricOperator, error)
}

type metricOperatorCache struct {
	mu   sync.RWMutex
	list []model.MetricOperator
	byID map[int]model.MetricOperator
}

type MetricOperatorDaoImpl struct {
	db    *gorm.DB
	cache metricOperatorCache
}

var (
	metricOperatorOnce sync.Once
	metricOperatorDao  *MetricOperatorDaoImpl
)

func GetMetricOperatorDao() *MetricOperatorDaoImpl {
	metricOperatorOnce.Do(func() {
		impl := &MetricOperatorDaoImpl{db: repository.DB}
		if repository.DB != nil {
			if err := impl.refreshCache(context.Background()); err != nil {
				log.Logger.Errorf("initial metric operator cache load failed: %v", err)
			}
			impl.startRefreshLoop()
		}
		metricOperatorDao = impl
		log.Logger.Infof("metric operator dao initialized")
	})
	return metricOperatorDao
}

func (d *MetricOperatorDaoImpl) startRefreshLoop() {
	ticker := time.NewTicker(metricOperatorCacheRefreshInterval)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			if err := d.refreshCache(context.Background()); err != nil {
				log.Logger.Errorf("refresh metric operator cache failed: %v", err)
			}
		}
	}()
}

func (d *MetricOperatorDaoImpl) refreshCache(ctx context.Context) error {
	operators, err := d.loadFromDB(ctx)
	if err != nil {
		return err
	}
	byID := make(map[int]model.MetricOperator, len(operators))
	for _, operator := range operators {
		byID[operator.ID] = operator
	}
	d.cache.mu.Lock()
	d.cache.list = operators
	d.cache.byID = byID
	d.cache.mu.Unlock()
	log.Logger.Infof("metric operator cache refreshed, count=%d", len(operators))
	return nil
}

func (d *MetricOperatorDaoImpl) loadFromDB(ctx context.Context) ([]model.MetricOperator, error) {
	var operators []model.MetricOperator
	ret := d.db.WithContext(ctx).Order("id ASC").Find(&operators)
	if ret.Error != nil {
		log.Logger.Errorf("failed to list metric operators from db: %v", ret.Error)
		return nil, ret.Error
	}
	return operators, nil
}

func (d *MetricOperatorDaoImpl) List(ctx context.Context) ([]model.MetricOperator, error) {
	_ = ctx
	d.cache.mu.RLock()
	defer d.cache.mu.RUnlock()
	if len(d.cache.list) == 0 {
		return []model.MetricOperator{}, nil
	}
	result := make([]model.MetricOperator, len(d.cache.list))
	copy(result, d.cache.list)
	return result, nil
}

func (d *MetricOperatorDaoImpl) GetByID(ctx context.Context, id int) (*model.MetricOperator, error) {
	_ = ctx
	d.cache.mu.RLock()
	defer d.cache.mu.RUnlock()
	operator, ok := d.cache.byID[id]
	if !ok {
		return nil, nil
	}
	copy := operator
	return &copy, nil
}

// Ensure MetricOperatorDaoImpl satisfies interface at compile time.
var _ MetricOperatorDao = (*MetricOperatorDaoImpl)(nil)
