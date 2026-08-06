package dao

import (
	"context"
	"errors"
	"sync"

	"github.com/nusiss-capstone-project/task-mservice/server/log"
	"github.com/nusiss-capstone-project/task-mservice/server/repository"
	"github.com/nusiss-capstone-project/task-mservice/server/repository/model"
	"gorm.io/gorm"
)

// EnrollProgressItem is one task enrollment: execution progress + its condition progresses.
// Service constructs models; DAO fills IDs after insert.
type EnrollProgressItem struct {
	Execution  *model.TaskExecutionProgress
	Conditions []model.TaskConditionExecutionProgress
}

type TaskExecutionProgressDao interface {
	Create(ctx context.Context, progress *model.TaskExecutionProgress) (int, error)
	Update(ctx context.Context, progress *model.TaskExecutionProgress) error
	GetByID(ctx context.Context, id int) (*model.TaskExecutionProgress, error)
	UpdateStatusIfIn(ctx context.Context, id int, newStatus string, fromStatuses []string) (bool, error)
	EnrollUserTasks(ctx context.Context, items []EnrollProgressItem) error
	CountByUserGroupAndStatus(ctx context.Context, userID, groupID int, status string) (int, error)
}

type TaskExecutionProgressDaoImpl struct {
	db *gorm.DB
}

var (
	taskExecutionProgressOnce sync.Once
	taskExecutionProgressDao  TaskExecutionProgressDao
)

func GetTaskExecutionProgressDao() TaskExecutionProgressDao {
	taskExecutionProgressOnce.Do(func() {
		taskExecutionProgressDao = &TaskExecutionProgressDaoImpl{db: repository.DB}
	})
	return taskExecutionProgressDao
}

func (d *TaskExecutionProgressDaoImpl) Create(ctx context.Context, progress *model.TaskExecutionProgress) (int, error) {
	ret := d.db.WithContext(ctx).Create(progress)
	if ret.Error != nil {
		log.WithContext(ctx).Errorf("failed to create task execution progress: %v", ret.Error)
		return 0, ret.Error
	}
	log.WithContext(ctx).Infof("task execution progress created with ID: %d", progress.ID)
	return progress.ID, nil
}

func (d *TaskExecutionProgressDaoImpl) Update(ctx context.Context, progress *model.TaskExecutionProgress) error {
	ret := d.db.WithContext(ctx).Model(&model.TaskExecutionProgress{}).
		Where("id = ?", progress.ID).
		Updates(map[string]interface{}{
			"task_id": progress.TaskID,
			"user_id": progress.UserID,
			"status":  progress.Status,
		})
	if ret.Error != nil {
		log.WithContext(ctx).Errorf("failed to update task execution progress %d: %v", progress.ID, ret.Error)
		return ret.Error
	}
	if ret.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	log.WithContext(ctx).Infof("task execution progress %d updated", progress.ID)
	return nil
}

func (d *TaskExecutionProgressDaoImpl) GetByID(ctx context.Context, id int) (*model.TaskExecutionProgress, error) {
	var progress model.TaskExecutionProgress
	ret := d.db.WithContext(ctx).Where("id = ?", id).First(&progress)
	if ret.Error != nil {
		if errors.Is(ret.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.WithContext(ctx).Errorf("failed to get task execution progress %d: %v", id, ret.Error)
		return nil, ret.Error
	}
	return &progress, nil
}

func (d *TaskExecutionProgressDaoImpl) UpdateStatusIfIn(
	ctx context.Context,
	id int,
	newStatus string,
	fromStatuses []string,
) (bool, error) {
	ret := d.db.WithContext(ctx).
		Model(&model.TaskExecutionProgress{}).
		Where("id = ? AND status IN ?", id, fromStatuses).
		Update("status", newStatus)
	if ret.Error != nil {
		log.WithContext(ctx).Errorf("failed to conditionally update task execution progress %d: %v", id, ret.Error)
		return false, ret.Error
	}
	if ret.RowsAffected > 0 {
		log.WithContext(ctx).Infof("task execution progress %d status updated to %s", id, newStatus)
	}
	return ret.RowsAffected > 0, nil
}

func (d *TaskExecutionProgressDaoImpl) EnrollUserTasks(ctx context.Context, items []EnrollProgressItem) error {
	if len(items) == 0 {
		return errors.New("enroll items is empty")
	}
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := range items {
			item := &items[i]
			if item.Execution == nil {
				return errors.New("execution progress is nil")
			}
			if err := tx.Create(item.Execution).Error; err != nil {
				log.WithContext(ctx).Errorf("failed to create task execution progress for user %d task %d: %v",
					item.Execution.UserID, item.Execution.TaskID, err)
				return err
			}
			log.WithContext(ctx).Infof("task execution progress created with ID: %d", item.Execution.ID)

			for j := range item.Conditions {
				cond := &item.Conditions[j]
				cond.TaskExecutionProgressID = item.Execution.ID
				if err := tx.Create(cond).Error; err != nil {
					log.WithContext(ctx).Errorf("failed to create condition progress for user %d task %d condition %d: %v",
						cond.UserID, cond.TaskID, cond.TaskConditionID, err)
					return err
				}
				log.WithContext(ctx).Infof("task condition execution progress created with ID: %d", cond.ID)
			}
		}
		return nil
	})
}

func (d *TaskExecutionProgressDaoImpl) CountByUserGroupAndStatus(
	ctx context.Context,
	userID, groupID int,
	status string,
) (int, error) {
	var count int64
	ret := d.db.WithContext(ctx).Model(&model.TaskExecutionProgress{}).
		Joins("JOIN task ON task.id = task_execution_progress.task_id").
		Where("task_execution_progress.user_id = ?", userID).
		Where("task.task_group_id = ?", groupID).
		Where("task_execution_progress.status = ?", status).
		Count(&count)
	if ret.Error != nil {
		log.WithContext(ctx).Errorf(
			"failed to count execution progress user=%d group=%d status=%s: %v",
			userID, groupID, status, ret.Error,
		)
		return 0, ret.Error
	}
	return int(count), nil
}
