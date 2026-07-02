package service

import (
	"context"
	"errors"
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
	taskGroupDao     dao.TaskGroupDao
	taskDao          dao.TaskDao
	taskConditionDao dao.TaskConditionDao
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
		return nil, errors.New(data.ErrTaskNil)
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
		StartTime:            vo.StartTime.TimePtr(),
		EndTime:              vo.EndTime.TimePtr(),
	}
	conditions := toTaskConditions(0, vo.Conditions)

	id, err := s.taskDao.Save(ctx, task)
	if err != nil {
		log.Logger.Errorf("create task in group %d: %v", groupID, err)
		return nil, errors.New(data.ErrServerError)
	}
	task.ID = id
	for i := range conditions {
		conditions[i].TaskID = id
	}
	if err := s.taskConditionDao.ReplaceByTaskID(ctx, id, conditions); err != nil {
		log.Logger.Errorf("create task conditions for task %d: %v", id, err)
		return nil, errors.New(data.ErrServerError)
	}
	log.Logger.Infof("task created, id=%d, group_id=%d", task.ID, groupID)
	return s.buildTaskVO(ctx, task, conditions)
}

func (s *TaskServiceImpl) SaveTask(ctx context.Context, groupID, taskID int, vo *data.TaskVO) (*data.TaskVO, error) {
	if vo == nil {
		return nil, errors.New(data.ErrTaskNil)
	}
	if err := validateTaskInput(vo); err != nil {
		return nil, err
	}

	task, err := s.taskDao.GetByID(ctx, taskID)
	if err != nil {
		log.Logger.Errorf("[SaveTask] get task %d: %v", taskID, err)
		return nil, errors.New(data.ErrServerError)
	}
	if task == nil || task.TaskGroupID != groupID {
		return nil, errors.New(data.ErrTaskNotFound)
	}
	if task.Status == model.StatusPublished {
		return nil, errors.New(data.ErrPublishedTaskCannotModify)
	}
	if err := s.validateTaskGroupWritable(ctx, groupID); err != nil {
		return nil, err
	}

	task.Name = vo.Name
	task.ConditionExpressions = vo.Expression
	task.StartTime = vo.StartTime.TimePtr()
	task.EndTime = vo.EndTime.TimePtr()
	conditions := toTaskConditions(taskID, vo.Conditions)

	if _, err := s.taskDao.Save(ctx, task); err != nil {
		log.Logger.Errorf("save task %d: %v", taskID, err)
		return nil, errors.New(data.ErrServerError)
	}
	if err := s.taskConditionDao.ReplaceByTaskID(ctx, taskID, conditions); err != nil {
		log.Logger.Errorf("save task conditions for task %d: %v", taskID, err)
		return nil, errors.New(data.ErrServerError)
	}
	log.Logger.Infof("task saved, id=%d", taskID)
	return s.buildTaskVO(ctx, task, conditions)
}

func (s *TaskServiceImpl) ListTasksByGroupID(ctx context.Context, groupID int) ([]data.TaskVO, error) {
	group, err := s.taskGroupDao.GetByID(ctx, groupID)
	if err != nil {
		log.Logger.Errorf("[ListTasksByGroupID] get task group %d: %v", groupID, err)
		return nil, errors.New(data.ErrServerError)
	}
	if group == nil {
		return nil, errors.New(data.ErrTaskGroupNotFound)
	}

	tasks, err := s.taskDao.ListByGroupID(ctx, groupID)
	if err != nil {
		log.Logger.Errorf("list tasks for group %d: %v", groupID, err)
		return nil, errors.New(data.ErrServerError)
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
		log.Logger.Errorf("[GetTaskDetail] get task %d: %v", taskID, err)
		return nil, errors.New(data.ErrServerError)
	}
	if task == nil || task.TaskGroupID != groupID {
		return nil, nil
	}
	conditions, err := s.taskConditionDao.ListByTaskID(ctx, taskID)
	if err != nil {
		log.Logger.Errorf("list conditions for task %d: %v", taskID, err)
		return nil, errors.New(data.ErrServerError)
	}
	return s.buildTaskVO(ctx, task, conditions)
}

func (s *TaskServiceImpl) PublishTask(ctx context.Context, taskID int) (*data.PublishStatusVO, error) {
	task, err := s.taskDao.GetByID(ctx, taskID)
	if err != nil {
		log.Logger.Errorf("[PublishTask] get task %d: %v", taskID, err)
		return nil, errors.New(data.ErrServerError)
	}
	if task == nil {
		return nil, errors.New(data.ErrTaskNotFound)
	}
	if task.Status == model.StatusPublished {
		return &data.PublishStatusVO{ID: taskID, Status: model.StatusPublished}, nil
	}
	if err := s.taskDao.UpdateStatus(ctx, taskID, model.StatusPublished); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(data.ErrTaskNotFound)
		}
		log.Logger.Errorf("publish task %d: %v", taskID, err)
		return nil, errors.New(data.ErrServerError)
	}
	log.Logger.Infof("task published, id=%d", taskID)
	return &data.PublishStatusVO{ID: taskID, Status: model.StatusPublished}, nil
}

func (s *TaskServiceImpl) validateTaskGroupWritable(ctx context.Context, groupID int) error {
	group, err := s.taskGroupDao.GetByID(ctx, groupID)
	if err != nil {
		log.Logger.Errorf("[validateTaskGroupWritable] get task group %d: %v", groupID, err)
		return errors.New(data.ErrServerError)
	}
	if group == nil {
		return errors.New(data.ErrTaskGroupNotFound)
	}
	if group.Status == model.StatusPublished {
		return errors.New(data.ErrTaskGroupPublishedCannotModify)
	}
	return nil
}

func validateTaskInput(vo *data.TaskVO) error {
	if vo.Name == "" {
		return errors.New(data.ErrTaskNameRequired)
	}
	if len(vo.Conditions) == 0 {
		return errors.New(data.ErrAtLeastOneConditionRequired)
	}
	for _, cond := range vo.Conditions {
		if cond.MetricID <= 0 || cond.OperatorID <= 0 {
			return errors.New(data.ErrInvalidInput)
		}
	}
	if err := ValidateTaskExpression(vo.Expression, conditionNosFromVO(vo.Conditions)); err != nil {
		return errors.New(data.ErrInvalidInput)
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
		StartTime:   data.DateTimeFromPtr(task.StartTime),
		EndTime:     data.DateTimeFromPtr(task.EndTime),
	}
}

func (s *TaskServiceImpl) buildTaskVO(ctx context.Context, task *model.Task, conditions []model.TaskCondition) (*data.TaskVO, error) {
	vo := s.buildTaskSummary(task)
	if conditions == nil {
		var err error
		conditions, err = s.taskConditionDao.ListByTaskID(ctx, task.ID)
		if err != nil {
			log.Logger.Errorf("list conditions for task %d: %v", task.ID, err)
			return nil, errors.New(data.ErrServerError)
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
