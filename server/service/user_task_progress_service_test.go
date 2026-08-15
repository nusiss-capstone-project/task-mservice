package service

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/nusiss-capstone-project/task-mservice/common/taskpb"
	"github.com/nusiss-capstone-project/task-mservice/server/http/data"
	producerpkg "github.com/nusiss-capstone-project/task-mservice/server/kafka/producer"
	prodMocks "github.com/nusiss-capstone-project/task-mservice/server/kafka/producer/mocks"
	"github.com/nusiss-capstone-project/task-mservice/server/repository/dao"
	"github.com/nusiss-capstone-project/task-mservice/server/repository/dao/mocks"
	"github.com/nusiss-capstone-project/task-mservice/server/repository/model"
	"github.com/stretchr/testify/mock"
)

func newUserTaskProgressServiceTestDeps() (
	*mocks.TaskExecutionProgressDao,
	*mocks.TaskConditionExecutionProgressDao,
	*mocks.TaskConditionDao,
	*mocks.TaskDao,
	*mocks.TaskGroupDao,
	*mocks.MetricOperatorDao,
	*mocks.DataMetricDao,
	*mocks.MetricEventDedupDao,
	*prodMocks.TaskCompleteProducer,
	*userTaskProgressServiceImpl,
) {
	execDao := new(mocks.TaskExecutionProgressDao)
	condProgressDao := new(mocks.TaskConditionExecutionProgressDao)
	condDao := new(mocks.TaskConditionDao)
	taskDao := new(mocks.TaskDao)
	groupDao := new(mocks.TaskGroupDao)
	opDao := new(mocks.MetricOperatorDao)
	metricDao := new(mocks.DataMetricDao)
	dedupDao := new(mocks.MetricEventDedupDao)
	producer := new(prodMocks.TaskCompleteProducer)
	svc := &userTaskProgressServiceImpl{
		taskExecutionProgressDao:          execDao,
		taskConditionExecutionProgressDao: condProgressDao,
		taskConditionDao:                  condDao,
		taskDao:                           taskDao,
		taskGroupDao:                      groupDao,
		metricOperatorDao:                 opDao,
		dataMetricDao:                     metricDao,
		metricEventDedupDao:               dedupDao,
		taskCompleteProducer:              producer,
	}
	return execDao, condProgressDao, condDao, taskDao, groupDao, opDao, metricDao, dedupDao, producer, svc
}

func TestGetUserTaskProgressService(t *testing.T) {
	initEnv()
	s1 := GetUserTaskProgressService()
	s2 := GetUserTaskProgressService()
	if s1 != s2 {
		t.Fatal("expected singleton instance")
	}
}

