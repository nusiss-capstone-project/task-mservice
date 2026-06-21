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

func newTaskServiceTestDeps() (*mocks.TaskGroupDao, *mocks.TaskDao, *mocks.TaskConditionDao, *TaskServiceImpl) {
	groupDao := new(mocks.TaskGroupDao)
	taskDao := new(mocks.TaskDao)
	condDao := new(mocks.TaskConditionDao)
	return groupDao, taskDao, condDao, &TaskServiceImpl{
		taskGroupDao:     groupDao,
		taskDao:          taskDao,
		taskConditionDao: condDao,
	}
}

func validTaskVO() *data.TaskVO {
	return &data.TaskVO{
		Name: "KYC",
		Conditions: []data.TaskConditionVO{
			{MetricID: 1, OperatorID: 1, MetricValue: "true"},
		},
		Expression: "(1)",
	}
}

func TestCreateTask(t *testing.T) {
	initEnv()

	t.Run("success", func(t *testing.T) {
		groupDao, taskDao, condDao, svc := newTaskServiceTestDeps()
		groupDao.On("GetByID", mock.Anything, 1).Return(&model.TaskGroup{ID: 1, Status: model.StatusDraft}, nil)
		taskDao.On("Save", mock.Anything, mock.Anything).Return(10, nil)
		condDao.On("ReplaceByTaskID", mock.Anything, 10, mock.Anything).Return(nil)

		got, err := svc.CreateTask(context.Background(), 1, validTaskVO())
		if err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}
		if got.ID != 10 || got.Name != "KYC" || len(got.Conditions) != 1 {
			t.Fatalf("unexpected task: %+v", got)
		}
	})

	t.Run("nil task", func(t *testing.T) {
		_, _, _, svc := newTaskServiceTestDeps()
		if _, err := svc.CreateTask(context.Background(), 1, nil); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("task group not found", func(t *testing.T) {
		groupDao, _, _, svc := newTaskServiceTestDeps()
		groupDao.On("GetByID", mock.Anything, 99).Return(nil, nil)
		if _, err := svc.CreateTask(context.Background(), 99, validTaskVO()); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("published task group", func(t *testing.T) {
		groupDao, _, _, svc := newTaskServiceTestDeps()
		groupDao.On("GetByID", mock.Anything, 2).Return(&model.TaskGroup{ID: 2, Status: model.StatusPublished}, nil)
		if _, err := svc.CreateTask(context.Background(), 2, validTaskVO()); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("group dao error", func(t *testing.T) {
		groupDao, _, _, svc := newTaskServiceTestDeps()
		groupDao.On("GetByID", mock.Anything, 1).Return(nil, errors.New("db down"))
		if _, err := svc.CreateTask(context.Background(), 1, validTaskVO()); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("save task error", func(t *testing.T) {
		groupDao, taskDao, _, svc := newTaskServiceTestDeps()
		groupDao.On("GetByID", mock.Anything, 1).Return(&model.TaskGroup{ID: 1, Status: model.StatusDraft}, nil)
		taskDao.On("Save", mock.Anything, mock.Anything).Return(0, errors.New("save failed"))
		if _, err := svc.CreateTask(context.Background(), 1, validTaskVO()); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("replace conditions error", func(t *testing.T) {
		groupDao, taskDao, condDao, svc := newTaskServiceTestDeps()
		groupDao.On("GetByID", mock.Anything, 1).Return(&model.TaskGroup{ID: 1, Status: model.StatusDraft}, nil)
		taskDao.On("Save", mock.Anything, mock.Anything).Return(10, nil)
		condDao.On("ReplaceByTaskID", mock.Anything, 10, mock.Anything).Return(errors.New("replace failed"))
		if _, err := svc.CreateTask(context.Background(), 1, validTaskVO()); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestSaveTask(t *testing.T) {
	initEnv()

	t.Run("success", func(t *testing.T) {
		groupDao, taskDao, condDao, svc := newTaskServiceTestDeps()
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
	})

	t.Run("nil task", func(t *testing.T) {
		_, _, _, svc := newTaskServiceTestDeps()
		if _, err := svc.SaveTask(context.Background(), 1, 5, nil); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("published task", func(t *testing.T) {
		_, taskDao, _, svc := newTaskServiceTestDeps()
		taskDao.On("GetByID", mock.Anything, 6).Return(&model.Task{
			ID: 6, TaskGroupID: 1, Status: model.StatusPublished,
		}, nil)
		if _, err := svc.SaveTask(context.Background(), 1, 6, validTaskVO()); err == nil {
			t.Fatal("expected published task error")
		}
	})

	t.Run("task not found", func(t *testing.T) {
		_, taskDao, _, svc := newTaskServiceTestDeps()
		taskDao.On("GetByID", mock.Anything, 99).Return(nil, nil)
		if _, err := svc.SaveTask(context.Background(), 1, 99, validTaskVO()); err == nil {
			t.Fatal("expected not found error")
		}
	})

	t.Run("wrong task group", func(t *testing.T) {
		_, taskDao, _, svc := newTaskServiceTestDeps()
		taskDao.On("GetByID", mock.Anything, 5).Return(&model.Task{
			ID: 5, TaskGroupID: 2, Status: model.StatusDraft,
		}, nil)
		if _, err := svc.SaveTask(context.Background(), 1, 5, validTaskVO()); err == nil {
			t.Fatal("expected not found error")
		}
	})

	t.Run("get task error", func(t *testing.T) {
		_, taskDao, _, svc := newTaskServiceTestDeps()
		taskDao.On("GetByID", mock.Anything, 5).Return(nil, errors.New("db down"))
		if _, err := svc.SaveTask(context.Background(), 1, 5, validTaskVO()); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("published task group", func(t *testing.T) {
		groupDao, taskDao, _, svc := newTaskServiceTestDeps()
		taskDao.On("GetByID", mock.Anything, 5).Return(&model.Task{
			ID: 5, TaskGroupID: 1, Status: model.StatusDraft,
		}, nil)
		groupDao.On("GetByID", mock.Anything, 1).Return(&model.TaskGroup{ID: 1, Status: model.StatusPublished}, nil)
		if _, err := svc.SaveTask(context.Background(), 1, 5, validTaskVO()); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("save error", func(t *testing.T) {
		groupDao, taskDao, _, svc := newTaskServiceTestDeps()
		taskDao.On("GetByID", mock.Anything, 5).Return(&model.Task{
			ID: 5, TaskGroupID: 1, Status: model.StatusDraft,
		}, nil)
		groupDao.On("GetByID", mock.Anything, 1).Return(&model.TaskGroup{ID: 1, Status: model.StatusDraft}, nil)
		taskDao.On("Save", mock.Anything, mock.Anything).Return(0, errors.New("save failed"))
		if _, err := svc.SaveTask(context.Background(), 1, 5, validTaskVO()); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("replace conditions error", func(t *testing.T) {
		groupDao, taskDao, condDao, svc := newTaskServiceTestDeps()
		taskDao.On("GetByID", mock.Anything, 5).Return(&model.Task{
			ID: 5, TaskGroupID: 1, Status: model.StatusDraft,
		}, nil)
		groupDao.On("GetByID", mock.Anything, 1).Return(&model.TaskGroup{ID: 1, Status: model.StatusDraft}, nil)
		taskDao.On("Save", mock.Anything, mock.Anything).Return(5, nil)
		condDao.On("ReplaceByTaskID", mock.Anything, 5, mock.Anything).Return(errors.New("replace failed"))
		if _, err := svc.SaveTask(context.Background(), 1, 5, validTaskVO()); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestPublishTask(t *testing.T) {
	initEnv()

	t.Run("success", func(t *testing.T) {
		_, taskDao, _, svc := newTaskServiceTestDeps()
		taskDao.On("GetByID", mock.Anything, 7).Return(&model.Task{ID: 7, Status: model.StatusDraft}, nil)
		taskDao.On("UpdateStatus", mock.Anything, 7, model.StatusPublished).Return(nil)

		got, err := svc.PublishTask(context.Background(), 7)
		if err != nil {
			t.Fatalf("PublishTask() error = %v", err)
		}
		if got.Status != model.StatusPublished {
			t.Fatalf("status = %s", got.Status)
		}
	})

	t.Run("already published", func(t *testing.T) {
		_, taskDao, _, svc := newTaskServiceTestDeps()
		taskDao.On("GetByID", mock.Anything, 12).Return(&model.Task{ID: 12, Status: model.StatusPublished}, nil)

		got, err := svc.PublishTask(context.Background(), 12)
		if err != nil {
			t.Fatalf("PublishTask() error = %v", err)
		}
		if got.Status != model.StatusPublished {
			t.Fatalf("status = %s", got.Status)
		}
	})

	t.Run("get error", func(t *testing.T) {
		_, taskDao, _, svc := newTaskServiceTestDeps()
		taskDao.On("GetByID", mock.Anything, 8).Return(nil, errors.New("db down"))
		if _, err := svc.PublishTask(context.Background(), 8); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, taskDao, _, svc := newTaskServiceTestDeps()
		taskDao.On("GetByID", mock.Anything, 9).Return(nil, nil)
		if _, err := svc.PublishTask(context.Background(), 9); err == nil {
			t.Fatal("expected not found")
		}
	})

	t.Run("update not found", func(t *testing.T) {
		_, taskDao, _, svc := newTaskServiceTestDeps()
		taskDao.On("GetByID", mock.Anything, 11).Return(&model.Task{ID: 11, Status: model.StatusDraft}, nil)
		taskDao.On("UpdateStatus", mock.Anything, 11, model.StatusPublished).Return(gorm.ErrRecordNotFound)
		if _, err := svc.PublishTask(context.Background(), 11); err == nil {
			t.Fatal("expected not found from update")
		}
	})

	t.Run("update error", func(t *testing.T) {
		_, taskDao, _, svc := newTaskServiceTestDeps()
		taskDao.On("GetByID", mock.Anything, 13).Return(&model.Task{ID: 13, Status: model.StatusDraft}, nil)
		taskDao.On("UpdateStatus", mock.Anything, 13, model.StatusPublished).Return(errors.New("update failed"))
		if _, err := svc.PublishTask(context.Background(), 13); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestGetTaskDetail(t *testing.T) {
	initEnv()

	t.Run("success", func(t *testing.T) {
		_, taskDao, condDao, svc := newTaskServiceTestDeps()
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
	})

	t.Run("not found", func(t *testing.T) {
		_, taskDao, _, svc := newTaskServiceTestDeps()
		taskDao.On("GetByID", mock.Anything, 99).Return(nil, nil)
		got, err := svc.GetTaskDetail(context.Background(), 1, 99)
		if err != nil {
			t.Fatalf("GetTaskDetail() error = %v", err)
		}
		if got != nil {
			t.Fatalf("expected nil, got %+v", got)
		}
	})

	t.Run("wrong group", func(t *testing.T) {
		_, taskDao, _, svc := newTaskServiceTestDeps()
		taskDao.On("GetByID", mock.Anything, 3).Return(&model.Task{
			ID: 3, TaskGroupID: 2, Name: "Detail",
		}, nil)
		got, err := svc.GetTaskDetail(context.Background(), 1, 3)
		if err != nil {
			t.Fatalf("GetTaskDetail() error = %v", err)
		}
		if got != nil {
			t.Fatalf("expected nil, got %+v", got)
		}
	})

	t.Run("get error", func(t *testing.T) {
		_, taskDao, _, svc := newTaskServiceTestDeps()
		taskDao.On("GetByID", mock.Anything, 3).Return(nil, errors.New("db down"))
		if _, err := svc.GetTaskDetail(context.Background(), 1, 3); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("list conditions error", func(t *testing.T) {
		_, taskDao, condDao, svc := newTaskServiceTestDeps()
		taskDao.On("GetByID", mock.Anything, 3).Return(&model.Task{
			ID: 3, TaskGroupID: 1, Name: "Detail",
		}, nil)
		condDao.On("ListByTaskID", mock.Anything, 3).Return(nil, errors.New("db down"))
		if _, err := svc.GetTaskDetail(context.Background(), 1, 3); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestListTasksByGroupID(t *testing.T) {
	initEnv()

	t.Run("success", func(t *testing.T) {
		groupDao, taskDao, _, svc := newTaskServiceTestDeps()
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
	})

	t.Run("empty list", func(t *testing.T) {
		groupDao, taskDao, _, svc := newTaskServiceTestDeps()
		groupDao.On("GetByID", mock.Anything, 1).Return(&model.TaskGroup{ID: 1}, nil)
		taskDao.On("ListByGroupID", mock.Anything, 1).Return([]model.Task{}, nil)

		got, err := svc.ListTasksByGroupID(context.Background(), 1)
		if err != nil {
			t.Fatalf("ListTasksByGroupID() error = %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("expected empty list, got %+v", got)
		}
	})

	t.Run("group not found", func(t *testing.T) {
		groupDao, _, _, svc := newTaskServiceTestDeps()
		groupDao.On("GetByID", mock.Anything, 99).Return(nil, nil)
		if _, err := svc.ListTasksByGroupID(context.Background(), 99); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("group get error", func(t *testing.T) {
		groupDao, _, _, svc := newTaskServiceTestDeps()
		groupDao.On("GetByID", mock.Anything, 1).Return(nil, errors.New("db down"))
		if _, err := svc.ListTasksByGroupID(context.Background(), 1); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("list error", func(t *testing.T) {
		groupDao, taskDao, _, svc := newTaskServiceTestDeps()
		groupDao.On("GetByID", mock.Anything, 1).Return(&model.TaskGroup{ID: 1}, nil)
		taskDao.On("ListByGroupID", mock.Anything, 1).Return(nil, errors.New("db down"))
		if _, err := svc.ListTasksByGroupID(context.Background(), 1); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestValidateTaskInput(t *testing.T) {
	if err := validateTaskInput(&data.TaskVO{}); err == nil {
		t.Fatal("expected name required error")
	}
	if err := validateTaskInput(&data.TaskVO{Name: "x"}); err == nil {
		t.Fatal("expected missing conditions error")
	}
	if err := validateTaskInput(&data.TaskVO{
		Name: "x",
		Conditions: []data.TaskConditionVO{{MetricID: 0, OperatorID: 1, MetricValue: "1"}},
	}); err == nil {
		t.Fatal("expected invalid condition error")
	}
	if err := validateTaskInput(validTaskVO()); err != nil {
		t.Fatalf("expected valid input, got %v", err)
	}
}

func TestToTaskConditions(t *testing.T) {
	got := toTaskConditions(5, []data.TaskConditionVO{
		{MetricID: 1, OperatorID: 1, MetricValue: "a"},
		{No: 3, MetricID: 2, OperatorID: 2, MetricValue: "b"},
	})
	if len(got) != 2 {
		t.Fatalf("len = %d", len(got))
	}
	if got[0].No != 1 || got[0].TaskID != 5 {
		t.Fatalf("first condition = %+v", got[0])
	}
	if got[1].No != 3 {
		t.Fatalf("second condition no = %d", got[1].No)
	}
}

func TestBuildTaskVOLoadsConditions(t *testing.T) {
	initEnv()
	_, taskDao, condDao, svc := newTaskServiceTestDeps()
	task := &model.Task{ID: 20, TaskGroupID: 1, Name: "Load", ConditionExpressions: "1"}
	condDao.On("ListByTaskID", mock.Anything, 20).Return([]model.TaskCondition{
		{No: 1, DataMetricID: 1, DataOperatorID: 1, ConditionValue: "x"},
	}, nil)

	got, err := svc.buildTaskVO(context.Background(), task, nil)
	if err != nil {
		t.Fatalf("buildTaskVO() error = %v", err)
	}
	if len(got.Conditions) != 1 {
		t.Fatalf("conditions = %+v", got.Conditions)
	}
	_ = taskDao
}

func TestBuildTaskVOListError(t *testing.T) {
	initEnv()
	_, _, condDao, svc := newTaskServiceTestDeps()
	task := &model.Task{ID: 21, TaskGroupID: 1, Name: "Fail"}
	condDao.On("ListByTaskID", mock.Anything, 21).Return(nil, errors.New("db down"))

	if _, err := svc.buildTaskVO(context.Background(), task, nil); err == nil {
		t.Fatal("expected error")
	}
}
