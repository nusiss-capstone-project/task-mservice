package client

import (
	"fmt"
	"sync"

	"github.com/nusiss-capstone-project/task-mservice/common/taskpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	conn           *grpc.ClientConn
	client         taskpb.TaskServiceClient
	clientSyncOnce sync.Once
)

func GetTaskServiceClient(config *GRpcClientConfig) (taskpb.TaskServiceClient, error) {
	clientSyncOnce.Do(func() {
		opts := []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024 * 1024)),
			grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(1024 * 1024)),
		}
		conn, err := grpc.NewClient(fmt.Sprintf("%s:%d", config.Host, config.Port), opts...)
		if err != nil {
			panic(err)
		}
		client = taskpb.NewTaskServiceClient(conn)
	})
	return client, nil
}

func Destroy() {
	if conn != nil {
		err := conn.Close()
		if err != nil {
			fmt.Printf("Failed to close gRPC connection: %v\n", err)
		}
	}
}
