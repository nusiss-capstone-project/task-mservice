package service

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/expr-lang/expr"
	"github.com/nusiss-capstone-project/task-mservice/common/taskpb"
	"github.com/nusiss-capstone-project/task-mservice/server/repository/model"
)

var (
	activeConditionProgressStatuses = []string{
		model.TaskConditionExecutionProgressStatusInit,
		model.TaskConditionExecutionProgressStatusInProgress,
	}
	conditionCompleteFromStatuses = []string{
		model.TaskConditionExecutionProgressStatusInit,
		model.TaskConditionExecutionProgressStatusInProgress,
	}
	taskExecutionCompleteFromStatuses = []string{
		model.TaskExecutionProgressStatusInit,
		model.TaskExecutionProgressStatusInProgress,
	}
)

func validateEnrollTaskRequest(req *taskpb.EnrollTaskRequest) (userID, taskID int, ok bool) {
	if req == nil || req.GetUserId() <= 0 || req.GetTaskId() <= 0 {
		return 0, 0, false
	}
	return int(req.GetUserId()), int(req.GetTaskId()), true
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

func formatEventTime(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339Nano)
}

func isStaleEvent(eventTime time.Time, lastEventTime *time.Time) bool {
	if lastEventTime == nil || lastEventTime.IsZero() {
		return false
	}
	// Same eventTime is allowed for retry; only strictly older events are stale.
	return eventTime.Before(*lastEventTime)
}

func isTerminalConditionProgressStatus(status string) bool {
	return status == model.TaskConditionExecutionProgressStatusComplete ||
		status == model.TaskConditionExecutionProgressStatusExpired
}

func isTerminalTaskExecutionProgressStatus(status string) bool {
	return status == model.TaskExecutionProgressStatusComplete ||
		status == model.TaskExecutionProgressStatusExpired
}

func canTransitionConditionProgressToComplete(currentStatus string) bool {
	for _, status := range conditionCompleteFromStatuses {
		if currentStatus == status {
			return true
		}
	}
	return false
}

func buildConditionCompletionByNo(
	conditions []model.TaskCondition,
	progresses []model.TaskConditionExecutionProgress,
) map[int]bool {
	conditionIDToNo := make(map[int]int, len(conditions))
	completedByNo := make(map[int]bool, len(conditions))
	for _, condition := range conditions {
		conditionIDToNo[condition.ID] = condition.No
		completedByNo[condition.No] = false
	}
	for _, progress := range progresses {
		no, ok := conditionIDToNo[progress.TaskConditionID]
		if !ok {
			continue
		}
		if progress.Status == model.TaskConditionExecutionProgressStatusComplete {
			completedByNo[no] = true
		}
	}
	return completedByNo
}

var conditionNumberPattern = regexp.MustCompile(`\d+`)

func evaluateTaskExpression(expression string, completedByNo map[int]bool) (bool, error) {
	exprStr := strings.TrimSpace(expression)
	if exprStr == "" {
		return false, fmt.Errorf("task expression is empty")
	}

	env := make(map[string]any)
	var replaceErr error
	normalized := conditionNumberPattern.ReplaceAllStringFunc(exprStr, func(token string) string {
		if replaceErr != nil {
			return token
		}
		no, err := strconv.Atoi(token)
		if err != nil {
			replaceErr = fmt.Errorf("invalid condition number %q: %w", token, err)
			return token
		}
		completed, ok := completedByNo[no]
		if !ok {
			replaceErr = fmt.Errorf("condition number %d not found", no)
			return token
		}
		key := fmt.Sprintf("c%d", no)
		env[key] = completed
		return key
	})
	if replaceErr != nil {
		return false, replaceErr
	}

	normalized = normalizeLogicalOperators(normalized)

	program, err := expr.Compile(normalized, expr.Env(env), expr.AsBool())
	if err != nil {
		return false, fmt.Errorf("compile task expression: %w", err)
	}
	result, err := expr.Run(program, env)
	if err != nil {
		return false, fmt.Errorf("run task expression: %w", err)
	}
	value, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("task expression result is not bool")
	}
	return value, nil
}

// normalizeLogicalOperators converts task DSL operators (&, |) to expr-lang syntax (&&, ||).
func normalizeLogicalOperators(expr string) string {
	expr = strings.ReplaceAll(expr, "||", "\x00")
	expr = strings.ReplaceAll(expr, "|", "||")
	expr = strings.ReplaceAll(expr, "\x00", "||")

	expr = strings.ReplaceAll(expr, "&&", "\x01")
	expr = strings.ReplaceAll(expr, "&", "&&")
	expr = strings.ReplaceAll(expr, "\x01", "&&")
	return expr
}

type metricOperatorFunc func(currentValue, targetValue string) (bool, error)

var metricOperatorEvaluators = map[string]metricOperatorFunc{
	"eq":     evalEq,
	"neq":    evalNeq,
	"gt":     numericOp(func(a, b float64) bool { return a > b }),
	"lt":     numericOp(func(a, b float64) bool { return a < b }),
	"ge":     numericOp(func(a, b float64) bool { return a >= b }),
	"le":     numericOp(func(a, b float64) bool { return a <= b }),
	"in":     evalIn,
	"not_in": evalNotIn,
}

func evaluateMetricOperator(operatorCode, currentValue, targetValue string) (bool, error) {
	eval, ok := metricOperatorEvaluators[operatorCode]
	if !ok {
		return false, fmt.Errorf("unsupported operator code %q", operatorCode)
	}
	return eval(currentValue, targetValue)
}

func evalEq(currentValue, targetValue string) (bool, error) {
	return currentValue == targetValue, nil
}

func evalNeq(currentValue, targetValue string) (bool, error) {
	return currentValue != targetValue, nil
}

func numericOp(cmp func(float64, float64) bool) metricOperatorFunc {
	return func(currentValue, targetValue string) (bool, error) {
		current, err := parseFloatValue(currentValue)
		if err != nil {
			return false, fmt.Errorf("parse current value %q: %w", currentValue, err)
		}
		target, err := parseFloatValue(targetValue)
		if err != nil {
			return false, fmt.Errorf("parse target value %q: %w", targetValue, err)
		}
		return cmp(current, target), nil
	}
}

func parseFloatValue(value string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(value), 64)
}

func parseTargetList(targetValue string) []string {
	parts := strings.Split(targetValue, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		result = append(result, part)
	}
	return result
}

func evalIn(currentValue, targetValue string) (bool, error) {
	for _, item := range parseTargetList(targetValue) {
		if currentValue == item {
			return true, nil
		}
	}
	return false, nil
}

func evalNotIn(currentValue, targetValue string) (bool, error) {
	for _, item := range parseTargetList(targetValue) {
		if currentValue == item {
			return false, nil
		}
	}
	return true, nil
}
