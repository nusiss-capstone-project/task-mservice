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
)

func TestGetReferenceService(t *testing.T) {
	initEnv()
	s1 := GetReferenceService()
	s2 := GetReferenceService()
	if s1 != s2 {
		t.Fatal("expected singleton instance")
	}
}

func TestListDataMetrics(t *testing.T) {
	initEnv()
	metricDao := new(mocks.DataMetricDao)
	svc := &ReferenceServiceImpl{dataMetricDao: metricDao}

	t.Run("success", func(t *testing.T) {
		metricDao.On("List", mock.Anything).Return([]model.DataMetric{
			{ID: 1, Code: "net_deposit_volume"},
			{ID: 2, Code: "kyc_completed"},
		}, nil).Once()

		got, err := svc.ListDataMetrics(context.Background())
		if err != nil {
			t.Fatalf("ListDataMetrics() error = %v", err)
		}
		want := []data.DataMetricVO{
			{ID: 1, Code: "net_deposit_volume"},
			{ID: 2, Code: "kyc_completed"},
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	})

	t.Run("empty list", func(t *testing.T) {
		metricDao.On("List", mock.Anything).Return([]model.DataMetric{}, nil).Once()

		got, err := svc.ListDataMetrics(context.Background())
		if err != nil {
			t.Fatalf("ListDataMetrics() error = %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("expected empty list, got %+v", got)
		}
	})

	t.Run("dao error", func(t *testing.T) {
		metricDao.On("List", mock.Anything).Return(nil, errors.New("db down")).Once()

		if _, err := svc.ListDataMetrics(context.Background()); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestListMetricOperators(t *testing.T) {
	initEnv()
	opDao := new(mocks.MetricOperatorDao)
	svc := &ReferenceServiceImpl{metricOperatorDao: opDao}

	t.Run("success", func(t *testing.T) {
		opDao.On("List", mock.Anything).Return([]model.MetricOperator{
			{ID: 1, Code: "lt", Display: "is less than"},
			{ID: 2, Code: "gt", Display: "is greater than"},
		}, nil).Once()

		got, err := svc.ListMetricOperators(context.Background())
		if err != nil {
			t.Fatalf("ListMetricOperators() error = %v", err)
		}
		want := []data.MetricOperatorVO{
			{ID: 1, Code: "lt", Display: "is less than"},
			{ID: 2, Code: "gt", Display: "is greater than"},
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	})

	t.Run("empty list", func(t *testing.T) {
		opDao.On("List", mock.Anything).Return([]model.MetricOperator{}, nil).Once()

		got, err := svc.ListMetricOperators(context.Background())
		if err != nil {
			t.Fatalf("ListMetricOperators() error = %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("expected empty list, got %+v", got)
		}
	})

	t.Run("dao error", func(t *testing.T) {
		opDao.On("List", mock.Anything).Return(nil, errors.New("db down")).Once()

		if _, err := svc.ListMetricOperators(context.Background()); err == nil {
			t.Fatal("expected error")
		}
	})
}
