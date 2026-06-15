package grpc

import (
	"context"

	"github.com/nusiss-capstone-project/task-mservice/common/taskpb"
	"github.com/nusiss-capstone-project/task-mservice/server/log"
)

type TaskService struct {
	taskpb.UnimplementedTaskServiceServer
}

func (s *TaskService) SayHello(ctx context.Context, in *taskpb.HelloRequest) (*taskpb.HelloResponse, error) {
	log.Logger.Infof("Received: %v", in.GetName())
	return &taskpb.HelloResponse{Message: "Hello " + in.GetName()}, nil
}
