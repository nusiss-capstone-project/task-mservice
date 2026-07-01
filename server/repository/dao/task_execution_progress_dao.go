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

type TaskExecutionProgressDao interface {
	Create(ctx context.Context, progress *model.TaskExecutionProgress) (int, error)
	Update(ctx context.Context, progress *model.TaskExecutionProgress) error
	GetByID(ctx context.Context, id int) (*model.TaskExecutionProgress, error)
	UpdateStatusIfIn(ctx context.Context, id int, newStatus string, fromStatuses []string) (bool, error)
	EnrollUserTask(ctx context.Context, userID, taskID int, conditions []model.TaskCondition) (int, []int, error)
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
		log.Logger.Errorf("failed to create task execution progress: %v", ret.Error)
		return 0, ret.Error
	}
	log.Logger.Infof("task execution progress created with ID: %d", progress.ID)
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
		log.Logger.Errorf("failed to update task execution progress %d: %v", progress.ID, ret.Error)
		return ret.Error
	}
	if ret.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	log.Logger.Infof("task execution progress %d updated", progress.ID)
	return nil
}

func (d *TaskExecutionProgressDaoImpl) GetByID(ctx context.Context, id int) (*model.TaskExecutionProgress, error) {
	var progress model.TaskExecutionProgress
	ret := d.db.WithContext(ctx).Where("id = ?", id).First(&progress)
	if ret.Error != nil {
		if errors.Is(ret.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.Logger.Errorf("failed to get task execution progress %d: %v", id, ret.Error)
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
		log.Logger.Errorf("failed to conditionally update task execution progress %d: %v", id, ret.Error)
		return false, ret.Error
	}
	if ret.RowsAffected > 0 {
		log.Logger.Infof("task execution progress %d status updated to %s", id, newStatus)
	}
	return ret.RowsAffected > 0, nil
}

func (d *TaskExecutionProgressDaoImpl) EnrollUserTask(
	ctx context.Context,
	userID, taskID int,
	conditions []model.TaskCondition,
) (int, []int, error) {
	var executionID int
	conditionProgressIDs := make([]int, 0, len(conditions))

	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		execution := &model.TaskExecutionProgress{
			TaskID: taskID,
			UserID: userID,
			Status: model.TaskExecutionProgressStatusInProgress,
		}
		if err := tx.Create(execution).Error; err != nil {
			log.Logger.Errorf("failed to create task execution progress for user %d task %d: %v", userID, taskID, err)
			return err
		}
		log.Logger.Infof("task execution progress created with ID: %d", execution.ID)
		executionID = execution.ID

		for _, condition := range conditions {
			conditionProgress := &model.TaskConditionExecutionProgress{
				UserID:                  userID,
				TaskExecutionProgressID: execution.ID,
				TaskID:                  taskID,
				TaskConditionID:         condition.ID,
				CurrentValue:            "",
				Status:                  model.TaskConditionExecutionProgressStatusInProgress,
			}
			if err := tx.Create(conditionProgress).Error; err != nil {
				log.Logger.Errorf("failed to create condition progress for user %d task %d condition %d: %v",
					userID, taskID, condition.ID, err)
				return err
			}
			log.Logger.Infof("task condition execution progress created with ID: %d", conditionProgress.ID)
			conditionProgressIDs = append(conditionProgressIDs, conditionProgress.ID)
		}
		return nil
	})
	if err != nil {
		return 0, nil, err
	}
	return executionID, conditionProgressIDs, nil
}
