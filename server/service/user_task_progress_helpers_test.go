package service

import (
	"testing"
	"time"

	"github.com/nusiss-capstone-project/task-mservice/common/taskpb"
	"github.com/nusiss-capstone-project/task-mservice/server/repository/model"
)

func TestIsStaleEvent(t *testing.T) {
	lastEventTime := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	if !isStaleEvent(lastEventTime.Add(-time.Minute), &lastEventTime) {
		t.Fatal("expected stale event")
	}
	if isStaleEvent(lastEventTime, &lastEventTime) {
		t.Fatal("same timestamp should not be stale (retry allowed)")
	}
	if isStaleEvent(lastEventTime.Add(time.Minute), &lastEventTime) {
		t.Fatal("newer event should not be stale")
	}
	if isStaleEvent(time.Now(), nil) {
		t.Fatal("nil last event time should not be stale")
	}
}

func TestEvaluateMetricOperator(t *testing.T) {
	cases := []struct {
		op      string
		current string
		target  string
		want    bool
	}{
		{"eq", "true", "true", true},
		{"neq", "true", "false", true},
		{"gt", "100", "50", true},
		{"lt", "10", "50", true},
		{"ge", "50", "50", true},
		{"le", "40", "50", true},
		{"in", "gold", "silver,gold,bronze", true},
		{"not_in", "gold", "silver,bronze", true},
		{"not_in", "gold", "gold,silver", false},
	}
	for _, tc := range cases {
		got, err := evaluateMetricOperator(tc.op, tc.current, tc.target)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.op, err)
		}
		if got != tc.want {
			t.Fatalf("%s(%q, %q) = %v, want %v", tc.op, tc.current, tc.target, got, tc.want)
		}
	}

	_, err := evaluateMetricOperator("unknown", "1", "2")
	if err == nil {
		t.Fatal("expected error for unknown operator")
	}
}

func TestEvaluateTaskExpression(t *testing.T) {
	completed := map[int]bool{1: true, 2: true, 3: false}

	ok, err := evaluateTaskExpression("(1&2)", completed)
	if err != nil || !ok {
		t.Fatalf("(1&2) = %v, err=%v", ok, err)
	}

	ok, err = evaluateTaskExpression("(1&2)|3", completed)
	if err != nil || !ok {
		t.Fatalf("(1&2)|3 = %v, err=%v", ok, err)
	}

	ok, err = evaluateTaskExpression("1&3", completed)
	if err != nil || ok {
		t.Fatalf("1&3 = %v, err=%v", ok, err)
	}

	_, err = evaluateTaskExpression("", completed)
	if err == nil {
		t.Fatal("expected error for empty expression")
	}
}

func TestBuildConditionCompletionByNo(t *testing.T) {
	conditions := []model.TaskCondition{
		{ID: 11, No: 1},
		{ID: 12, No: 2},
	}
	progresses := []model.TaskConditionExecutionProgress{
		{TaskConditionID: 11, Status: model.TaskConditionExecutionProgressStatusComplete},
		{TaskConditionID: 12, Status: model.TaskConditionExecutionProgressStatusInProgress},
	}
	completed := buildConditionCompletionByNo(conditions, progresses)
	if !completed[1] || completed[2] {
		t.Fatalf("unexpected completion map: %+v", completed)
	}
}

func TestTerminalStatuses(t *testing.T) {
	if !isTerminalConditionProgressStatus(model.TaskConditionExecutionProgressStatusComplete) {
		t.Fatal("complete should be terminal")
	}
	if isTerminalConditionProgressStatus(model.TaskConditionExecutionProgressStatusInProgress) {
		t.Fatal("in progress should not be terminal")
	}
	if !isTerminalTaskExecutionProgressStatus(model.TaskExecutionProgressStatusComplete) {
		t.Fatal("complete task execution should be terminal")
	}
	if !canTransitionConditionProgressToComplete(model.TaskConditionExecutionProgressStatusInProgress) {
		t.Fatal("in progress should allow transition to complete")
	}
	if canTransitionConditionProgressToComplete(model.TaskConditionExecutionProgressStatusComplete) {
		t.Fatal("complete should not allow transition to complete")
	}
}

func TestValidateEnrollTaskRequest(t *testing.T) {
	userID, taskID, ok := validateEnrollTaskRequest(&taskpb.EnrollTaskRequest{UserId: 1, TaskId: 2})
	if !ok || userID != 1 || taskID != 2 {
		t.Fatalf("unexpected result: ok=%v user=%d task=%d", ok, userID, taskID)
	}
	if _, _, ok := validateEnrollTaskRequest(nil); ok {
		t.Fatal("nil request should be invalid")
	}
	if _, _, ok := validateEnrollTaskRequest(&taskpb.EnrollTaskRequest{UserId: 0, TaskId: 1}); ok {
		t.Fatal("zero user id should be invalid")
	}
}
