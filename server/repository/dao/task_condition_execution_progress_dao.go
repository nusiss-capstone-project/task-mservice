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

type TaskConditionExecutionProgressDao interface {
	Create(ctx context.Context, progress *model.TaskConditionExecutionProgress) (int, error)
	Update(ctx context.Context, progress *model.TaskConditionExecutionProgress) error
	ListInProgressByUserAndMetric(ctx context.Context, userID, metricID int) ([]model.TaskConditionExecutionProgress, error)
	ListByTaskExecutionProgressID(ctx context.Context, taskExecutionProgressID int) ([]model.TaskConditionExecutionProgress, error)
	UpdateIfStatusIn(ctx context.Context, id int, currentValue, newStatus string, eventTime time.Time, fromStatuses []string) (bool, error)
}

type taskConditionExecutionProgressDaoImpl struct {
	db *gorm.DB
}

var (
	taskConditionExecutionProgressOnce sync.Once
	taskConditionExecutionProgressDao  TaskConditionExecutionProgressDao
)

func GetTaskConditionExecutionProgressDao() TaskConditionExecutionProgressDao {
	taskConditionExecutionProgressOnce.Do(func() {
		taskConditionExecutionProgressDao = &taskConditionExecutionProgressDaoImpl{db: repository.DB}
	})
	return taskConditionExecutionProgressDao
}

func (d *taskConditionExecutionProgressDaoImpl) Create(ctx context.Context, progress *model.TaskConditionExecutionProgress) (int, error) {
	ret := d.db.WithContext(ctx).Create(progress)
	if ret.Error != nil {
		log.WithContext(ctx).Errorf("failed to create task condition execution progress: %v", ret.Error)
		return 0, ret.Error
	}
	log.WithContext(ctx).Infof("task condition execution progress created with ID: %d", progress.ID)
	return progress.ID, nil
}

func (d *taskConditionExecutionProgressDaoImpl) Update(ctx context.Context, progress *model.TaskConditionExecutionProgress) error {
	ret := d.db.WithContext(ctx).Model(&model.TaskConditionExecutionProgress{}).
		Where("id = ?", progress.ID).
		Updates(map[string]interface{}{
			"user_id":                     progress.UserID,
			"task_execution_progress_id":  progress.TaskExecutionProgressID,
			"task_id":                     progress.TaskID,
			"task_condition_id":           progress.TaskConditionID,
			"current_value":               progress.CurrentValue,
			"status":                      progress.Status,
			"last_event_time":             progress.LastEventTime,
		})
	if ret.Error != nil {
		log.WithContext(ctx).Errorf("failed to update task condition execution progress %d: %v", progress.ID, ret.Error)
		return ret.Error
	}
	if ret.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	log.WithContext(ctx).Infof("task condition execution progress %d updated", progress.ID)
	return nil
}

func (d *taskConditionExecutionProgressDaoImpl) ListInProgressByUserAndMetric(
	ctx context.Context,
	userID, metricID int,
) ([]model.TaskConditionExecutionProgress, error) {
	var progresses []model.TaskConditionExecutionProgress
	ret := d.db.WithContext(ctx).
		Joins("JOIN task_condition ON task_condition.id = task_condition_execution_progress.task_condition_id").
		Where("task_condition_execution_progress.user_id = ?", userID).
		Where("task_condition.data_metric_id = ?", metricID).
		Where("task_condition_execution_progress.status = ?", model.TaskConditionExecutionProgressStatusInProgress).
		Find(&progresses)
	if ret.Error != nil {
		log.WithContext(ctx).Errorf("failed to list in-progress condition progress for user %d metric %d: %v", userID, metricID, ret.Error)
		return nil, ret.Error
	}
	return progresses, nil
}

func (d *taskConditionExecutionProgressDaoImpl) ListByTaskExecutionProgressID(
	ctx context.Context,
	taskExecutionProgressID int,
) ([]model.TaskConditionExecutionProgress, error) {
	var progresses []model.TaskConditionExecutionProgress
	ret := d.db.WithContext(ctx).
		Where("task_execution_progress_id = ?", taskExecutionProgressID).
		Order("id ASC").
		Find(&progresses)
	if ret.Error != nil {
		log.WithContext(ctx).Errorf("failed to list condition progress for execution %d: %v", taskExecutionProgressID, ret.Error)
		return nil, ret.Error
	}
	return progresses, nil
}

func (d *taskConditionExecutionProgressDaoImpl) UpdateIfStatusIn(
	ctx context.Context,
	id int,
	currentValue, newStatus string,
	eventTime time.Time,
	fromStatuses []string,
) (bool, error) {
	updates := map[string]interface{}{
		"current_value":   currentValue,
		"last_event_time": eventTime,
	}
	if newStatus != "" {
		updates["status"] = newStatus
	}
	ret := d.db.WithContext(ctx).
		Model(&model.TaskConditionExecutionProgress{}).
		Where("id = ? AND status IN ? AND (last_event_time IS NULL OR last_event_time <= ?)", id, fromStatuses, eventTime).
		Updates(updates)
	if ret.Error != nil {
		log.WithContext(ctx).Errorf("failed to conditionally update condition progress %d: %v", id, ret.Error)
		return false, ret.Error
	}
	if ret.RowsAffected > 0 {
		log.WithContext(ctx).Infof("task condition execution progress %d updated", id)
	}
	return ret.RowsAffected > 0, nil
}
