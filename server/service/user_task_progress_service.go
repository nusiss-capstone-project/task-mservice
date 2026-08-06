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
	// UpdateUserTaskProgress applies a metric event. When bizID is non-empty, (metric_code, bizID)
	// is claimed for idempotency; duplicate claims are no-ops. Codes ending with "_accum" add to
	// current_value; otherwise the value is set.
	UpdateUserTaskProgress(ctx context.Context, userID, metricID int, metricValue string, eventTime time.Time, bizID string) error
	EnrollTask(ctx context.Context, enrollTaskRequest *taskpb.EnrollTaskRequest) (*taskpb.EnrollTaskResponse, error)
}

type userTaskProgressServiceImpl struct {
	taskExecutionProgressDao          dao.TaskExecutionProgressDao
	taskConditionExecutionProgressDao dao.TaskConditionExecutionProgressDao
	taskConditionDao                  dao.TaskConditionDao
	taskDao                           dao.TaskDao
	taskGroupDao                      dao.TaskGroupDao
	metricOperatorDao                 dao.MetricOperatorDao
	dataMetricDao                     dao.DataMetricDao
	metricEventDedupDao               dao.MetricEventDedupDao
	taskCompleteProducer              producer.TaskCompleteProducer
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
			taskGroupDao:                      dao.GetTaskGroupDao(),
			metricOperatorDao:                 dao.GetMetricOperatorDao(),
			dataMetricDao:                     dao.GetDataMetricDao(),
			metricEventDedupDao:               dao.GetMetricEventDedupDao(),
			taskCompleteProducer:              producer.GetTaskCompleteProducer(),
		}
	})
	return userTaskProgressServiceInst
}

func (s *userTaskProgressServiceImpl) EnrollTask(ctx context.Context, req *taskpb.EnrollTaskRequest) (*taskpb.EnrollTaskResponse, error) {
	userID, taskID, taskGroupID, ok := validateEnrollTaskRequest(req)
	if !ok {
		log.WithContext(ctx).Errorf("invalid enroll task request: %v", req)
		return enrollTaskFail(taskpb.ErrorCode_INVALID_PARAM, data.ErrInvalidInput), nil
	}
	if taskGroupID > 0 {
		return s.enrollGroup(ctx, userID, taskGroupID), nil
	}
	return s.enrollSingleTask(ctx, userID, taskID), nil
}

func (s *userTaskProgressServiceImpl) enrollSingleTask(ctx context.Context, userID, taskID int) *taskpb.EnrollTaskResponse {
	task, err := s.taskDao.GetByID(ctx, taskID)
	if err != nil {
		log.WithContext(ctx).Errorf("load task %d: %v", taskID, err)
		return enrollTaskFail(taskpb.ErrorCode_UNKNOWN_ERROR, data.ErrServerError)
	}
	if task == nil {
		return enrollTaskFail(taskpb.ErrorCode_DATA_NOT_EXIST, data.ErrTaskNotFound)
	}
	conditions, failResp := s.loadTaskConditions(ctx, taskID)
	if failResp != nil {
		return failResp
	}
	return s.createEnrollment(ctx, []dao.EnrollProgressItem{
		buildEnrollProgressItem(userID, taskID, conditions),
	})
}

func (s *userTaskProgressServiceImpl) enrollGroup(ctx context.Context, userID, taskGroupID int) *taskpb.EnrollTaskResponse {
	group, err := s.taskGroupDao.GetByID(ctx, taskGroupID)
	if err != nil {
		log.WithContext(ctx).Errorf("load task group %d: %v", taskGroupID, err)
		return enrollTaskFail(taskpb.ErrorCode_UNKNOWN_ERROR, data.ErrServerError)
	}
	if group == nil {
		return enrollTaskFail(taskpb.ErrorCode_DATA_NOT_EXIST, data.ErrTaskGroupNotFound)
	}
	tasks, err := s.taskDao.ListByGroupID(ctx, taskGroupID)
	if err != nil {
		log.WithContext(ctx).Errorf("list tasks for group %d: %v", taskGroupID, err)
		return enrollTaskFail(taskpb.ErrorCode_UNKNOWN_ERROR, data.ErrServerError)
	}
	if len(tasks) == 0 {
		log.WithContext(ctx).Errorf("task group %d has no tasks", taskGroupID)
		return enrollTaskFail(taskpb.ErrorCode_DATA_NOT_EXIST, data.ErrTaskNotFound)
	}

	taskIDs := make([]int, 0, len(tasks))
	for _, task := range tasks {
		taskIDs = append(taskIDs, task.ID)
	}
	conditionsByTask, failResp := s.loadTaskConditionsByTaskIDs(ctx, taskIDs)
	if failResp != nil {
		return failResp
	}

	items := make([]dao.EnrollProgressItem, 0, len(tasks))
	for _, task := range tasks {
		conditions := conditionsByTask[task.ID]
		if len(conditions) == 0 {
			log.WithContext(ctx).Errorf("task %d has no conditions", task.ID)
			return enrollTaskFail(taskpb.ErrorCode_INVALID_PARAM, data.ErrAtLeastOneConditionRequired)
		}
		items = append(items, buildEnrollProgressItem(userID, task.ID, conditions))
	}
	return s.createEnrollment(ctx, items)
}

