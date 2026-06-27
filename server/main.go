package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nusiss-capstone-project/task-mservice/server/config"
	"github.com/nusiss-capstone-project/task-mservice/server/grpc"
	"github.com/nusiss-capstone-project/task-mservice/server/http"
	"github.com/nusiss-capstone-project/task-mservice/server/kafka/listener"
	_ "github.com/nusiss-capstone-project/task-mservice/server/kafka/listener/handlers"
	"github.com/nusiss-capstone-project/task-mservice/server/log"
	"github.com/nusiss-capstone-project/task-mservice/server/repository"
	"github.com/nusiss-capstone-project/task-mservice/server/telemetry"
)

var (
	sigCh = make(chan os.Signal, 1)
)

func main() {
	config.Init()
	log.InitLogger()
	repository.Init()

	shutdownTelemetry := telemetry.Init(context.Background())
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTelemetry(ctx); err != nil {
			log.Logger.Errorw("telemetry shutdown failed", "error", err)
		}
	}()

	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()

	go grpc.Init(sigCh)
	go http.Init(sigCh)
	consumer.Init(appCtx)

	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	appCancel()
	log.Logger.Infof("Received signal: %v, shutting down...", sig)
}
