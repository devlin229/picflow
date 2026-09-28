package handler

import (
	"picflow/backend/internal/api/resp"
	"picflow/backend/internal/service"
	"picflow/backend/internal/svc"

	"github.com/gin-gonic/gin"
)

func Live(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		healthService := service.NewHealthService(c.Request.Context(), svcCtx)
		result, err := healthService.Live()
		if err != nil {
			_ = c.Error(err)
			c.Abort()
			return
		}
		resp.OK(c, result)
	}
}

func Ready(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		healthService := service.NewHealthService(c.Request.Context(), svcCtx)
		result, err := healthService.Ready()
		if err != nil {
			_ = c.Error(err)
			c.Abort()
			return
		}
		resp.OK(c, result)
	}
}
