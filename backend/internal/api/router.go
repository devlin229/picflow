package api

import (
	"net/http"

	"picflow/backend/internal/api/middleware"
	"picflow/backend/internal/svc"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// NewRouter 创建 PicFlow HTTP 路由。
func NewRouter(svcCtx *svc.ServiceContext) *gin.Engine {
	gin.SetMode(svcCtx.Config.Env)
	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.Use(otelgin.Middleware(svcCtx.Config.Name))
	router.Use(middleware.TraceID())
	router.Use(middleware.CORS(svcCtx.Config.CORSAllowedOrigins))
	router.Use(middleware.AccessLog(svcCtx.Logger))
	router.Use(middleware.Response(svcCtx.Logger))
	router.Use(middleware.Recovery(svcCtx.Logger))
	router.Use(middleware.BodyLimit(middleware.DefaultBodyLimit, middleware.RouteBodyLimit{
		Method: http.MethodPost, Path: "/api/tasks", Limit: svcCtx.Config.MaxUploadSize*100 + 2<<20,
	}))
	registerRoutes(router, svcCtx)
	return router
}
