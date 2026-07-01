package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/nusiss-capstone-project/task-mservice/common/taskpb"
	"github.com/nusiss-capstone-project/task-mservice/server/http/data"
	"github.com/nusiss-capstone-project/task-mservice/server/kafka/producer"
	"github.com/nusiss-capstone-project/task-mservice/server/log"
	"github.com/nusiss-capstone-project/task-mservice/server/repository/dao"
	"github.com/nusiss-capstone-project/task-mservice/server/repository/model"
)

type UserTaskProgressService interface {
	UpdateUserTaskProgress(ctx context.Context, userID, metricID int, metricValue string, eventTime time.Time) error
	EnrollTask(ctx context.Context, enrollTaskRequest *taskpb.EnrollTaskRequest) (*taskpb.EnrollTaskResponse, error)
}

type userTaskProgressServiceImpl struct {
	taskExecutionProgressDao          dao.TaskExecutionProgressDao
	taskConditionExecutionProgressDao dao.TaskConditionExecutionProgressDao
	taskConditionDao                  dao.TaskConditionDao
	taskDao                           dao.TaskDao
	metricOperatorDao                 dao.MetricOperatorDao
	taskCompleteProducer              producer.TaskCompleteProducer
}

func (s *userTaskProgressServiceImpl) EnrollTask(ctx context.Context, req *taskpb.EnrollTaskRequest) (*taskpb.EnrollTaskResponse, error) {
	userID, taskID, ok := validateEnrollTaskRequest(req)
	if !ok {
		return enrollTaskFail(taskpb.ErrorCode_INVALID_PARAM, data.ErrInvalidInput), nil
	}
	task, err := s.taskDao.GetByID(ctx, taskID)
	if err != nil {
		log.Logger.Errorf("load task %d: %v", taskID, err)
		return enrollTaskFail(taskpb.ErrorCode_UNKNOWN_ERROR, data.ErrServerError), nil
	}
	if task == nil {
		return enrollTaskFail(taskpb.ErrorCode_DATA_NOT_EXIST, data.ErrTaskNotFound), nil
	}
	conditions, failResp := s.loadTaskConditions(ctx, taskID)
	if failResp != nil {
		return failResp, nil
	}
	return s.createEnrollment(ctx, userID, taskID, conditions), nil
}

var (
	userTaskProgressServiceOnce sync.Once
	userTaskProgressServiceInst UserTaskProgressService
)

func GetUserTaskProgressService() UserTaskProgressService {
	userTaskProgressServiceOnce.Do(func() {
		userTaskProgressServiceInst = &userTaskProgressServiceImpl{
			taskExecutionProgressDao:          dao.GetTaskExecutionProgressDao(),
			taskConditionExecutionProgressDao: dao.GetTaskConditionExecutionProgressDao(),
			taskConditionDao:                  dao.GetTaskConditionDao(),
			taskDao:                           dao.GetTaskDao(),
			metricOperatorDao:                 dao.GetMetricOperatorDao(),
			taskCompleteProducer:              producer.GetTaskCompleteProducer(),
		}
	})
	return userTaskProgressServiceInst
}

func (s *userTaskProgressServiceImpl) UpdateUserTaskProgress(
	ctx context.Context,
	userID, metricID int,
	metricValue string,
	eventTime time.Time,
) error {
	if userID <= 0 || metricID <= 0 || metricValue == "" || eventTime.IsZero() {
		return errors.New(data.ErrInvalidInput)
	}

	progresses, err := s.loadInProgressConditionProgresses(ctx, userID, metricID)
	if err != nil {
		log.Logger.Errorf("load in-progress condition progress user=%d metric=%d: %v", userID, metricID, err)
		return err
	}
	for _, progress := range progresses {
		if err := s.processConditionProgress(ctx, progress, metricValue, eventTime); err != nil {
			log.Logger.Errorf("process condition progress user=%d metric=%d: %v", userID, metricID, err)
			return err
		}
	}
	return nil
}

func (s *userTaskProgressServiceImpl) loadInProgressConditionProgresses(
	ctx context.Context,
	userID, metricID int,
) ([]model.TaskConditionExecutionProgress, error) {
	progresses, err := s.taskConditionExecutionProgressDao.ListInProgressByUserAndMetric(ctx, userID, metricID)
	if err != nil {
		log.Logger.Errorf("load in-progress condition progress user=%d metric=%d: %v", userID, metricID, err)
		return nil, errors.New(data.ErrServerError)
	}
	return progresses, nil
}

