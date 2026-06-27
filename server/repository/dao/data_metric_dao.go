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

const dataMetricCacheRefreshInterval = time.Minute

type DataMetricDao interface {
	List(ctx context.Context) ([]model.DataMetric, error)
	GetByCode(ctx context.Context, code string) (*model.DataMetric, error)
}

type dataMetricCache struct {
	mu     sync.RWMutex
	list   []model.DataMetric
	byID   map[int]model.DataMetric
	byCode map[string]model.DataMetric
}

type DataMetricDaoImpl struct {
	db    *gorm.DB
	cache dataMetricCache
}

var (
	dataMetricOnce sync.Once
	dataMetricDao  *DataMetricDaoImpl
)

func GetDataMetricDao() *DataMetricDaoImpl {
	dataMetricOnce.Do(func() {
		impl := &DataMetricDaoImpl{db: repository.DB}
		if repository.DB != nil {
			if err := impl.refreshCache(context.Background()); err != nil {
				log.Logger.Errorf("initial data metric cache load failed: %v", err)
			}
			impl.startRefreshLoop()
		}
		dataMetricDao = impl
		log.Logger.Infof("data metric dao initialized")
	})
	return dataMetricDao
}

func (d *DataMetricDaoImpl) startRefreshLoop() {
	ticker := time.NewTicker(dataMetricCacheRefreshInterval)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			if err := d.refreshCache(context.Background()); err != nil {
				log.Logger.Errorf("refresh data metric cache failed: %v", err)
			}
		}
	}()
}

func (d *DataMetricDaoImpl) refreshCache(ctx context.Context) error {
	metrics, err := d.loadFromDB(ctx)
	if err != nil {
		return err
	}
	byID := make(map[int]model.DataMetric, len(metrics))
	byCode := make(map[string]model.DataMetric, len(metrics))
	for _, metric := range metrics {
		byID[metric.ID] = metric
		byCode[metric.Code] = metric
	}
	d.cache.mu.Lock()
	d.cache.list = metrics
	d.cache.byID = byID
	d.cache.byCode = byCode
	d.cache.mu.Unlock()
	log.Logger.Infof("data metric cache refreshed, count=%d", len(metrics))
	return nil
}

func (d *DataMetricDaoImpl) loadFromDB(ctx context.Context) ([]model.DataMetric, error) {
	var metrics []model.DataMetric
	ret := d.db.WithContext(ctx).Order("id ASC").Find(&metrics)
	if ret.Error != nil {
		log.Logger.Errorf("failed to list data metrics from db: %v", ret.Error)
		return nil, ret.Error
	}
	return metrics, nil
}

func (d *DataMetricDaoImpl) List(ctx context.Context) ([]model.DataMetric, error) {
	_ = ctx
	d.cache.mu.RLock()
	defer d.cache.mu.RUnlock()
	if len(d.cache.list) == 0 {
		return []model.DataMetric{}, nil
	}
	result := make([]model.DataMetric, len(d.cache.list))
	copy(result, d.cache.list)
	return result, nil
}

func (d *DataMetricDaoImpl) GetByCode(ctx context.Context, code string) (*model.DataMetric, error) {
	_ = ctx
	if code == "" {
		return nil, nil
	}
	d.cache.mu.RLock()
	defer d.cache.mu.RUnlock()
	metric, ok := d.cache.byCode[code]
	if !ok {
		return nil, nil
	}
	copy := metric
	return &copy, nil
}

var _ DataMetricDao = (*DataMetricDaoImpl)(nil)
