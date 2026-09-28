package handler

import (
	"picflow/backend/internal/api/req"
	"picflow/backend/internal/api/resp"
	"picflow/backend/internal/service"
	"picflow/backend/internal/svc"
	"picflow/backend/internal/types"

	"github.com/gin-gonic/gin"
)

func ListTemplates(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := service.NewTemplateService(svcCtx).List()
		if err != nil {
			abort(c, err)
			return
		}
		resp.OK(c, result)
	}
}

func CreateTemplate(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input types.CreateTemplateRequest
		if err := req.Bind(c, &input); err != nil {
			abort(c, err)
			return
		}
		result, err := service.NewTemplateService(svcCtx).Create(input)
		if err != nil {
			abort(c, err)
			return
		}
		resp.Created(c, result)
	}
}

func DeleteTemplate(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := service.NewTemplateService(svcCtx).Delete(c.Param("templateID")); err != nil {
			abort(c, err)
			return
		}
		resp.OK(c, nil)
	}
}
