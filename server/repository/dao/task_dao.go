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

type TaskDao interface {
	Save(ctx context.Context, task *model.Task) (int, error)
	GetByID(ctx context.Context, id int) (*model.Task, error)
	ListByGroupID(ctx context.Context, groupID int) ([]model.Task, error)
	UpdateStatus(ctx context.Context, id int, status string) error
}

type TaskDaoImpl struct {
	db *gorm.DB
}

var (
	taskOnce sync.Once
	taskDao  *TaskDaoImpl
)

func GetTaskDao() *TaskDaoImpl {
	taskOnce.Do(func() {
		taskDao = &TaskDaoImpl{db: repository.DB}
	})
	return taskDao
}

func (d *TaskDaoImpl) Save(ctx context.Context, task *model.Task) (int, error) {
	ret := d.db.WithContext(ctx).Save(task)
	if ret.Error != nil {
		log.Logger.Errorf("failed to save task: %v", ret.Error)
		return 0, ret.Error
	}
	log.Logger.Infof("task saved with ID: %d", task.ID)
	return task.ID, nil
}

func (d *TaskDaoImpl) GetByID(ctx context.Context, id int) (*model.Task, error) {
	var task model.Task
	ret := d.db.WithContext(ctx).Where("id = ?", id).First(&task)
	if ret.Error != nil {
		if errors.Is(ret.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.Logger.Errorf("failed to get task by id %d: %v", id, ret.Error)
		return nil, ret.Error
	}
	return &task, nil
}

func (d *TaskDaoImpl) ListByGroupID(ctx context.Context, groupID int) ([]model.Task, error) {
	var tasks []model.Task
	ret := d.db.WithContext(ctx).Where("task_group_id = ?", groupID).Order("id ASC").Find(&tasks)
	if ret.Error != nil {
		log.Logger.Errorf("failed to list tasks for group %d: %v", groupID, ret.Error)
		return nil, ret.Error
	}
	return tasks, nil
}

func (d *TaskDaoImpl) UpdateStatus(ctx context.Context, id int, status string) error {
	ret := d.db.WithContext(ctx).Model(&model.Task{}).Where("id = ?", id).Update("status", status)
	if ret.Error != nil {
		log.Logger.Errorf("failed to update task %d status: %v", id, ret.Error)
		return ret.Error
	}
	if ret.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	log.Logger.Infof("task %d status updated to %s", id, status)
	return nil
}