func (s *userTaskProgressServiceImpl) UpdateUserTaskProgress(
	ctx context.Context,
	userID, metricID int,
	metricValue string,
	eventTime time.Time,
	bizID string,
) error {
	if userID <= 0 || metricID <= 0 || metricValue == "" || eventTime.IsZero() {
		return errors.New(data.ErrInvalidInput)
	}

	metric, err := s.dataMetricDao.GetByID(ctx, metricID)
	if err != nil {
		log.WithContext(ctx).Errorf("load data metric %d: %v", metricID, err)
		return errors.New(data.ErrServerError)
	}
	if metric == nil {
		log.WithContext(ctx).Errorf("data metric %d not found", metricID)
		return errors.New(data.ErrInvalidInput)
	}

	claimed := false
	if bizID != "" {
		ok, claimErr := s.metricEventDedupDao.TryClaim(ctx, metric.Code, bizID)
		if claimErr != nil {
			log.WithContext(ctx).Errorf("claim metric event %s/%s: %v", metric.Code, bizID, claimErr)
			return errors.New(data.ErrServerError)
		}
		if !ok {
			log.WithContext(ctx).Infof("skip duplicate metric event %s/%s", metric.Code, bizID)
			return nil
		}
		claimed = true
	}

	err = s.applyMetricToProgresses(ctx, userID, metricID, metric.Code, metricValue, eventTime)
	if err != nil && claimed {
		if releaseErr := s.metricEventDedupDao.Release(ctx, metric.Code, bizID); releaseErr != nil {
			log.WithContext(ctx).Errorf("release metric event %s/%s after failure: %v", metric.Code, bizID, releaseErr)
		}
	}
	return err
}

func (s *userTaskProgressServiceImpl) applyMetricToProgresses(
	ctx context.Context,
	userID, metricID int,
	metricCode, metricValue string,
	eventTime time.Time,
) error {
	progresses, err := s.loadInProgressConditionProgresses(ctx, userID, metricID)
	if err != nil {
		log.WithContext(ctx).Errorf("load in-progress condition progress user=%d metric=%d: %v", userID, metricID, err)
		return err
	}
	for _, progress := range progresses {
		if err := s.processConditionProgress(ctx, progress, metricCode, metricValue, eventTime); err != nil {
			log.WithContext(ctx).Errorf("process condition progress user=%d metric=%d: %v", userID, metricID, err)
			return err
		}
	}
	return nil
}

func (s *userTaskProgressServiceImpl) loadTaskConditions(ctx context.Context, taskID int) ([]model.TaskCondition, *taskpb.EnrollTaskResponse) {
	conditions, err := s.taskConditionDao.ListByTaskID(ctx, taskID)
	if err != nil {
		log.WithContext(ctx).Errorf("list task conditions for task %d: %v", taskID, err)
		return nil, enrollTaskFail(taskpb.ErrorCode_UNKNOWN_ERROR, data.ErrServerError)
	}
	if len(conditions) == 0 {
		log.WithContext(ctx).Errorf("task %d has no conditions", taskID)
		return nil, enrollTaskFail(taskpb.ErrorCode_INVALID_PARAM, data.ErrAtLeastOneConditionRequired)
	}
	return conditions, nil
}

func (s *userTaskProgressServiceImpl) loadTaskConditionsByTaskIDs(
	ctx context.Context,
	taskIDs []int,
) (map[int][]model.TaskCondition, *taskpb.EnrollTaskResponse) {
	conditions, err := s.taskConditionDao.ListByTaskIDs(ctx, taskIDs)
	if err != nil {
		log.WithContext(ctx).Errorf("list task conditions for tasks %v: %v", taskIDs, err)
		return nil, enrollTaskFail(taskpb.ErrorCode_UNKNOWN_ERROR, data.ErrServerError)
	}
	byTask := make(map[int][]model.TaskCondition, len(taskIDs))
	for _, condition := range conditions {
		byTask[condition.TaskID] = append(byTask[condition.TaskID], condition)
	}
	return byTask, nil
}

