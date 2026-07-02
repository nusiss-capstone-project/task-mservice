package grpc

import (
	"context"

	"github.com/nusiss-capstone-project/task-mservice/common/taskpb"
	"github.com/nusiss-capstone-project/task-mservice/server/log"
	"github.com/nusiss-capstone-project/task-mservice/server/service"
)

type TaskService struct {
	taskpb.UnimplementedTaskServiceServer
}

func (s *TaskService) SayHello(ctx context.Context, in *taskpb.HelloRequest) (*taskpb.HelloResponse, error) {
	log.WithContext(ctx).Infof("Received: %v", in.GetName())
	return &taskpb.HelloResponse{Message: "Hello " + in.GetName()}, nil
}

func (s *TaskService) EnrollTask(ctx context.Context, in *taskpb.EnrollTaskRequest) (*taskpb.EnrollTaskResponse, error) {
	log.WithContext(ctx).Infof("enroll task request user_id=%d task_id=%d", in.GetUserId(), in.GetTaskId())
	return service.GetUserTaskProgressService().EnrollTask(ctx, in)
}
