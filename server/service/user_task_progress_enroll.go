package service

import (
	"context"
	"strings"

	"github.com/nusiss-capstone-project/task-mservice/common/taskpb"
	"github.com/nusiss-capstone-project/task-mservice/server/http/data"
	"github.com/nusiss-capstone-project/task-mservice/server/log"
	"github.com/nusiss-capstone-project/task-mservice/server/repository/model"
)

func validateEnrollTaskRequest(req *taskpb.EnrollTaskRequest) (userID, taskID int, ok bool) {
	if req == nil || req.GetUserId() <= 0 || req.GetTaskId() <= 0 {
		return 0, 0, false
	}
	return int(req.GetUserId()), int(req.GetTaskId()), true
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

func (s *userTaskProgressServiceImpl) createEnrollment(
	ctx context.Context,
	userID, taskID int,
	conditions []model.TaskCondition,
) *taskpb.EnrollTaskResponse {
	executionID, conditionProgressIDs, err := s.taskExecutionProgressDao.EnrollUserTask(ctx, userID, taskID, conditions)
	if err != nil {
		if isDuplicateEntryError(err) {
			log.WithContext(ctx).Infof("user %d already enrolled in task %d", userID, taskID)
			return enrollTaskFail(taskpb.ErrorCode_INVALID_PARAM, data.ErrInvalidInput)
		}
		log.WithContext(ctx).Errorf("enroll user %d task %d: %v", userID, taskID, err)
		return enrollTaskFail(taskpb.ErrorCode_UNKNOWN_ERROR, data.ErrServerError)
	}
	return enrollTaskSuccess(executionID, conditionProgressIDs)
}

func enrollTaskBase(code taskpb.ErrorCode, message string) *taskpb.BaseInfo {
	return &taskpb.BaseInfo{
		Code:    code,
		Message: message,
	}
}

func enrollTaskSuccess(executionID int, conditionProgressIDs []int) *taskpb.EnrollTaskResponse {
	return &taskpb.EnrollTaskResponse{
		Base: enrollTaskBase(taskpb.ErrorCode_OK, "ok"),
		Data: &taskpb.EnrollTaskData{
			TaskExecutionProgressId:           int64(executionID),
			TaskConditionExecutionProgressIds: toInt64Slice(conditionProgressIDs),
		},
	}
}

func enrollTaskFail(code taskpb.ErrorCode, message string) *taskpb.EnrollTaskResponse {
	return &taskpb.EnrollTaskResponse{
		Base: enrollTaskBase(code, message),
	}
}

func isDuplicateEntryError(err error) bool {
	return strings.Contains(err.Error(), "Duplicate entry")
}

func toInt64Slice(values []int) []int64 {
	result := make([]int64, len(values))
	for i, v := range values {
		result[i] = int64(v)
	}
	return result
}
