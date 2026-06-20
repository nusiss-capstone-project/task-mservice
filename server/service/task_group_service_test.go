package service

import (
	"context"
	"testing"

	"github.com/nusiss-capstone-project/task-mservice/server/http/data"
	"github.com/nusiss-capstone-project/task-mservice/server/repository/dao/mocks"
	"github.com/nusiss-capstone-project/task-mservice/server/repository/model"
	"github.com/stretchr/testify/mock"
)

func TestGetTaskGroupService(t *testing.T) {
	initEnv()
	s1 := GetTaskGroupService()
	s2 := GetTaskGroupService()
	if s1 != s2 {
		t.Fatal("expected singleton instance")
	}
}

func TestSaveTaskGroup(t *testing.T) {
	initEnv()
	daoMock := new(mocks.TaskGroupDao)
	svc := &TaskGroupServiceImpl{taskGroupDao: daoMock}

	tests := []struct {
		name    string
		vo      *data.TaskGroupVO
		setup   func()
		wantErr bool
	}{
		{
			name:    "nil input",
			vo:      nil,
			wantErr: true,
		},
		{
			name: "create draft group",
			vo:   &data.TaskGroupVO{Name: "Campaign Tasks"},
			setup: func() {
				daoMock.On("Save", mock.Anything, mock.MatchedBy(func(g *model.TaskGroup) bool {
					return g.Name == "Campaign Tasks" && g.Status == model.StatusDraft
				})).Return(1, nil).Once()
			},
			wantErr: false,
		},
		{
			name: "update draft group",
			vo:   &data.TaskGroupVO{ID: 2, Name: "Updated"},
			setup: func() {
				daoMock.On("GetByID", mock.Anything, 2).Return(&model.TaskGroup{
					ID: 2, Name: "Old", Status: model.StatusDraft,
				}, nil).Once()
				daoMock.On("Save", mock.Anything, mock.Anything).Return(2, nil).Once()
			},
			wantErr: false,
		},
		{
			name: "reject published group update",
			vo:   &data.TaskGroupVO{ID: 3, Name: "Blocked"},
			setup: func() {
				daoMock.On("GetByID", mock.Anything, 3).Return(&model.TaskGroup{
					ID: 3, Status: model.StatusPublished,
				}, nil).Once()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			got, err := svc.SaveTaskGroup(context.Background(), tt.vo)
			if (err != nil) != tt.wantErr {
				t.Fatalf("SaveTaskGroup() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got == nil {
				t.Fatal("expected result")
			}
		})
	}
}

func TestPublishTaskGroup(t *testing.T) {
	initEnv()
	daoMock := new(mocks.TaskGroupDao)
	svc := &TaskGroupServiceImpl{taskGroupDao: daoMock}

	daoMock.On("GetByID", mock.Anything, 1).Return(&model.TaskGroup{ID: 1, Status: model.StatusDraft}, nil)
	daoMock.On("UpdateStatus", mock.Anything, 1, model.StatusPublished).Return(nil)

	got, err := svc.PublishTaskGroup(context.Background(), 1)
	if err != nil {
		t.Fatalf("PublishTaskGroup() error = %v", err)
	}
	if got.Status != model.StatusPublished {
		t.Fatalf("status = %s", got.Status)
	}

	daoMock.On("GetByID", mock.Anything, 99).Return(nil, nil)
	if _, err := svc.PublishTaskGroup(context.Background(), 99); err == nil {
		t.Fatal("expected not found error")
	}
}

func TestListTaskGroups(t *testing.T) {
	initEnv()
	daoMock := new(mocks.TaskGroupDao)
	svc := &TaskGroupServiceImpl{taskGroupDao: daoMock}

	daoMock.On("List", mock.Anything).Return([]model.TaskGroup{
		{ID: 1, Name: "G1", Status: model.StatusDraft},
	}, nil)

	got, err := svc.ListTaskGroups(context.Background())
	if err != nil {
		t.Fatalf("ListTaskGroups() error = %v", err)
	}
	if len(got) != 1 || got[0].Name != "G1" {
		t.Fatalf("unexpected list: %+v", got)
	}
}