func (s *userTaskProgressServiceImpl) createEnrollment(ctx context.Context, items []dao.EnrollProgressItem) *taskpb.EnrollTaskResponse {
	if err := s.taskExecutionProgressDao.EnrollUserTasks(ctx, items); err != nil {
		if isDuplicateEntryError(err) {
			log.WithContext(ctx).Infof("duplicate enrollment detected: %v", err)
			return enrollTaskFail(taskpb.ErrorCode_INVALID_PARAM, data.ErrInvalidInput)
		}
		log.WithContext(ctx).Errorf("enroll tasks: %v", err)
		return enrollTaskFail(taskpb.ErrorCode_UNKNOWN_ERROR, data.ErrServerError)
	}
	return enrollTaskSuccess(items)
}

func (s *userTaskProgressServiceImpl) loadInProgressConditionProgresses(
	ctx context.Context,
	userID, metricID int,
) ([]model.TaskConditionExecutionProgress, error) {
	progresses, err := s.taskConditionExecutionProgressDao.ListInProgressByUserAndMetric(ctx, userID, metricID)
	if err != nil {
		log.WithContext(ctx).Errorf("load in-progress condition progress user=%d metric=%d: %v", userID, metricID, err)
		return nil, errors.New(data.ErrServerError)
	}
	return progresses, nil
}

func (s *userTaskProgressServiceImpl) processConditionProgress(
	ctx context.Context,
	progress model.TaskConditionExecutionProgress,
	metricCode, metricValue string,
	eventTime time.Time,
) error {
	if isStaleEvent(eventTime, progress.LastEventTime) {
		log.WithContext(ctx).Infof("skip stale event for condition progress %d, event_time=%s last_event_time=%s",
			progress.ID, eventTime.Format(time.RFC3339Nano), formatEventTime(progress.LastEventTime))
		return nil
	}

	condition, err := s.loadTaskCondition(ctx, progress.TaskConditionID)
	if err != nil {
		return err
	}

	effectiveValue, err := resolveMetricValue(metricCode, progress.CurrentValue, metricValue)
	if err != nil {
		log.WithContext(ctx).Errorf("resolve metric value for condition progress %d: %v", progress.ID, err)
		return errors.New(data.ErrInvalidInput)
	}

	if err := s.updateConditionCurrentValue(ctx, progress.ID, effectiveValue, eventTime); err != nil {
		return err
	}

	operator, err := s.loadMetricOperator(ctx, condition.DataOperatorID)
	if err != nil {
		return err
	}

	matched, err := evaluateMetricOperator(operator.Code, effectiveValue, condition.ConditionValue)
	if err != nil {
		log.WithContext(ctx).Errorf("evaluate metric operator for condition progress %d: %v", progress.ID, err)
		return errors.New(data.ErrInvalidInput)
	}
	if !matched {
		return s.tryCompleteTaskExecution(ctx, progress.TaskExecutionProgressID, progress.TaskID, progress.UserID)
	}

	if !canTransitionConditionProgressToComplete(progress.Status) {
		log.WithContext(ctx).Infof("condition progress %d not completed, ret: %s", progress.ID, progress.Status)
		return nil
	}
	completed, err := s.markConditionProgressComplete(ctx, progress.ID, effectiveValue, eventTime)
	if err != nil {
		return err
	}
	log.WithContext(ctx).Infof("condition progress %d completed, ret: %s", progress.ID, completed)
	return s.tryCompleteTaskExecution(ctx, progress.TaskExecutionProgressID, progress.TaskID, progress.UserID)
}

func (s *userTaskProgressServiceImpl) loadTaskCondition(ctx context.Context, conditionID int) (*model.TaskCondition, error) {
	condition, err := s.taskConditionDao.GetByID(ctx, conditionID)
	if err != nil {
		log.WithContext(ctx).Errorf("load task condition %d: %v", conditionID, err)
		return nil, errors.New(data.ErrServerError)
	}
	if condition == nil {
		log.WithContext(ctx).Errorf("task condition %d not found", conditionID)
		return nil, errors.New(data.ErrInvalidInput)
	}
	return condition, nil
}