func TestEnrollTask(t *testing.T) {
	initEnv()
	ctx := context.Background()

	t.Run("invalid request", func(t *testing.T) {
		_, _, _, _, _, _, _, _, _, svc := newUserTaskProgressServiceTestDeps()
		resp, err := svc.EnrollTask(ctx, nil)
		if err != nil {
			t.Fatalf("EnrollTask() error = %v", err)
		}
		if resp.GetBase().GetCode() != taskpb.ErrorCode_INVALID_PARAM {
			t.Fatalf("unexpected code: %v", resp.GetBase().GetCode())
		}
	})

	t.Run("both task and group set", func(t *testing.T) {
		_, _, _, _, _, _, _, _, _, svc := newUserTaskProgressServiceTestDeps()
		resp, err := svc.EnrollTask(ctx, &taskpb.EnrollTaskRequest{UserId: 1, TaskId: 1, TaskGroupId: 2})
		if err != nil {
			t.Fatalf("EnrollTask() error = %v", err)
		}
		if resp.GetBase().GetCode() != taskpb.ErrorCode_INVALID_PARAM {
			t.Fatalf("unexpected code: %v", resp.GetBase().GetCode())
		}
	})

	t.Run("task not found", func(t *testing.T) {
		_, _, _, taskDao, _, _, _, _, _, svc := newUserTaskProgressServiceTestDeps()
		taskDao.On("GetByID", mock.Anything, 99).Return(nil, nil)
		resp, err := svc.EnrollTask(ctx, &taskpb.EnrollTaskRequest{UserId: 1, TaskId: 99})
		if err != nil {
			t.Fatalf("EnrollTask() error = %v", err)
		}
		if resp.GetBase().GetCode() != taskpb.ErrorCode_DATA_NOT_EXIST {
			t.Fatalf("unexpected code: %v", resp.GetBase().GetCode())
		}
	})

	t.Run("load task error", func(t *testing.T) {
		_, _, _, taskDao, _, _, _, _, _, svc := newUserTaskProgressServiceTestDeps()
		taskDao.On("GetByID", mock.Anything, 1).Return(nil, errors.New("db down"))
		resp, err := svc.EnrollTask(ctx, &taskpb.EnrollTaskRequest{UserId: 1, TaskId: 1})
		if err != nil {
			t.Fatalf("EnrollTask() error = %v", err)
		}
		if resp.GetBase().GetCode() != taskpb.ErrorCode_UNKNOWN_ERROR {
			t.Fatalf("unexpected code: %v", resp.GetBase().GetCode())
		}
	})

	t.Run("no conditions", func(t *testing.T) {
		_, _, condDao, taskDao, _, _, _, _, _, svc := newUserTaskProgressServiceTestDeps()
		taskDao.On("GetByID", mock.Anything, 1).Return(&model.Task{ID: 1}, nil)
		condDao.On("ListByTaskID", mock.Anything, 1).Return([]model.TaskCondition{}, nil)
		resp, err := svc.EnrollTask(ctx, &taskpb.EnrollTaskRequest{UserId: 1, TaskId: 1})
		if err != nil {
			t.Fatalf("EnrollTask() error = %v", err)
		}
		if resp.GetBase().GetCode() != taskpb.ErrorCode_INVALID_PARAM {
			t.Fatalf("unexpected code: %v", resp.GetBase().GetCode())
		}
	})

	t.Run("duplicate enrollment", func(t *testing.T) {
		execDao, _, condDao, taskDao, _, _, _, _, _, svc := newUserTaskProgressServiceTestDeps()
		conditions := []model.TaskCondition{{ID: 10, No: 1}}
		taskDao.On("GetByID", mock.Anything, 1).Return(&model.Task{ID: 1}, nil)
		condDao.On("ListByTaskID", mock.Anything, 1).Return(conditions, nil)
		execDao.On("EnrollUserTasks", mock.Anything, mock.Anything).Return(errors.New("Duplicate entry for key"))
		resp, err := svc.EnrollTask(ctx, &taskpb.EnrollTaskRequest{UserId: 1, TaskId: 1})
		if err != nil {
			t.Fatalf("EnrollTask() error = %v", err)
		}
		if resp.GetBase().GetCode() != taskpb.ErrorCode_INVALID_PARAM {
			t.Fatalf("unexpected code: %v", resp.GetBase().GetCode())
		}
	})

	t.Run("success single task", func(t *testing.T) {
		execDao, _, condDao, taskDao, _, _, _, _, _, svc := newUserTaskProgressServiceTestDeps()
		conditions := []model.TaskCondition{{ID: 10, No: 1}}
		taskDao.On("GetByID", mock.Anything, 1).Return(&model.Task{ID: 1}, nil)
		condDao.On("ListByTaskID", mock.Anything, 1).Return(conditions, nil)
		execDao.On("EnrollUserTasks", mock.Anything, mock.MatchedBy(func(items []dao.EnrollProgressItem) bool {
			if len(items) != 1 || items[0].Execution == nil {
				return false
			}
			items[0].Execution.ID = 100
			if len(items[0].Conditions) != 1 {
				return false
			}
			items[0].Conditions[0].ID = 201
			return items[0].Execution.TaskID == 1 && items[0].Execution.UserID == 1
		})).Return(nil)
		resp, err := svc.EnrollTask(ctx, &taskpb.EnrollTaskRequest{UserId: 1, TaskId: 1})
		if err != nil {
			t.Fatalf("EnrollTask() error = %v", err)
		}
		if resp.GetBase().GetCode() != taskpb.ErrorCode_OK {
			t.Fatalf("unexpected code: %v", resp.GetBase().GetCode())
		}
		if len(resp.GetData()) != 1 {
			t.Fatalf("unexpected data len: %d", len(resp.GetData()))
		}
		got := resp.GetData()[0]
		if got.GetTaskId() != 1 || got.GetTaskExecutionProgressId() != 100 {
			t.Fatalf("unexpected data: %+v", got)
		}
		wantIDs := []int64{201}
		if !reflect.DeepEqual(got.GetTaskConditionExecutionProgressIds(), wantIDs) {
			t.Fatalf("unexpected condition progress ids: %v", got.GetTaskConditionExecutionProgressIds())
		}
	})

	t.Run("group not found", func(t *testing.T) {
		_, _, _, _, groupDao, _, _, _, _, svc := newUserTaskProgressServiceTestDeps()
		groupDao.On("GetByID", mock.Anything, 5).Return(nil, nil)
		resp, err := svc.EnrollTask(ctx, &taskpb.EnrollTaskRequest{UserId: 1, TaskGroupId: 5})
		if err != nil {
			t.Fatalf("EnrollTask() error = %v", err)
		}
		if resp.GetBase().GetCode() != taskpb.ErrorCode_DATA_NOT_EXIST {
			t.Fatalf("unexpected code: %v", resp.GetBase().GetCode())
		}
	})

	t.Run("group has no tasks", func(t *testing.T) {
		_, _, _, taskDao, groupDao, _, _, _, _, svc := newUserTaskProgressServiceTestDeps()
		groupDao.On("GetByID", mock.Anything, 5).Return(&model.TaskGroup{ID: 5}, nil)
		taskDao.On("ListByGroupID", mock.Anything, 5).Return([]model.Task{}, nil)
		resp, err := svc.EnrollTask(ctx, &taskpb.EnrollTaskRequest{UserId: 1, TaskGroupId: 5})
		if err != nil {
			t.Fatalf("EnrollTask() error = %v", err)
		}
		if resp.GetBase().GetCode() != taskpb.ErrorCode_DATA_NOT_EXIST {
			t.Fatalf("unexpected code: %v", resp.GetBase().GetCode())
		}
	})

	t.Run("success group enroll", func(t *testing.T) {
		execDao, _, condDao, taskDao, groupDao, _, _, _, _, svc := newUserTaskProgressServiceTestDeps()
		groupDao.On("GetByID", mock.Anything, 5).Return(&model.TaskGroup{ID: 5}, nil)
		taskDao.On("ListByGroupID", mock.Anything, 5).Return([]model.Task{
			{ID: 1}, {ID: 2},
		}, nil)
		condDao.On("ListByTaskIDs", mock.Anything, []int{1, 2}).Return([]model.TaskCondition{
			{ID: 10, TaskID: 1, No: 1},
			{ID: 20, TaskID: 2, No: 1},
		}, nil)
		execDao.On("EnrollUserTasks", mock.Anything, mock.MatchedBy(func(items []dao.EnrollProgressItem) bool {
			if len(items) != 2 {
				return false
			}
			items[0].Execution.ID = 100
			items[0].Conditions[0].ID = 201
			items[1].Execution.ID = 101
			items[1].Conditions[0].ID = 202
			return items[0].Execution.TaskID == 1 && items[1].Execution.TaskID == 2
		})).Return(nil)
		resp, err := svc.EnrollTask(ctx, &taskpb.EnrollTaskRequest{UserId: 1, TaskGroupId: 5})
		if err != nil {
			t.Fatalf("EnrollTask() error = %v", err)
		}
		if resp.GetBase().GetCode() != taskpb.ErrorCode_OK {
			t.Fatalf("unexpected code: %v", resp.GetBase().GetCode())
		}
		if len(resp.GetData()) != 2 {
			t.Fatalf("unexpected data len: %d", len(resp.GetData()))
		}
	})
}

