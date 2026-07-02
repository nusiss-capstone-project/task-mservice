package service

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/nusiss-capstone-project/task-mservice/common/taskpb"
	"github.com/nusiss-capstone-project/task-mservice/server/http/data"
	prodMocks "github.com/nusiss-capstone-project/task-mservice/server/kafka/producer/mocks"
	"github.com/nusiss-capstone-project/task-mservice/server/repository/dao/mocks"
	"github.com/nusiss-capstone-project/task-mservice/server/repository/model"
	"github.com/stretchr/testify/mock"
)

func newUserTaskProgressServiceTestDeps() (
	*mocks.TaskExecutionProgressDao,
	*mocks.TaskConditionExecutionProgressDao,
	*mocks.TaskConditionDao,
	*mocks.TaskDao,
	*mocks.MetricOperatorDao,
	*prodMocks.TaskCompleteProducer,
	*userTaskProgressServiceImpl,
) {
	execDao := new(mocks.TaskExecutionProgressDao)
	condProgressDao := new(mocks.TaskConditionExecutionProgressDao)
	condDao := new(mocks.TaskConditionDao)
	taskDao := new(mocks.TaskDao)
	opDao := new(mocks.MetricOperatorDao)
	producer := new(prodMocks.TaskCompleteProducer)
	svc := &userTaskProgressServiceImpl{
		taskExecutionProgressDao:          execDao,
		taskConditionExecutionProgressDao: condProgressDao,
		taskConditionDao:                  condDao,
		taskDao:                           taskDao,
		metricOperatorDao:                 opDao,
		taskCompleteProducer:              producer,
	}
	return execDao, condProgressDao, condDao, taskDao, opDao, producer, svc
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
		_, _, _, _, _, _, svc := newUserTaskProgressServiceTestDeps()
		resp, err := svc.EnrollTask(ctx, nil)
		if err != nil {
			t.Fatalf("EnrollTask() error = %v", err)
		}
		if resp.GetBase().GetCode() != taskpb.ErrorCode_INVALID_PARAM {
			t.Fatalf("unexpected code: %v", resp.GetBase().GetCode())
		}
	})

	t.Run("task not found", func(t *testing.T) {
		_, _, _, taskDao, _, _, svc := newUserTaskProgressServiceTestDeps()
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
		_, _, _, taskDao, _, _, svc := newUserTaskProgressServiceTestDeps()
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
		_, _, condDao, taskDao, _, _, svc := newUserTaskProgressServiceTestDeps()
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
		execDao, _, condDao, taskDao, _, _, svc := newUserTaskProgressServiceTestDeps()
		conditions := []model.TaskCondition{{ID: 10, No: 1}}
		taskDao.On("GetByID", mock.Anything, 1).Return(&model.Task{ID: 1}, nil)
		condDao.On("ListByTaskID", mock.Anything, 1).Return(conditions, nil)
		execDao.On("EnrollUserTask", mock.Anything, 1, 1, conditions).Return(0, nil, errors.New("Duplicate entry for key"))
		resp, err := svc.EnrollTask(ctx, &taskpb.EnrollTaskRequest{UserId: 1, TaskId: 1})
		if err != nil {
			t.Fatalf("EnrollTask() error = %v", err)
		}
		if resp.GetBase().GetCode() != taskpb.ErrorCode_INVALID_PARAM {
			t.Fatalf("unexpected code: %v", resp.GetBase().GetCode())
		}
	})

	t.Run("success", func(t *testing.T) {
		execDao, _, condDao, taskDao, _, _, svc := newUserTaskProgressServiceTestDeps()
		conditions := []model.TaskCondition{{ID: 10, No: 1}}
		taskDao.On("GetByID", mock.Anything, 1).Return(&model.Task{ID: 1}, nil)
		condDao.On("ListByTaskID", mock.Anything, 1).Return(conditions, nil)
		execDao.On("EnrollUserTask", mock.Anything, 1, 1, conditions).Return(100, []int{201, 202}, nil)
		resp, err := svc.EnrollTask(ctx, &taskpb.EnrollTaskRequest{UserId: 1, TaskId: 1})
		if err != nil {
			t.Fatalf("EnrollTask() error = %v", err)
		}
		if resp.GetBase().GetCode() != taskpb.ErrorCode_OK {
			t.Fatalf("unexpected code: %v", resp.GetBase().GetCode())
		}
		if resp.GetData().GetTaskExecutionProgressId() != 100 {
			t.Fatalf("unexpected execution id: %d", resp.GetData().GetTaskExecutionProgressId())
		}
		wantIDs := []int64{201, 202}
		if !reflect.DeepEqual(resp.GetData().GetTaskConditionExecutionProgressIds(), wantIDs) {
			t.Fatalf("unexpected condition progress ids: %v", resp.GetData().GetTaskConditionExecutionProgressIds())
		}
	})
}

