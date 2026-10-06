package api

import (
	"picflow/backend/internal/api/handler"
	"picflow/backend/internal/errno"
	"picflow/backend/internal/svc"

	"github.com/gin-gonic/gin"
)

// registerRoutes 集中声明所有 HTTP 路由。
func registerRoutes(router *gin.Engine, svcCtx *svc.ServiceContext) {
	router.GET("/health/live", handler.Live(svcCtx))
	router.GET("/health/ready", handler.Ready(svcCtx))

	api := router.Group("/api")
	api.POST("/tasks", handler.CreateTask(svcCtx))
	api.GET("/tasks", handler.ListTasks(svcCtx))
	api.GET("/tasks/:taskID", handler.GetTask(svcCtx))
	api.POST("/tasks/:taskID/retry", handler.RetryTask(svcCtx))
	api.DELETE("/tasks/:taskID", handler.DeleteTask(svcCtx))
	api.DELETE("/tasks/:taskID/assets/:assetID", handler.DeleteTaskAsset(svcCtx))
	api.POST("/tasks/:taskID/size-charts", handler.CreateSizeChart(svcCtx))
	api.GET("/tasks/:taskID/download.zip", handler.DownloadTaskArchive(svcCtx))
	api.GET("/outputs/:outputID/download", handler.DownloadOutput(svcCtx))

	api.GET("/templates", handler.ListTemplates(svcCtx))
	api.POST("/templates", handler.CreateTemplate(svcCtx))
	api.DELETE("/templates/:templateID", handler.DeleteTemplate(svcCtx))

	router.NoRoute(handler.Web(svcCtx))
	router.NoMethod(func(c *gin.Context) {
		_ = c.Error(errno.MethodNotAllowed())
	})
}
