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

type TaskGroupDao interface {
	Save(ctx context.Context, group *model.TaskGroup) (int, error)
	GetByID(ctx context.Context, id int) (*model.TaskGroup, error)
	List(ctx context.Context) ([]model.TaskGroup, error)
	UpdateStatus(ctx context.Context, id int, status string) error
}

type TaskGroupDaoImpl struct {
	db *gorm.DB
}

var (
	taskGroupOnce sync.Once
	taskGroupDao  *TaskGroupDaoImpl
)

func GetTaskGroupDao() *TaskGroupDaoImpl {
	taskGroupOnce.Do(func() {
		taskGroupDao = &TaskGroupDaoImpl{db: repository.DB}
	})
	return taskGroupDao
}

func (d *TaskGroupDaoImpl) Save(ctx context.Context, group *model.TaskGroup) (int, error) {
	ret := d.db.WithContext(ctx).Save(group)
	if ret.Error != nil {
		log.Logger.Errorf("failed to save task group: %v", ret.Error)
		return 0, ret.Error
	}
	log.Logger.Infof("task group saved with ID: %d", group.ID)
	return group.ID, nil
}

func (d *TaskGroupDaoImpl) GetByID(ctx context.Context, id int) (*model.TaskGroup, error) {
	var group model.TaskGroup
	ret := d.db.WithContext(ctx).Where("id = ?", id).First(&group)
	if ret.Error != nil {
		if errors.Is(ret.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.Logger.Errorf("failed to get task group by id %d: %v", id, ret.Error)
		return nil, ret.Error
	}
	return &group, nil
}

func (d *TaskGroupDaoImpl) List(ctx context.Context) ([]model.TaskGroup, error) {
	var groups []model.TaskGroup
	ret := d.db.WithContext(ctx).Order("id DESC").Find(&groups)
	if ret.Error != nil {
		log.Logger.Errorf("failed to list task groups: %v", ret.Error)
		return nil, ret.Error
	}
	return groups, nil
}

func (d *TaskGroupDaoImpl) UpdateStatus(ctx context.Context, id int, status string) error {
	ret := d.db.WithContext(ctx).Model(&model.TaskGroup{}).Where("id = ?", id).Update("status", status)
	if ret.Error != nil {
		log.Logger.Errorf("failed to update task group %d status: %v", id, ret.Error)
		return ret.Error
	}
	if ret.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	log.Logger.Infof("task group %d status updated to %s", id, status)
	return nil
}
