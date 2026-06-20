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

type TaskService interface {
	CreateTask(ctx context.Context, groupID int, vo *data.TaskVO) (*data.TaskVO, error)
	SaveTask(ctx context.Context, groupID, taskID int, vo *data.TaskVO) (*data.TaskVO, error)
	ListTasksByGroupID(ctx context.Context, groupID int) ([]data.TaskVO, error)
	GetTaskDetail(ctx context.Context, groupID, taskID int) (*data.TaskVO, error)
	PublishTask(ctx context.Context, taskID int) (*data.PublishStatusVO, error)
}

type TaskServiceImpl struct {
	taskGroupDao      dao.TaskGroupDao
	taskDao           dao.TaskDao
	taskConditionDao  dao.TaskConditionDao
}

var (
	taskServiceOnce sync.Once
	taskServiceInst *TaskServiceImpl
)

func GetTaskService() *TaskServiceImpl {
	taskServiceOnce.Do(func() {
		taskServiceInst = &TaskServiceImpl{
			taskGroupDao:     dao.GetTaskGroupDao(),
			taskDao:          dao.GetTaskDao(),
			taskConditionDao: dao.GetTaskConditionDao(),
		}
	})
	return taskServiceInst
}

func (s *TaskServiceImpl) CreateTask(ctx context.Context, groupID int, vo *data.TaskVO) (*data.TaskVO, error) {
	if vo == nil {
		return nil, errors.New("task is nil")
	}
	if err := s.validateTaskGroupWritable(ctx, groupID); err != nil {
		return nil, err
	}
	if err := validateTaskInput(vo); err != nil {
		return nil, err
	}

	task := &model.Task{
		TaskGroupID:          groupID,
		Name:                 vo.Name,
		Status:               model.StatusDraft,
		ConditionExpressions: vo.Expression,
		StartTime:            vo.StartTime,
		EndTime:              vo.EndTime,
	}
	conditions := toTaskConditions(0, vo.Conditions)

	id, err := s.taskDao.Save(ctx, task)
	if err != nil {
		log.Logger.Errorf("failed to create task: %v", err)
		return nil, fmt.Errorf("failed to create task: %w", err)
	}
	task.ID = id
	for i := range conditions {
		conditions[i].TaskID = id
	}
	if err := s.taskConditionDao.ReplaceByTaskID(ctx, id, conditions); err != nil {
		log.Logger.Errorf("failed to create task conditions: %v", err)
		return nil, fmt.Errorf("failed to create task conditions: %w", err)
	}
	log.Logger.Infof("task created, id=%d, group_id=%d", task.ID, groupID)
	return s.buildTaskVO(ctx, task, conditions)
}

func (s *TaskServiceImpl) SaveTask(ctx context.Context, groupID, taskID int, vo *data.TaskVO) (*data.TaskVO, error) {
	if vo == nil {
		return nil, errors.New("task is nil")
	}
	if err := validateTaskInput(vo); err != nil {
		return nil, err
	}

	task, err := s.taskDao.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil || task.TaskGroupID != groupID {
		return nil, errors.New("task not found")
	}
	if task.Status == model.StatusPublished {
		return nil, errors.New("published task cannot be modified")
	}
	if err := s.validateTaskGroupWritable(ctx, groupID); err != nil {
		return nil, err
	}

	task.Name = vo.Name
	task.ConditionExpressions = vo.Expression
	task.StartTime = vo.StartTime
	task.EndTime = vo.EndTime
	conditions := toTaskConditions(taskID, vo.Conditions)

	if _, err := s.taskDao.Save(ctx, task); err != nil {
		log.Logger.Errorf("failed to save task %d: %v", taskID, err)
		return nil, fmt.Errorf("failed to save task: %w", err)
	}
	if err := s.taskConditionDao.ReplaceByTaskID(ctx, taskID, conditions); err != nil {
		log.Logger.Errorf("failed to save task conditions %d: %v", taskID, err)
		return nil, fmt.Errorf("failed to save task conditions: %w", err)
	}
	log.Logger.Infof("task saved, id=%d", taskID)
	return s.buildTaskVO(ctx, task, conditions)
}

