package handler

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"

	"picflow/backend/internal/errno"
	"picflow/backend/internal/svc"

	"github.com/gin-gonic/gin"
)

// Web 在统一镜像中提供前端静态文件，并为前端路由返回 index.html。
// 本地开发时 WEB_DIR 为空，前端继续由 Vite 单独提供。
func Web(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestPath := c.Request.URL.Path
		if isBackendPath(requestPath) || svcCtx.Config.WebDir == "" {
			abort(c, errno.NotFound("请求的接口不存在"))
			return
		}

		if c.Request.Method != "GET" && c.Request.Method != "HEAD" {
			abort(c, errno.MethodNotAllowed())
			return
		}

		relativePath := strings.TrimPrefix(path.Clean("/"+requestPath), "/")
		if relativePath != "" {
			candidate := filepath.Join(svcCtx.Config.WebDir, filepath.FromSlash(relativePath))
			if isInsideDirectory(svcCtx.Config.WebDir, candidate) {
				if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
					c.File(candidate)
					return
				}
			}
		}

		// 带扩展名的静态资源不存在时应返回 404，不能返回 HTML。
		if filepath.Ext(relativePath) != "" {
			abort(c, errno.NotFound("静态资源不存在"))
			return
		}

		indexPath := filepath.Join(svcCtx.Config.WebDir, "index.html")
		if _, err := os.Stat(indexPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				abort(c, errno.NotFound("前端页面不存在"))
			} else {
				abort(c, errno.Internal(err))
			}
			return
		}
		c.File(indexPath)
	}
}

func isBackendPath(requestPath string) bool {
	return requestPath == "/api" || strings.HasPrefix(requestPath, "/api/") ||
		requestPath == "/health" || strings.HasPrefix(requestPath, "/health/")
}

func isInsideDirectory(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
