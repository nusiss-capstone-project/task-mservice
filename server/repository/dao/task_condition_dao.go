package dao

import (
	"context"
	"sync"

	"github.com/nusiss-capstone-project/task-mservice/server/log"
	"github.com/nusiss-capstone-project/task-mservice/server/repository"
	"github.com/nusiss-capstone-project/task-mservice/server/repository/model"
	"gorm.io/gorm"
)

type TaskConditionDao interface {
	ReplaceByTaskID(ctx context.Context, taskID int, conditions []model.TaskCondition) error
	ListByTaskID(ctx context.Context, taskID int) ([]model.TaskCondition, error)
}

type TaskConditionDaoImpl struct {
	db *gorm.DB
}

var (
	taskConditionOnce sync.Once
	taskConditionDao  *TaskConditionDaoImpl
)

func GetTaskConditionDao() *TaskConditionDaoImpl {
	taskConditionOnce.Do(func() {
		taskConditionDao = &TaskConditionDaoImpl{db: repository.DB}
	})
	return taskConditionDao
}

func (d *TaskConditionDaoImpl) ReplaceByTaskID(ctx context.Context, taskID int, conditions []model.TaskCondition) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("task_id = ?", taskID).Delete(&model.TaskCondition{}).Error; err != nil {
			log.Logger.Errorf("failed to delete conditions for task %d: %v", taskID, err)
			return err
		}
		if len(conditions) == 0 {
			log.Logger.Infof("conditions cleared for task %d", taskID)
			return nil
		}
		if err := tx.Create(&conditions).Error; err != nil {
			log.Logger.Errorf("failed to create conditions for task %d: %v", taskID, err)
			return err
		}
		log.Logger.Infof("conditions replaced for task %d, count=%d", taskID, len(conditions))
		return nil
	})
}

func (d *TaskConditionDaoImpl) ListByTaskID(ctx context.Context, taskID int) ([]model.TaskCondition, error) {
	var conditions []model.TaskCondition
	ret := d.db.WithContext(ctx).Where("task_id = ?", taskID).Order("no ASC").Find(&conditions)
	if ret.Error != nil {
		log.Logger.Errorf("failed to list conditions for task %d: %v", taskID, ret.Error)
		return nil, ret.Error
	}
	return conditions, nil
}