func TestUpdateUserTaskProgress(t *testing.T) {
	initEnv()
	ctx := context.Background()
	eventTime := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

	t.Run("invalid input", func(t *testing.T) {
		_, _, _, _, _, _, _, _, _, svc := newUserTaskProgressServiceTestDeps()
		if err := svc.UpdateUserTaskProgress(ctx, 0, 1, "true", eventTime, ""); err == nil {
			t.Fatal("expected error for invalid user id")
		}
		if err := svc.UpdateUserTaskProgress(ctx, 1, 1, "", eventTime, ""); err == nil {
			t.Fatal("expected error for empty metric value")
		}
		if err := svc.UpdateUserTaskProgress(ctx, 1, 1, "true", time.Time{}, ""); err == nil {
			t.Fatal("expected error for zero event time")
		}
	})

	t.Run("load progresses error", func(t *testing.T) {
		_, condProgressDao, _, _, _, _, metricDao, _, _, svc := newUserTaskProgressServiceTestDeps()
		metricDao.On("GetByID", mock.Anything, 2).Return(&model.DataMetric{ID: 2, Code: "kyc_passed"}, nil)
		condProgressDao.On("ListInProgressByUserAndMetric", mock.Anything, 1, 2).Return(nil, errors.New("db down"))
		err := svc.UpdateUserTaskProgress(ctx, 1, 2, "true", eventTime, "")
		if err == nil || err.Error() != data.ErrServerError {
			t.Fatalf("UpdateUserTaskProgress() error = %v", err)
		}
	})

	t.Run("no in-progress records", func(t *testing.T) {
		_, condProgressDao, _, _, _, _, metricDao, _, _, svc := newUserTaskProgressServiceTestDeps()
		metricDao.On("GetByID", mock.Anything, 2).Return(&model.DataMetric{ID: 2, Code: "kyc_passed"}, nil)
		condProgressDao.On("ListInProgressByUserAndMetric", mock.Anything, 1, 2).Return([]model.TaskConditionExecutionProgress{}, nil)
		if err := svc.UpdateUserTaskProgress(ctx, 1, 2, "true", eventTime, ""); err != nil {
			t.Fatalf("UpdateUserTaskProgress() error = %v", err)
		}
	})

	t.Run("skip duplicate biz id", func(t *testing.T) {
		_, _, _, _, _, _, metricDao, dedupDao, _, svc := newUserTaskProgressServiceTestDeps()
		metricDao.On("GetByID", mock.Anything, 2).Return(&model.DataMetric{ID: 2, Code: "trade_amount_accum"}, nil)
		dedupDao.On("TryClaim", mock.Anything, "trade_amount_accum", "99").Return(false, nil)
		if err := svc.UpdateUserTaskProgress(ctx, 1, 2, "10", eventTime, "99"); err != nil {
			t.Fatalf("UpdateUserTaskProgress() error = %v", err)
		}
	})

	t.Run("accum adds to current value", func(t *testing.T) {
		execDao, condProgressDao, condDao, taskDao, _, opDao, metricDao, dedupDao, _, svc := newUserTaskProgressServiceTestDeps()
		metricDao.On("GetByID", mock.Anything, 2).Return(&model.DataMetric{ID: 2, Code: "trade_amount_accum"}, nil)
		dedupDao.On("TryClaim", mock.Anything, "trade_amount_accum", "100").Return(true, nil)
		progress := model.TaskConditionExecutionProgress{
			ID: 1, TaskConditionID: 10, TaskExecutionProgressID: 50, TaskID: 5, UserID: 1,
			Status: model.TaskConditionExecutionProgressStatusInProgress, CurrentValue: "20",
		}
		condProgressDao.On("ListInProgressByUserAndMetric", mock.Anything, 1, 2).Return([]model.TaskConditionExecutionProgress{progress}, nil)
		condDao.On("GetByID", mock.Anything, 10).Return(&model.TaskCondition{ID: 10, No: 1, DataOperatorID: 3, ConditionValue: "100"}, nil)
		condProgressDao.On("UpdateIfStatusIn", mock.Anything, 1, "30", "", eventTime, activeConditionProgressStatuses).Return(true, nil)
		opDao.On("GetByID", mock.Anything, 3).Return(&model.MetricOperator{ID: 3, Code: "ge"}, nil)
		execDao.On("GetByID", mock.Anything, 50).Return(&model.TaskExecutionProgress{ID: 50, Status: model.TaskExecutionProgressStatusInProgress}, nil)
		taskDao.On("GetByID", mock.Anything, 5).Return(&model.Task{ID: 5, ConditionExpressions: "(1)"}, nil)
		condDao.On("ListByTaskID", mock.Anything, 5).Return([]model.TaskCondition{{ID: 10, No: 1}}, nil)
		condProgressDao.On("ListByTaskExecutionProgressID", mock.Anything, 50).Return([]model.TaskConditionExecutionProgress{
			{TaskConditionID: 10, Status: model.TaskConditionExecutionProgressStatusInProgress},
		}, nil)
		if err := svc.UpdateUserTaskProgress(ctx, 1, 2, "10", eventTime, "100"); err != nil {
			t.Fatalf("UpdateUserTaskProgress() error = %v", err)
		}
		dedupDao.AssertExpectations(t)
	})

	t.Run("skip stale event", func(t *testing.T) {
		_, condProgressDao, _, _, _, _, metricDao, _, _, svc := newUserTaskProgressServiceTestDeps()
		metricDao.On("GetByID", mock.Anything, 2).Return(&model.DataMetric{ID: 2, Code: "kyc_passed"}, nil)
		lastEvent := eventTime.Add(time.Hour)
		progress := model.TaskConditionExecutionProgress{
			ID: 1, TaskConditionID: 10, TaskExecutionProgressID: 50, TaskID: 5, UserID: 1,
			Status: model.TaskConditionExecutionProgressStatusInProgress, LastEventTime: &lastEvent,
		}
		condProgressDao.On("ListInProgressByUserAndMetric", mock.Anything, 1, 2).Return([]model.TaskConditionExecutionProgress{progress}, nil)
		if err := svc.UpdateUserTaskProgress(ctx, 1, 2, "true", eventTime, ""); err != nil {
			t.Fatalf("UpdateUserTaskProgress() error = %v", err)
		}
	})

	t.Run("condition not found", func(t *testing.T) {
		_, condProgressDao, condDao, _, _, _, metricDao, _, _, svc := newUserTaskProgressServiceTestDeps()
		metricDao.On("GetByID", mock.Anything, 2).Return(&model.DataMetric{ID: 2, Code: "kyc_passed"}, nil)
		progress := model.TaskConditionExecutionProgress{
			ID: 1, TaskConditionID: 10, TaskExecutionProgressID: 50, TaskID: 5, UserID: 1,
			Status: model.TaskConditionExecutionProgressStatusInProgress,
		}
		condProgressDao.On("ListInProgressByUserAndMetric", mock.Anything, 1, 2).Return([]model.TaskConditionExecutionProgress{progress}, nil)
		condDao.On("GetByID", mock.Anything, 10).Return(nil, nil)
		err := svc.UpdateUserTaskProgress(ctx, 1, 2, "true", eventTime, "")
		if err == nil || err.Error() != data.ErrInvalidInput {
			t.Fatalf("UpdateUserTaskProgress() error = %v", err)
		}
	})

	t.Run("operator mismatch skips completion", func(t *testing.T) {
		execDao, condProgressDao, condDao, taskDao, _, opDao, metricDao, _, _, svc := newUserTaskProgressServiceTestDeps()
		metricDao.On("GetByID", mock.Anything, 2).Return(&model.DataMetric{ID: 2, Code: "kyc_passed"}, nil)
		progress := model.TaskConditionExecutionProgress{
			ID: 1, TaskConditionID: 10, TaskExecutionProgressID: 50, TaskID: 5, UserID: 1,
			Status: model.TaskConditionExecutionProgressStatusInProgress,
		}
		condProgressDao.On("ListInProgressByUserAndMetric", mock.Anything, 1, 2).Return([]model.TaskConditionExecutionProgress{progress}, nil)
		condDao.On("GetByID", mock.Anything, 10).Return(&model.TaskCondition{ID: 10, No: 1, DataOperatorID: 3, ConditionValue: "true"}, nil)
		condProgressDao.On("UpdateIfStatusIn", mock.Anything, 1, "false", "", eventTime, activeConditionProgressStatuses).Return(true, nil)
		opDao.On("GetByID", mock.Anything, 3).Return(&model.MetricOperator{ID: 3, Code: "eq"}, nil)
		execDao.On("GetByID", mock.Anything, 50).Return(&model.TaskExecutionProgress{ID: 50, Status: model.TaskExecutionProgressStatusInProgress}, nil)
		taskDao.On("GetByID", mock.Anything, 5).Return(&model.Task{ID: 5, ConditionExpressions: "(1)"}, nil)
		condDao.On("ListByTaskID", mock.Anything, 5).Return([]model.TaskCondition{{ID: 10, No: 1}}, nil)
		condProgressDao.On("ListByTaskExecutionProgressID", mock.Anything, 50).Return([]model.TaskConditionExecutionProgress{
			{TaskConditionID: 10, Status: model.TaskConditionExecutionProgressStatusInProgress},
		}, nil)
		if err := svc.UpdateUserTaskProgress(ctx, 1, 2, "false", eventTime, ""); err != nil {
			t.Fatalf("UpdateUserTaskProgress() error = %v", err)
		}
	})

	t.Run("complete task and publish", func(t *testing.T) {
		execDao, condProgressDao, condDao, taskDao, _, opDao, metricDao, _, producer, svc := newUserTaskProgressServiceTestDeps()
		metricDao.On("GetByID", mock.Anything, 2).Return(&model.DataMetric{ID: 2, Code: "kyc_passed"}, nil)
		progress := model.TaskConditionExecutionProgress{
			ID: 1, TaskConditionID: 10, TaskExecutionProgressID: 50, TaskID: 5, UserID: 1,
			Status: model.TaskConditionExecutionProgressStatusInProgress,
		}
		condProgressDao.On("ListInProgressByUserAndMetric", mock.Anything, 1, 2).Return([]model.TaskConditionExecutionProgress{progress}, nil)
		condDao.On("GetByID", mock.Anything, 10).Return(&model.TaskCondition{ID: 10, No: 1, DataOperatorID: 3, ConditionValue: "true"}, nil)
		opDao.On("GetByID", mock.Anything, 3).Return(&model.MetricOperator{ID: 3, Code: "eq"}, nil)
		condProgressDao.On("UpdateIfStatusIn", mock.Anything, 1, "true", model.TaskConditionExecutionProgressStatusComplete, eventTime, conditionCompleteFromStatuses).Return(true, nil)
		execDao.On("GetByID", mock.Anything, 50).Return(&model.TaskExecutionProgress{ID: 50, Status: model.TaskExecutionProgressStatusInProgress}, nil)
		taskDao.On("GetByID", mock.Anything, 5).Return(&model.Task{ID: 5, TaskGroupID: 9, ConditionExpressions: "(1)"}, nil)
		condDao.On("ListByTaskID", mock.Anything, 5).Return([]model.TaskCondition{{ID: 10, No: 1}}, nil)
		condProgressDao.On("ListByTaskExecutionProgressID", mock.Anything, 50).Return([]model.TaskConditionExecutionProgress{
			{TaskConditionID: 10, Status: model.TaskConditionExecutionProgressStatusComplete},
		}, nil)
		execDao.On("UpdateStatusIfIn", mock.Anything, 50, model.TaskExecutionProgressStatusComplete, taskExecutionCompleteFromStatuses).Return(true, nil)
		taskDao.On("CountByGroupIDAndStatus", mock.Anything, 9, model.StatusPublished).Return(3, nil)
		execDao.On("CountByUserGroupAndStatus", mock.Anything, 1, 9, model.TaskExecutionProgressStatusComplete).Return(1, nil)
		producer.On("PublishTaskCompleted", mock.Anything, producerpkg.TaskCompletedEvent{
			TaskID: 5, UserID: 1, Status: producerpkg.TaskCompletionStatusCompleted,
			GroupID: 9, CompletedTaskCount: 1, TotalTaskCount: 3,
		}).Return(nil)
		if err := svc.UpdateUserTaskProgress(ctx, 1, 2, "true", eventTime, ""); err != nil {
			t.Fatalf("UpdateUserTaskProgress() error = %v", err)
		}
		producer.AssertExpectations(t)
	})

	t.Run("retry publish when already complete", func(t *testing.T) {
		execDao, condProgressDao, condDao, taskDao, _, opDao, metricDao, _, producer, svc := newUserTaskProgressServiceTestDeps()
		metricDao.On("GetByID", mock.Anything, 2).Return(&model.DataMetric{ID: 2, Code: "kyc_passed"}, nil)
		progress := model.TaskConditionExecutionProgress{
			ID: 1, TaskConditionID: 10, TaskExecutionProgressID: 50, TaskID: 5, UserID: 1,
			Status: model.TaskConditionExecutionProgressStatusInProgress,
		}
		condProgressDao.On("ListInProgressByUserAndMetric", mock.Anything, 1, 2).Return([]model.TaskConditionExecutionProgress{progress}, nil)
		condDao.On("GetByID", mock.Anything, 10).Return(&model.TaskCondition{ID: 10, No: 1, DataOperatorID: 3, ConditionValue: "true"}, nil)
		opDao.On("GetByID", mock.Anything, 3).Return(&model.MetricOperator{ID: 3, Code: "eq"}, nil)
		condProgressDao.On("UpdateIfStatusIn", mock.Anything, 1, "true", model.TaskConditionExecutionProgressStatusComplete, eventTime, conditionCompleteFromStatuses).Return(true, nil)
		execDao.On("GetByID", mock.Anything, 50).Return(&model.TaskExecutionProgress{ID: 50, Status: model.TaskExecutionProgressStatusInProgress}, nil).Once()
		taskDao.On("GetByID", mock.Anything, 5).Return(&model.Task{ID: 5, TaskGroupID: 9, ConditionExpressions: "(1)"}, nil)
		condDao.On("ListByTaskID", mock.Anything, 5).Return([]model.TaskCondition{{ID: 10, No: 1}}, nil)
		condProgressDao.On("ListByTaskExecutionProgressID", mock.Anything, 50).Return([]model.TaskConditionExecutionProgress{
			{TaskConditionID: 10, Status: model.TaskConditionExecutionProgressStatusComplete},
		}, nil)
		execDao.On("UpdateStatusIfIn", mock.Anything, 50, model.TaskExecutionProgressStatusComplete, taskExecutionCompleteFromStatuses).Return(false, nil)
		execDao.On("GetByID", mock.Anything, 50).Return(&model.TaskExecutionProgress{ID: 50, Status: model.TaskExecutionProgressStatusComplete}, nil).Once()
		taskDao.On("CountByGroupIDAndStatus", mock.Anything, 9, model.StatusPublished).Return(3, nil)
		execDao.On("CountByUserGroupAndStatus", mock.Anything, 1, 9, model.TaskExecutionProgressStatusComplete).Return(2, nil)
		producer.On("PublishTaskCompleted", mock.Anything, mock.MatchedBy(func(e producerpkg.TaskCompletedEvent) bool {
			return e.TaskID == 5 && e.UserID == 1 && e.GroupID == 9 && e.CompletedTaskCount == 2 && e.TotalTaskCount == 3
		})).Return(nil)
		if err := svc.UpdateUserTaskProgress(ctx, 1, 2, "true", eventTime, ""); err != nil {
			t.Fatalf("UpdateUserTaskProgress() error = %v", err)
		}
	})

	t.Run("publish error", func(t *testing.T) {
		execDao, condProgressDao, condDao, taskDao, _, opDao, metricDao, _, producer, svc := newUserTaskProgressServiceTestDeps()
		metricDao.On("GetByID", mock.Anything, 2).Return(&model.DataMetric{ID: 2, Code: "kyc_passed"}, nil)
		progress := model.TaskConditionExecutionProgress{
			ID: 1, TaskConditionID: 10, TaskExecutionProgressID: 50, TaskID: 5, UserID: 1,
			Status: model.TaskConditionExecutionProgressStatusInProgress,
		}
		condProgressDao.On("ListInProgressByUserAndMetric", mock.Anything, 1, 2).Return([]model.TaskConditionExecutionProgress{progress}, nil)
		condDao.On("GetByID", mock.Anything, 10).Return(&model.TaskCondition{ID: 10, No: 1, DataOperatorID: 3, ConditionValue: "true"}, nil)
		opDao.On("GetByID", mock.Anything, 3).Return(&model.MetricOperator{ID: 3, Code: "eq"}, nil)
		condProgressDao.On("UpdateIfStatusIn", mock.Anything, 1, "true", model.TaskConditionExecutionProgressStatusComplete, eventTime, conditionCompleteFromStatuses).Return(true, nil)
		execDao.On("GetByID", mock.Anything, 50).Return(&model.TaskExecutionProgress{ID: 50, Status: model.TaskExecutionProgressStatusInProgress}, nil)
		taskDao.On("GetByID", mock.Anything, 5).Return(&model.Task{ID: 5, TaskGroupID: 9, ConditionExpressions: "(1)"}, nil)
		condDao.On("ListByTaskID", mock.Anything, 5).Return([]model.TaskCondition{{ID: 10, No: 1}}, nil)
		condProgressDao.On("ListByTaskExecutionProgressID", mock.Anything, 50).Return([]model.TaskConditionExecutionProgress{
			{TaskConditionID: 10, Status: model.TaskConditionExecutionProgressStatusComplete},
		}, nil)
		execDao.On("UpdateStatusIfIn", mock.Anything, 50, model.TaskExecutionProgressStatusComplete, taskExecutionCompleteFromStatuses).Return(true, nil)
		taskDao.On("CountByGroupIDAndStatus", mock.Anything, 9, model.StatusPublished).Return(3, nil)
		execDao.On("CountByUserGroupAndStatus", mock.Anything, 1, 9, model.TaskExecutionProgressStatusComplete).Return(1, nil)
		producer.On("PublishTaskCompleted", mock.Anything, mock.Anything).Return(errors.New("kafka down"))
		err := svc.UpdateUserTaskProgress(ctx, 1, 2, "true", eventTime, "")
		if err == nil || err.Error() != data.ErrServerError {
			t.Fatalf("UpdateUserTaskProgress() error = %v", err)
		}
	})
}