func (s *TaskServiceImpl) ListTasksByGroupID(ctx context.Context, groupID int) ([]data.TaskVO, error) {
	group, err := s.taskGroupDao.GetByID(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task group: %w", err)
	}
	if group == nil {
		return nil, errors.New("task group not found")
	}

	tasks, err := s.taskDao.ListByGroupID(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	result := make([]data.TaskVO, 0, len(tasks))
	for i := range tasks {
		result = append(result, *s.buildTaskSummary(&tasks[i]))
	}
	return result, nil
}

func (s *TaskServiceImpl) GetTaskDetail(ctx context.Context, groupID, taskID int) (*data.TaskVO, error) {
	task, err := s.taskDao.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil || task.TaskGroupID != groupID {
		return nil, nil
	}
	conditions, err := s.taskConditionDao.ListByTaskID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to list task conditions: %w", err)
	}
	return s.buildTaskVO(ctx, task, conditions)
}

func (s *TaskServiceImpl) PublishTask(ctx context.Context, taskID int) (*data.PublishStatusVO, error) {
	task, err := s.taskDao.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return nil, errors.New("task not found")
	}
	if task.Status == model.StatusPublished {
		return &data.PublishStatusVO{ID: taskID, Status: model.StatusPublished}, nil
	}
	if err := s.taskDao.UpdateStatus(ctx, taskID, model.StatusPublished); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("task not found")
		}
		return nil, fmt.Errorf("failed to publish task: %w", err)
	}
	log.Logger.Infof("task published, id=%d", taskID)
	return &data.PublishStatusVO{ID: taskID, Status: model.StatusPublished}, nil
}

func (s *TaskServiceImpl) validateTaskGroupWritable(ctx context.Context, groupID int) error {
	group, err := s.taskGroupDao.GetByID(ctx, groupID)
	if err != nil {
		return fmt.Errorf("failed to get task group: %w", err)
	}
	if group == nil {
		return errors.New("task group not found")
	}
	if group.Status == model.StatusPublished {
		return errors.New("task group is published and cannot be modified")
	}
	return nil
}

func validateTaskInput(vo *data.TaskVO) error {
	if vo.Name == "" {
		return errors.New("task name is required")
	}
	if len(vo.Conditions) == 0 {
		return errors.New("at least one condition is required")
	}
	for i, cond := range vo.Conditions {
		if cond.MetricID <= 0 || cond.OperatorID <= 0 {
			return fmt.Errorf("invalid condition at index %d", i)
		}
	}
	return nil
}

func toTaskConditions(taskID int, conditions []data.TaskConditionVO) []model.TaskCondition {
	result := make([]model.TaskCondition, 0, len(conditions))
	for i, cond := range conditions {
		no := cond.No
		if no <= 0 {
			no = i + 1
		}
		result = append(result, model.TaskCondition{
			TaskID:         taskID,
			No:             no,
			DataMetricID:   cond.MetricID,
			DataOperatorID: cond.OperatorID,
			ConditionValue: cond.MetricValue,
		})
	}
	return result
}

func (s *TaskServiceImpl) buildTaskSummary(task *model.Task) *data.TaskVO {
	return &data.TaskVO{
		ID:          task.ID,
		Name:        task.Name,
		TaskGroupID: task.TaskGroupID,
		Status:      task.Status,
		Expression:  task.ConditionExpressions,
		StartTime:   task.StartTime,
		EndTime:     task.EndTime,
	}
}

func (s *TaskServiceImpl) buildTaskVO(ctx context.Context, task *model.Task, conditions []model.TaskCondition) (*data.TaskVO, error) {
	vo := s.buildTaskSummary(task)
	if conditions == nil {
		var err error
		conditions, err = s.taskConditionDao.ListByTaskID(ctx, task.ID)
		if err != nil {
			return nil, err
		}
	}
	vo.Conditions = make([]data.TaskConditionVO, 0, len(conditions))
	for _, cond := range conditions {
		vo.Conditions = append(vo.Conditions, data.TaskConditionVO{
			No:          cond.No,
			MetricID:    cond.DataMetricID,
			OperatorID:  cond.DataOperatorID,
			MetricValue: cond.ConditionValue,
		})
	}
	return vo, nil
}
