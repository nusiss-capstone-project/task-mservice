package router

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	commonauth "github.com/nusiss-capstone-project/identity-mservice/common/auth"
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

	adminAuth := commonauth.RequireRole([]string{
		commonauth.RoleCampaignOps, commonauth.RoleAdmin,
	})

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

		adminGroup := basicGroup.Group("/admin")
		adminGroup.Use(adminAuth)
		{
			adminGroup.POST("/task-groups", api.SaveTaskGroup)
			adminGroup.GET("/task-groups", api.ListTaskGroups)
			adminGroup.PATCH("/task-groups/:task_group_id", api.PublishTaskGroup)

			adminGroup.POST("/task-group/:task_group_id/tasks", api.CreateTask)
			adminGroup.PUT("/task-group/:task_group_id/tasks/:task_id", api.SaveTask)
			adminGroup.GET("/task-group/:task_group_id/tasks", api.ListTasksByGroup)
			adminGroup.GET("/task-group/:task_group_id/tasks/:task_id", api.GetTaskDetail)
			adminGroup.PATCH("/tasks/:task_id", api.PublishTask)

			adminGroup.GET("/data-metrics", api.ListDataMetrics)
			adminGroup.GET("/data-metric-operators", api.ListDataMetricOperators)
		}
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
			"Origin", "Content-Type", "Accept", "Authorization",
			commonauth.HeaderInternalUserID, commonauth.HeaderUserRole, log.RequestIDHeader,
			"traceparent", "tracestate",
		},
		ExposeHeaders: []string{
			"Content-Length", commonauth.HeaderInternalUserID, commonauth.HeaderUserRole, log.RequestIDHeader,
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