func TestUpdateUserTaskProgress(t *testing.T) {
	initEnv()
	ctx := context.Background()
	eventTime := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

	t.Run("invalid input", func(t *testing.T) {
		_, _, _, _, _, _, svc := newUserTaskProgressServiceTestDeps()
		if err := svc.UpdateUserTaskProgress(ctx, 0, 1, "true", eventTime); err == nil {
			t.Fatal("expected error for invalid user id")
		}
		if err := svc.UpdateUserTaskProgress(ctx, 1, 1, "", eventTime); err == nil {
			t.Fatal("expected error for empty metric value")
		}
		if err := svc.UpdateUserTaskProgress(ctx, 1, 1, "true", time.Time{}); err == nil {
			t.Fatal("expected error for zero event time")
		}
	})

	t.Run("load progresses error", func(t *testing.T) {
		_, condProgressDao, _, _, _, _, svc := newUserTaskProgressServiceTestDeps()
		condProgressDao.On("ListInProgressByUserAndMetric", mock.Anything, 1, 2).Return(nil, errors.New("db down"))
		err := svc.UpdateUserTaskProgress(ctx, 1, 2, "true", eventTime)
		if err == nil || err.Error() != data.ErrServerError {
			t.Fatalf("UpdateUserTaskProgress() error = %v", err)
		}
	})

	t.Run("no in-progress records", func(t *testing.T) {
		_, condProgressDao, _, _, _, _, svc := newUserTaskProgressServiceTestDeps()
		condProgressDao.On("ListInProgressByUserAndMetric", mock.Anything, 1, 2).Return([]model.TaskConditionExecutionProgress{}, nil)
		if err := svc.UpdateUserTaskProgress(ctx, 1, 2, "true", eventTime); err != nil {
			t.Fatalf("UpdateUserTaskProgress() error = %v", err)
		}
	})

	t.Run("skip stale event", func(t *testing.T) {
		_, condProgressDao, _, _, _, _, svc := newUserTaskProgressServiceTestDeps()
		lastEvent := eventTime.Add(time.Hour)
		progress := model.TaskConditionExecutionProgress{
			ID: 1, TaskConditionID: 10, TaskExecutionProgressID: 50, TaskID: 5, UserID: 1,
			Status: model.TaskConditionExecutionProgressStatusInProgress, LastEventTime: &lastEvent,
		}
		condProgressDao.On("ListInProgressByUserAndMetric", mock.Anything, 1, 2).Return([]model.TaskConditionExecutionProgress{progress}, nil)
		if err := svc.UpdateUserTaskProgress(ctx, 1, 2, "true", eventTime); err != nil {
			t.Fatalf("UpdateUserTaskProgress() error = %v", err)
		}
	})

	t.Run("condition not found", func(t *testing.T) {
		_, condProgressDao, condDao, _, _, _, svc := newUserTaskProgressServiceTestDeps()
		progress := model.TaskConditionExecutionProgress{
			ID: 1, TaskConditionID: 10, TaskExecutionProgressID: 50, TaskID: 5, UserID: 1,
			Status: model.TaskConditionExecutionProgressStatusInProgress,
		}
		condProgressDao.On("ListInProgressByUserAndMetric", mock.Anything, 1, 2).Return([]model.TaskConditionExecutionProgress{progress}, nil)
		condDao.On("GetByID", mock.Anything, 10).Return(nil, nil)
		err := svc.UpdateUserTaskProgress(ctx, 1, 2, "true", eventTime)
		if err == nil || err.Error() != data.ErrInvalidInput {
			t.Fatalf("UpdateUserTaskProgress() error = %v", err)
		}
	})

	t.Run("operator mismatch skips completion", func(t *testing.T) {
		execDao, condProgressDao, condDao, taskDao, opDao, _, svc := newUserTaskProgressServiceTestDeps()
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
		if err := svc.UpdateUserTaskProgress(ctx, 1, 2, "false", eventTime); err != nil {
			t.Fatalf("UpdateUserTaskProgress() error = %v", err)
		}
	})

	t.Run("complete task and publish", func(t *testing.T) {
		execDao, condProgressDao, condDao, taskDao, opDao, producer, svc := newUserTaskProgressServiceTestDeps()
		progress := model.TaskConditionExecutionProgress{
			ID: 1, TaskConditionID: 10, TaskExecutionProgressID: 50, TaskID: 5, UserID: 1,
			Status: model.TaskConditionExecutionProgressStatusInProgress,
		}
		condProgressDao.On("ListInProgressByUserAndMetric", mock.Anything, 1, 2).Return([]model.TaskConditionExecutionProgress{progress}, nil)
		condDao.On("GetByID", mock.Anything, 10).Return(&model.TaskCondition{ID: 10, No: 1, DataOperatorID: 3, ConditionValue: "true"}, nil)
		condProgressDao.On("UpdateIfStatusIn", mock.Anything, 1, "true", "", eventTime, activeConditionProgressStatuses).Return(true, nil)
		opDao.On("GetByID", mock.Anything, 3).Return(&model.MetricOperator{ID: 3, Code: "eq"}, nil)
		condProgressDao.On("UpdateIfStatusIn", mock.Anything, 1, "true", model.TaskConditionExecutionProgressStatusComplete, eventTime, conditionCompleteFromStatuses).Return(true, nil)
		execDao.On("GetByID", mock.Anything, 50).Return(&model.TaskExecutionProgress{ID: 50, Status: model.TaskExecutionProgressStatusInProgress}, nil)
		taskDao.On("GetByID", mock.Anything, 5).Return(&model.Task{ID: 5, ConditionExpressions: "(1)"}, nil)
		condDao.On("ListByTaskID", mock.Anything, 5).Return([]model.TaskCondition{{ID: 10, No: 1}}, nil)
		condProgressDao.On("ListByTaskExecutionProgressID", mock.Anything, 50).Return([]model.TaskConditionExecutionProgress{
			{TaskConditionID: 10, Status: model.TaskConditionExecutionProgressStatusComplete},
		}, nil)
		execDao.On("UpdateStatusIfIn", mock.Anything, 50, model.TaskExecutionProgressStatusComplete, taskExecutionCompleteFromStatuses).Return(true, nil)
		producer.On("PublishTaskCompleted", mock.Anything, 5, 1, mock.Anything).Return(nil)
		if err := svc.UpdateUserTaskProgress(ctx, 1, 2, "true", eventTime); err != nil {
			t.Fatalf("UpdateUserTaskProgress() error = %v", err)
		}
		producer.AssertExpectations(t)
	})

	t.Run("retry publish when already complete", func(t *testing.T) {
		execDao, condProgressDao, condDao, taskDao, opDao, producer, svc := newUserTaskProgressServiceTestDeps()
		progress := model.TaskConditionExecutionProgress{
			ID: 1, TaskConditionID: 10, TaskExecutionProgressID: 50, TaskID: 5, UserID: 1,
			Status: model.TaskConditionExecutionProgressStatusInProgress,
		}
		condProgressDao.On("ListInProgressByUserAndMetric", mock.Anything, 1, 2).Return([]model.TaskConditionExecutionProgress{progress}, nil)
		condDao.On("GetByID", mock.Anything, 10).Return(&model.TaskCondition{ID: 10, No: 1, DataOperatorID: 3, ConditionValue: "true"}, nil)
		condProgressDao.On("UpdateIfStatusIn", mock.Anything, 1, "true", "", eventTime, activeConditionProgressStatuses).Return(true, nil)
		opDao.On("GetByID", mock.Anything, 3).Return(&model.MetricOperator{ID: 3, Code: "eq"}, nil)
		condProgressDao.On("UpdateIfStatusIn", mock.Anything, 1, "true", model.TaskConditionExecutionProgressStatusComplete, eventTime, conditionCompleteFromStatuses).Return(true, nil)
		execDao.On("GetByID", mock.Anything, 50).Return(&model.TaskExecutionProgress{ID: 50, Status: model.TaskExecutionProgressStatusInProgress}, nil).Once()
		taskDao.On("GetByID", mock.Anything, 5).Return(&model.Task{ID: 5, ConditionExpressions: "(1)"}, nil)
		condDao.On("ListByTaskID", mock.Anything, 5).Return([]model.TaskCondition{{ID: 10, No: 1}}, nil)
		condProgressDao.On("ListByTaskExecutionProgressID", mock.Anything, 50).Return([]model.TaskConditionExecutionProgress{
			{TaskConditionID: 10, Status: model.TaskConditionExecutionProgressStatusComplete},
		}, nil)
		execDao.On("UpdateStatusIfIn", mock.Anything, 50, model.TaskExecutionProgressStatusComplete, taskExecutionCompleteFromStatuses).Return(false, nil)
		execDao.On("GetByID", mock.Anything, 50).Return(&model.TaskExecutionProgress{ID: 50, Status: model.TaskExecutionProgressStatusComplete}, nil).Once()
		producer.On("PublishTaskCompleted", mock.Anything, 5, 1, mock.Anything).Return(nil)
		if err := svc.UpdateUserTaskProgress(ctx, 1, 2, "true", eventTime); err != nil {
			t.Fatalf("UpdateUserTaskProgress() error = %v", err)
		}
	})

	t.Run("publish error", func(t *testing.T) {
		execDao, condProgressDao, condDao, taskDao, opDao, producer, svc := newUserTaskProgressServiceTestDeps()
		progress := model.TaskConditionExecutionProgress{
			ID: 1, TaskConditionID: 10, TaskExecutionProgressID: 50, TaskID: 5, UserID: 1,
			Status: model.TaskConditionExecutionProgressStatusInProgress,
		}
		condProgressDao.On("ListInProgressByUserAndMetric", mock.Anything, 1, 2).Return([]model.TaskConditionExecutionProgress{progress}, nil)
		condDao.On("GetByID", mock.Anything, 10).Return(&model.TaskCondition{ID: 10, No: 1, DataOperatorID: 3, ConditionValue: "true"}, nil)
		condProgressDao.On("UpdateIfStatusIn", mock.Anything, 1, "true", "", eventTime, activeConditionProgressStatuses).Return(true, nil)
		opDao.On("GetByID", mock.Anything, 3).Return(&model.MetricOperator{ID: 3, Code: "eq"}, nil)
		condProgressDao.On("UpdateIfStatusIn", mock.Anything, 1, "true", model.TaskConditionExecutionProgressStatusComplete, eventTime, conditionCompleteFromStatuses).Return(true, nil)
		execDao.On("GetByID", mock.Anything, 50).Return(&model.TaskExecutionProgress{ID: 50, Status: model.TaskExecutionProgressStatusInProgress}, nil)
		taskDao.On("GetByID", mock.Anything, 5).Return(&model.Task{ID: 5, ConditionExpressions: "(1)"}, nil)
		condDao.On("ListByTaskID", mock.Anything, 5).Return([]model.TaskCondition{{ID: 10, No: 1}}, nil)
		condProgressDao.On("ListByTaskExecutionProgressID", mock.Anything, 50).Return([]model.TaskConditionExecutionProgress{
			{TaskConditionID: 10, Status: model.TaskConditionExecutionProgressStatusComplete},
		}, nil)
		execDao.On("UpdateStatusIfIn", mock.Anything, 50, model.TaskExecutionProgressStatusComplete, taskExecutionCompleteFromStatuses).Return(true, nil)
		producer.On("PublishTaskCompleted", mock.Anything, 5, 1, mock.Anything).Return(errors.New("kafka down"))
		err := svc.UpdateUserTaskProgress(ctx, 1, 2, "true", eventTime)
		if err == nil || err.Error() != data.ErrServerError {
			t.Fatalf("UpdateUserTaskProgress() error = %v", err)
		}
	})
}
