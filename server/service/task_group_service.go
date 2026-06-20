package service

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/nusiss-capstone-project/task-mservice/server/http/data"
	"github.com/nusiss-capstone-project/task-mservice/server/log"
	"github.com/nusiss-capstone-project/task-mservice/server/repository/dao"
	"github.com/nusiss-capstone-project/task-mservice/server/repository/model"
	"gorm.io/gorm"
)

type TaskGroupService interface {
	SaveTaskGroup(ctx context.Context, vo *data.TaskGroupVO) (*data.TaskGroupVO, error)
	ListTaskGroups(ctx context.Context) ([]data.TaskGroupVO, error)
	PublishTaskGroup(ctx context.Context, id int) (*data.PublishStatusVO, error)
}

type TaskGroupServiceImpl struct {
	taskGroupDao dao.TaskGroupDao
}

var (
	taskGroupServiceOnce sync.Once
	taskGroupServiceInst *TaskGroupServiceImpl
)

func GetTaskGroupService() *TaskGroupServiceImpl {
	taskGroupServiceOnce.Do(func() {
		taskGroupServiceInst = &TaskGroupServiceImpl{taskGroupDao: dao.GetTaskGroupDao()}
	})
	return taskGroupServiceInst
}

func (s *TaskGroupServiceImpl) SaveTaskGroup(ctx context.Context, vo *data.TaskGroupVO) (*data.TaskGroupVO, error) {
	if vo == nil {
		return nil, errors.New("task group is nil")
	}
	if vo.Name == "" {
		return nil, errors.New("task group name is required")
	}

	if vo.ID > 0 {
		existing, err := s.taskGroupDao.GetByID(ctx, vo.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get task group: %w", err)
		}
		if existing == nil {
			return nil, errors.New("task group not found")
		}
		if existing.Status == model.StatusPublished {
			return nil, errors.New("published task group cannot be modified")
		}
		existing.Name = vo.Name
		id, err := s.taskGroupDao.Save(ctx, existing)
		if err != nil {
			return nil, fmt.Errorf("failed to update task group: %w", err)
		}
		log.Logger.Infof("task group updated, id=%d", id)
		return toTaskGroupVO(existing), nil
	}

	group := &model.TaskGroup{
		Name:   vo.Name,
		Status: model.StatusDraft,
	}
	id, err := s.taskGroupDao.Save(ctx, group)
	if err != nil {
		return nil, fmt.Errorf("failed to create task group: %w", err)
	}
	group.ID = id
	log.Logger.Infof("task group created, id=%d", id)
	return toTaskGroupVO(group), nil
}

func (s *TaskGroupServiceImpl) ListTaskGroups(ctx context.Context) ([]data.TaskGroupVO, error) {
	groups, err := s.taskGroupDao.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list task groups: %w", err)
	}
	result := make([]data.TaskGroupVO, 0, len(groups))
	for i := range groups {
		result = append(result, *toTaskGroupVO(&groups[i]))
	}
	return result, nil
}

func (s *TaskGroupServiceImpl) PublishTaskGroup(ctx context.Context, id int) (*data.PublishStatusVO, error) {
	group, err := s.taskGroupDao.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get task group: %w", err)
	}
	if group == nil {
		return nil, errors.New("task group not found")
	}
	if group.Status == model.StatusPublished {
		return &data.PublishStatusVO{ID: id, Status: model.StatusPublished}, nil
	}
	if err := s.taskGroupDao.UpdateStatus(ctx, id, model.StatusPublished); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("task group not found")
		}
		return nil, fmt.Errorf("failed to publish task group: %w", err)
	}
	log.Logger.Infof("task group published, id=%d", id)
	return &data.PublishStatusVO{ID: id, Status: model.StatusPublished}, nil
}

func toTaskGroupVO(group *model.TaskGroup) *data.TaskGroupVO {
	return &data.TaskGroupVO{
		ID:     group.ID,
		Name:   group.Name,
		Status: group.Status,
	}
}