func (s *userTaskProgressServiceImpl) processConditionProgress(
	ctx context.Context,
	progress model.TaskConditionExecutionProgress,
	metricValue string,
	eventTime time.Time,
) error {
	if isStaleEvent(eventTime, progress.LastEventTime) {
		log.Logger.Infof("skip stale event for condition progress %d, event_time=%s last_event_time=%s",
			progress.ID, eventTime.Format(time.RFC3339Nano), formatEventTime(progress.LastEventTime))
		return nil
	}

	condition, err := s.loadTaskCondition(ctx, progress.TaskConditionID)
	if err != nil {
		return err
	}

	if err := s.updateConditionCurrentValue(ctx, progress.ID, metricValue, eventTime); err != nil {
		return err
	}

	operator, err := s.loadMetricOperator(ctx, condition.DataOperatorID)
	if err != nil {
		return err
	}

	matched, err := evaluateMetricOperator(operator.Code, metricValue, condition.ConditionValue)
	if err != nil {
		log.Logger.Errorf("evaluate metric operator for condition progress %d: %v", progress.ID, err)
		return errors.New(data.ErrInvalidInput)
	}
	if !matched {
		return s.tryCompleteTaskExecution(ctx, progress.TaskExecutionProgressID, progress.TaskID, progress.UserID)
	}

	if !canTransitionConditionProgressToComplete(progress.Status) {
		log.Logger.Infof("condition progress %d not completed, ret: %s", progress.ID, progress.Status)
		return nil
	}
	completed, err := s.markConditionProgressComplete(ctx, progress.ID, metricValue, eventTime)
	if err != nil {
		return err
	}
	log.Logger.Infof("condition progress %d completed, ret: %s", progress.ID, completed)
	return s.tryCompleteTaskExecution(ctx, progress.TaskExecutionProgressID, progress.TaskID, progress.UserID)
}

func (s *userTaskProgressServiceImpl) loadTaskCondition(ctx context.Context, conditionID int) (*model.TaskCondition, error) {
	condition, err := s.taskConditionDao.GetByID(ctx, conditionID)
	if err != nil {
		log.Logger.Errorf("load task condition %d: %v", conditionID, err)
		return nil, errors.New(data.ErrServerError)
	}
	if condition == nil {
		log.Logger.Errorf("task condition %d not found", conditionID)
		return nil, errors.New(data.ErrInvalidInput)
	}
	return condition, nil
}

func (s *userTaskProgressServiceImpl) loadMetricOperator(ctx context.Context, operatorID int) (*model.MetricOperator, error) {
	operator, err := s.metricOperatorDao.GetByID(ctx, operatorID)
	if err != nil {
		log.Logger.Errorf("load metric operator %d: %v", operatorID, err)
		return nil, errors.New(data.ErrServerError)
	}
	if operator == nil {
		log.Logger.Errorf("metric operator %d not found", operatorID)
		return nil, errors.New(data.ErrInvalidInput)
	}
	return operator, nil
}

func formatEventTime(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339Nano)
}

func (s *userTaskProgressServiceImpl) updateConditionCurrentValue(
	ctx context.Context,
	progressID int,
	metricValue string,
	eventTime time.Time,
) error {
	updated, err := s.taskConditionExecutionProgressDao.UpdateIfStatusIn(
		ctx, progressID, metricValue, "", eventTime, activeConditionProgressStatuses,
	)
	if err != nil {
		log.Logger.Errorf("update condition progress %d current value: %v", progressID, err)
		return errors.New(data.ErrServerError)
	}
	if !updated {
		log.Logger.Infof("skip current_value update for condition progress %d, status inactive or stale event", progressID)
	}
	return nil
}

func (s *userTaskProgressServiceImpl) markConditionProgressComplete(
	ctx context.Context,
	progressID int,
	metricValue string,
	eventTime time.Time,
) (bool, error) {
	updated, err := s.taskConditionExecutionProgressDao.UpdateIfStatusIn(
		ctx,
		progressID,
		metricValue,
		model.TaskConditionExecutionProgressStatusComplete,
		eventTime,
		conditionCompleteFromStatuses,
	)
	if err != nil {
		log.Logger.Errorf("mark condition progress %d complete: %v", progressID, err)
		return false, errors.New(data.ErrServerError)
	}
	return updated, nil
}