func (s *userTaskProgressServiceImpl) loadMetricOperator(ctx context.Context, operatorID int) (*model.MetricOperator, error) {
	operator, err := s.metricOperatorDao.GetByID(ctx, operatorID)
	if err != nil {
		log.WithContext(ctx).Errorf("load metric operator %d: %v", operatorID, err)
		return nil, errors.New(data.ErrServerError)
	}
	if operator == nil {
		log.WithContext(ctx).Errorf("metric operator %d not found", operatorID)
		return nil, errors.New(data.ErrInvalidInput)
	}
	return operator, nil
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
		log.WithContext(ctx).Errorf("update condition progress %d current value: %v", progressID, err)
		return errors.New(data.ErrServerError)
	}
	if !updated {
		log.WithContext(ctx).Infof("skip current_value update for condition progress %d, status inactive or stale event", progressID)
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
		log.WithContext(ctx).Errorf("mark condition progress %d complete: %v", progressID, err)
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
		log.WithContext(ctx).Errorf("load task execution progress %d: %v", taskExecutionProgressID, err)
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
		log.WithContext(ctx).Errorf("list task conditions for task %d: %v", taskID, err)
		return errors.New(data.ErrServerError)
	}

	conditionProgresses, err := s.taskConditionExecutionProgressDao.ListByTaskExecutionProgressID(ctx, taskExecutionProgressID)
	if err != nil {
		log.WithContext(ctx).Errorf("list condition progress for execution %d: %v", taskExecutionProgressID, err)
		return errors.New(data.ErrServerError)
	}

	completedByNo := buildConditionCompletionByNo(conditions, conditionProgresses)
	taskCompleted, err := evaluateTaskExpression(task.ConditionExpressions, completedByNo)
	if err != nil {
		log.WithContext(ctx).Errorf("evaluate task %d expression: %v", taskID, err)
		return errors.New(data.ErrInvalidInput)
	}
	if !taskCompleted {
		return nil
	}
	log.WithContext(ctx).Infof("task %d completed, ret: %v", taskID, taskCompleted)
	return s.markTaskExecutionCompleteAndPublish(ctx, taskExecutionProgressID, taskID, userID)
}

func (s *userTaskProgressServiceImpl) loadTask(ctx context.Context, taskID int) (*model.Task, error) {
	task, err := s.taskDao.GetByID(ctx, taskID)
	if err != nil {
		log.WithContext(ctx).Errorf("load task %d: %v", taskID, err)
		return nil, errors.New(data.ErrServerError)
	}
	if task == nil {
		log.WithContext(ctx).Errorf("task %d not found", taskID)
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
		log.WithContext(ctx).Errorf("mark task execution progress %d complete: %v", taskExecutionProgressID, err)
		return errors.New(data.ErrServerError)
	}
	// to handler task_execution_progress no existing case
	if !updated {
		current, reloadErr := s.taskExecutionProgressDao.GetByID(ctx, taskExecutionProgressID)
		if reloadErr != nil {
			log.WithContext(ctx).Errorf("reload task execution progress %d: %v", taskExecutionProgressID, reloadErr)
			return errors.New(data.ErrServerError)
		}
		if current == nil || current.Status != model.TaskExecutionProgressStatusComplete {
			return nil
		}
	}

	return s.publishTaskCompletedEvent(ctx, taskID, userID)
}

// publishTaskCompletedEvent loads group progress stats and publishes task.events.completed.
func (s *userTaskProgressServiceImpl) publishTaskCompletedEvent(ctx context.Context, taskID, userID int) error {
	task, err := s.loadTask(ctx, taskID)
	if err != nil {
		return err
	}

	total, err := s.taskDao.CountByGroupIDAndStatus(ctx, task.TaskGroupID, model.StatusPublished)
	if err != nil {
		log.WithContext(ctx).Errorf("count published tasks for group %d: %v", task.TaskGroupID, err)
		return errors.New(data.ErrServerError)
	}
	completed, err := s.taskExecutionProgressDao.CountByUserGroupAndStatus(
		ctx, userID, task.TaskGroupID, model.TaskExecutionProgressStatusComplete,
	)
	if err != nil {
		log.WithContext(ctx).Errorf(
			"count completed executions user=%d group=%d: %v", userID, task.TaskGroupID, err,
		)
		return errors.New(data.ErrServerError)
	}

	event := producer.TaskCompletedEvent{
		TaskID:             taskID,
		UserID:             userID,
		Status:             producer.TaskCompletionStatusCompleted,
		GroupID:            task.TaskGroupID,
		CompletedTaskCount: completed,
		TotalTaskCount:     total,
	}
	if err := s.taskCompleteProducer.PublishTaskCompleted(ctx, event); err != nil {
		log.WithContext(ctx).Errorf("publish task completed event task=%d user=%d: %v", taskID, userID, err)
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
