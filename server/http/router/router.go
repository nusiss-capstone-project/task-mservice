package router

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/nusiss-capstone-project/task-mservice/server/config"
	_ "github.com/nusiss-capstone-project/task-mservice/server/docs"
	"github.com/nusiss-capstone-project/task-mservice/server/http/api"
	"github.com/nusiss-capstone-project/task-mservice/server/http/data"
	"github.com/nusiss-capstone-project/task-mservice/server/log"
	swaggerFiles "github.com/swaggo/files"
	gs "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

const (
	serviceURIPrefix = "/task-ms/v1"
)

func NewRouter() *gin.Engine {
	r := gin.New()
	r.Use(log.RecoveryMiddleware())
	r.Use(otelgin.Middleware(data.ServiceName))
	r.Use(log.HTTPObservabilityMiddleware())
	r.Use(corsMiddleware())

	basicGroup := r.Group(serviceURIPrefix)
	{
		basicGroup.GET("/swagger/*any", gs.WrapHandler(
			swaggerFiles.Handler,
			gs.URL("/task-ms/v1/swagger/doc.json"),
		))
		basicGroup.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "pong",
			})
		})
		basicGroup.POST("/items", api.CreateItem)
		basicGroup.GET("/items/:item_id", api.GetItems)

		basicGroup.POST("/task-groups", api.SaveTaskGroup)
		basicGroup.GET("/task-groups", api.ListTaskGroups)
		basicGroup.PATCH("/task-groups/:task_group_id", api.PublishTaskGroup)

		basicGroup.POST("/task-group/:task_group_id/tasks", api.CreateTask)
		basicGroup.PUT("/task-group/:task_group_id/tasks/:task_id", api.SaveTask)
		basicGroup.GET("/task-group/:task_group_id/tasks", api.ListTasksByGroup)
		basicGroup.GET("/task-group/:task_group_id/tasks/:task_id", api.GetTaskDetail)
		basicGroup.PATCH("/tasks/:task_id", api.PublishTask)

		basicGroup.GET("/data-metrics", api.ListDataMetrics)
		basicGroup.GET("/data-metric-operators", api.ListDataMetricOperators)
	}
	return r
}

func corsMiddleware() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins: allowedOrigins(),
		AllowMethods: []string{
			"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS",
		},
		AllowHeaders: []string{
			"Origin", "Content-Type", "Accept", "Authorization", log.RequestIDHeader,
			"traceparent", "tracestate",
		},
		ExposeHeaders: []string{
			"Content-Length", log.RequestIDHeader,
		},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}

func allowedOrigins() []string {
	if config.Config == nil || config.Config.SystemConfig == nil {
		return []string{}
	}
	return config.Config.SystemConfig.AllowedOrigins
}