func TestListUserTaskProgressInGroup(t *testing.T) {
	initEnv()
	ctx := context.Background()
	created := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	updated := time.Date(2026, 8, 2, 11, 0, 0, 0, time.UTC)

	t.Run("invalid input", func(t *testing.T) {
		_, _, _, _, _, _, _, _, _, svc := newUserTaskProgressServiceTestDeps()
		if _, err := svc.ListUserTaskProgressInGroup(ctx, 0, 1); err == nil {
			t.Fatal("expected invalid input")
		}
	})

	t.Run("group not found", func(t *testing.T) {
		_, _, _, _, groupDao, _, _, _, _, svc := newUserTaskProgressServiceTestDeps()
		groupDao.On("GetByID", mock.Anything, 9).Return(nil, nil)
		_, err := svc.ListUserTaskProgressInGroup(ctx, 9, 1)
		if err == nil || err.Error() != data.ErrTaskGroupNotFound {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("success merges progress", func(t *testing.T) {
		execDao, _, _, taskDao, groupDao, _, _, _, _, svc := newUserTaskProgressServiceTestDeps()
		groupDao.On("GetByID", mock.Anything, 9).Return(&model.TaskGroup{ID: 9}, nil)
		taskDao.On("ListByGroupIDAndStatus", mock.Anything, 9, model.StatusPublished).Return([]model.Task{
			{ID: 1, Name: "KYC Completed", CreatedAt: created, UpdatedAt: created},
			{ID: 2, Name: "add payment method", CreatedAt: created, UpdatedAt: created},
		}, nil)
		execDao.On("ListByUserAndGroupID", mock.Anything, 1, 9).Return([]model.TaskExecutionProgress{
			{
				TaskID: 1, UserID: 1, Status: model.TaskExecutionProgressStatusComplete,
				CreatedAt: created, UpdatedAt: updated,
			},
		}, nil)

		got, err := svc.ListUserTaskProgressInGroup(ctx, 9, 1)
		if err != nil {
			t.Fatalf("ListUserTaskProgressInGroup() error = %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("len = %d", len(got))
		}
		if got[0].Status != model.TaskExecutionProgressStatusComplete || got[0].UpdatedAt != updated.Unix() {
			t.Fatalf("task1 unexpected: %+v", got[0])
		}
		if got[1].Status != model.TaskExecutionProgressStatusInit || got[1].Name != "add payment method" {
			t.Fatalf("task2 unexpected: %+v", got[1])
		}
	})
}
