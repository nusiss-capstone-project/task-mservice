package service

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/expr-lang/expr"
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
