package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/nusiss-capstone-project/task-mservice/server/http/data"
	"github.com/nusiss-capstone-project/task-mservice/server/repository/dao/mocks"
	"github.com/nusiss-capstone-project/task-mservice/server/repository/model"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

func TestGetTaskService(t *testing.T) {
	initEnv()
	s1 := GetTaskService()
	s2 := GetTaskService()
	if s1 != s2 {
		t.Fatal("expected singleton instance")
	}
}

func TestCreateTask(t *testing.T) {
	initEnv()
	groupDao := new(mocks.TaskGroupDao)
	taskDao := new(mocks.TaskDao)
	condDao := new(mocks.TaskConditionDao)
	svc := &TaskServiceImpl{
		taskGroupDao:     groupDao,
		taskDao:          taskDao,
		taskConditionDao: condDao,
	}

	groupDao.On("GetByID", mock.Anything, 1).Return(&model.TaskGroup{ID: 1, Status: model.StatusDraft}, nil)
	taskDao.On("Save", mock.Anything, mock.Anything).Return(10, nil)
	condDao.On("ReplaceByTaskID", mock.Anything, 10, mock.Anything).Return(nil)

	got, err := svc.CreateTask(context.Background(), 1, &data.TaskVO{
		Name: "KYC",
		Conditions: []data.TaskConditionVO{
			{MetricID: 1, OperatorID: 1, MetricValue: "true"},
		},
		Expression: "(1)",
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if got.ID != 10 || got.Name != "KYC" || len(got.Conditions) != 1 {
		t.Fatalf("unexpected task: %+v", got)
	}
}

func TestSaveTask(t *testing.T) {
	initEnv()
	groupDao := new(mocks.TaskGroupDao)
	taskDao := new(mocks.TaskDao)
	condDao := new(mocks.TaskConditionDao)
	svc := &TaskServiceImpl{
		taskGroupDao:     groupDao,
		taskDao:          taskDao,
		taskConditionDao: condDao,
	}

	taskDao.On("GetByID", mock.Anything, 5).Return(&model.Task{
		ID: 5, TaskGroupID: 1, Status: model.StatusDraft,
	}, nil)
	groupDao.On("GetByID", mock.Anything, 1).Return(&model.TaskGroup{ID: 1, Status: model.StatusDraft}, nil)
	taskDao.On("Save", mock.Anything, mock.Anything).Return(5, nil)
	condDao.On("ReplaceByTaskID", mock.Anything, 5, mock.Anything).Return(nil)

	got, err := svc.SaveTask(context.Background(), 1, 5, &data.TaskVO{
		Name: "Updated",
		Conditions: []data.TaskConditionVO{
			{MetricID: 1, OperatorID: 1, MetricValue: "100"},
		},
		Expression: "1",
	})
	if err != nil {
		t.Fatalf("SaveTask() error = %v", err)
	}
	if got.Name != "Updated" {
		t.Fatalf("name = %s", got.Name)
	}

	taskDao.On("GetByID", mock.Anything, 6).Return(&model.Task{
		ID: 6, TaskGroupID: 1, Status: model.StatusPublished,
	}, nil)
	if _, err := svc.SaveTask(context.Background(), 1, 6, &data.TaskVO{
		Name: "Blocked",
		Conditions: []data.TaskConditionVO{
			{MetricID: 1, OperatorID: 1, MetricValue: "x"},
		},
	}); err == nil {
		t.Fatal("expected published task error")
	}
}

func TestPublishTask(t *testing.T) {
	initEnv()
	taskDao := new(mocks.TaskDao)
	svc := &TaskServiceImpl{taskDao: taskDao}

	taskDao.On("GetByID", mock.Anything, 7).Return(&model.Task{ID: 7, Status: model.StatusDraft}, nil)
	taskDao.On("UpdateStatus", mock.Anything, 7, model.StatusPublished).Return(nil)

	got, err := svc.PublishTask(context.Background(), 7)
	if err != nil {
		t.Fatalf("PublishTask() error = %v", err)
	}
	if got.Status != model.StatusPublished {
		t.Fatalf("status = %s", got.Status)
	}

	taskDao.On("GetByID", mock.Anything, 8).Return(nil, errors.New("db down"))
	if _, err := svc.PublishTask(context.Background(), 8); err == nil {
		t.Fatal("expected error")
	}

	taskDao.On("GetByID", mock.Anything, 9).Return(nil, nil)
	if _, err := svc.PublishTask(context.Background(), 9); err == nil {
		t.Fatal("expected not found")
	}

	taskDao.On("GetByID", mock.Anything, 11).Return(&model.Task{ID: 11, Status: model.StatusDraft}, nil)
	taskDao.On("UpdateStatus", mock.Anything, 11, model.StatusPublished).Return(gorm.ErrRecordNotFound)
	if _, err := svc.PublishTask(context.Background(), 11); err == nil {
		t.Fatal("expected not found from update")
	}
}

func TestGetTaskDetail(t *testing.T) {
	initEnv()
	taskDao := new(mocks.TaskDao)
	condDao := new(mocks.TaskConditionDao)
	svc := &TaskServiceImpl{taskDao: taskDao, taskConditionDao: condDao}

	taskDao.On("GetByID", mock.Anything, 3).Return(&model.Task{
		ID: 3, TaskGroupID: 1, Name: "Detail", ConditionExpressions: "1",
	}, nil)
	condDao.On("ListByTaskID", mock.Anything, 3).Return([]model.TaskCondition{
		{No: 1, DataMetricID: 1, DataOperatorID: 1, ConditionValue: "true"},
	}, nil)

	got, err := svc.GetTaskDetail(context.Background(), 1, 3)
	if err != nil {
		t.Fatalf("GetTaskDetail() error = %v", err)
	}
	want := &data.TaskVO{
		ID: 3, Name: "Detail", TaskGroupID: 1, Expression: "1",
		Conditions: []data.TaskConditionVO{
			{No: 1, MetricID: 1, OperatorID: 1, MetricValue: "true"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestValidateTaskInput(t *testing.T) {
	if err := validateTaskInput(&data.TaskVO{Name: "x"}); err == nil {
		t.Fatal("expected missing conditions error")
	}
	if err := validateTaskInput(&data.TaskVO{
		Name: "x",
		Conditions: []data.TaskConditionVO{{MetricID: 0, OperatorID: 1, MetricValue: "1"}},
	}); err == nil {
		t.Fatal("expected invalid condition error")
	}
}

func TestListTasksByGroupID(t *testing.T) {
	initEnv()
	groupDao := new(mocks.TaskGroupDao)
	taskDao := new(mocks.TaskDao)
	svc := &TaskServiceImpl{taskGroupDao: groupDao, taskDao: taskDao}

	groupDao.On("GetByID", mock.Anything, 1).Return(&model.TaskGroup{ID: 1}, nil)
	taskDao.On("ListByGroupID", mock.Anything, 1).Return([]model.Task{
		{ID: 1, TaskGroupID: 1, Name: "T1", ConditionExpressions: "1"},
	}, nil)

	got, err := svc.ListTasksByGroupID(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListTasksByGroupID() error = %v", err)
	}
	if len(got) != 1 || got[0].Name != "T1" {
		t.Fatalf("unexpected list: %+v", got)
	}
}