func (s *userTaskProgressServiceImpl) tryCompleteTaskExecution(
	ctx context.Context,
	taskExecutionProgressID, taskID, userID int,
) error {
	taskExecution, err := s.taskExecutionProgressDao.GetByID(ctx, taskExecutionProgressID)
	if err != nil {
		log.Logger.Errorf("load task execution progress %d: %v", taskExecutionProgressID, err)
		return errors.New(data.ErrServerError)
	}
	if taskExecution == nil || isTerminalTaskExecutionProgressStatus(taskExecution.Status) {
		return nil
	}

	task, err := s.loadTask(ctx, taskID)
	if err != nil {
		return err
	}

	conditions, err := s.taskConditionDao.ListByTaskID(ctx, taskID)
	if err != nil {
		log.Logger.Errorf("list task conditions for task %d: %v", taskID, err)
		return errors.New(data.ErrServerError)
	}

	conditionProgresses, err := s.taskConditionExecutionProgressDao.ListByTaskExecutionProgressID(ctx, taskExecutionProgressID)
	if err != nil {
		log.Logger.Errorf("list condition progress for execution %d: %v", taskExecutionProgressID, err)
		return errors.New(data.ErrServerError)
	}

	completedByNo := buildConditionCompletionByNo(conditions, conditionProgresses)
	taskCompleted, err := evaluateTaskExpression(task.ConditionExpressions, completedByNo)
	if err != nil {
		log.Logger.Errorf("evaluate task %d expression: %v", taskID, err)
		return errors.New(data.ErrInvalidInput)
	}
	if !taskCompleted {
		return nil
	}
	log.Logger.Infof("task %d completed, ret: %v", taskID, taskCompleted)
	return s.markTaskExecutionCompleteAndPublish(ctx, taskExecutionProgressID, taskID, userID)
}

func (s *userTaskProgressServiceImpl) loadTask(ctx context.Context, taskID int) (*model.Task, error) {
	task, err := s.taskDao.GetByID(ctx, taskID)
	if err != nil {
		log.Logger.Errorf("load task %d: %v", taskID, err)
		return nil, errors.New(data.ErrServerError)
	}
	if task == nil {
		log.Logger.Errorf("task %d not found", taskID)
		return nil, errors.New(data.ErrTaskNotFound)
	}
	return task, nil
}

func (s *userTaskProgressServiceImpl) markTaskExecutionCompleteAndPublish(
	ctx context.Context,
	taskExecutionProgressID, taskID, userID int,
) error {
	updated, err := s.taskExecutionProgressDao.UpdateStatusIfIn(
		ctx,
		taskExecutionProgressID,
		model.TaskExecutionProgressStatusComplete,
		taskExecutionCompleteFromStatuses,
	)
	if err != nil {
		log.Logger.Errorf("mark task execution progress %d complete: %v", taskExecutionProgressID, err)
		return errors.New(data.ErrServerError)
	}
	// to handler task_execution_progress no existing case
	if !updated {
		current, reloadErr := s.taskExecutionProgressDao.GetByID(ctx, taskExecutionProgressID)
		if reloadErr != nil {
			log.Logger.Errorf("reload task execution progress %d: %v", taskExecutionProgressID, reloadErr)
			return errors.New(data.ErrServerError)
		}
		if current == nil || current.Status != model.TaskExecutionProgressStatusComplete {
			return nil
		}
	}

	if err := s.taskCompleteProducer.PublishTaskCompleted(ctx, taskID, userID, producer.TaskCompletionStatusCompleted); err != nil {
		log.Logger.Errorf("publish task completed event task=%d user=%d: %v", taskID, userID, err)
		return errors.New(data.ErrServerError)
	}
	return nil
}

/*
Idempotency notes without wrapping steps in a DB transaction:

1. Stale events: skipped when eventTime is strictly before progress.LastEventTime.
   Same eventTime is allowed to retry after partial failure; UpdatedAt is not used.
2. current_value / last_event_time: updated together via UpdateIfStatusIn CAS
   (last_event_time IS NULL OR last_event_time <= eventTime).
3. status: guarded by UpdateIfStatusIn / UpdateStatusIfIn (compare-and-set on allowed from-states).
4. Kafka: sent when task execution is Complete (fresh CAS or already Complete for retry after publish failure).

Limitations:
- Kafka may be published more than once on retry or concurrent completion; consumers should deduplicate.
- Concurrent condition updates for the same task can race before expression evaluation; task-level CAS
  ensures only one completion publish in the common case, but intermediate reads are not isolated.
*/
