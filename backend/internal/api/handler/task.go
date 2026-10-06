package handler

import (
	"encoding/json"
	"os"
	"strings"

	"picflow/backend/internal/api/req"
	"picflow/backend/internal/api/resp"
	"picflow/backend/internal/errno"
	"picflow/backend/internal/service"
	"picflow/backend/internal/svc"
	"picflow/backend/internal/types"

	"github.com/gin-gonic/gin"
)

func CreateTask(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
			abort(c, errno.InvalidArgument("上传表单无效"))
			return
		}
		configValues := c.Request.MultipartForm.Value["config"]
		if len(configValues) != 1 {
			abort(c, errno.InvalidArgument("config 必须且只能提交一次"))
			return
		}
		var config types.ProcessConfigRequest
		decoder := json.NewDecoder(strings.NewReader(configValues[0]))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&config); err != nil {
			abort(c, errno.InvalidArgument("处理配置格式无效"))
			return
		}
		taskService := service.NewTaskService(svcCtx)
		result, err := taskService.Create(c.Request.MultipartForm.File["files"], config)
		if err != nil {
			abort(c, err)
			return
		}
		resp.Created(c, result)
	}
}

func GetTask(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := service.NewTaskService(svcCtx).Get(c.Param("taskID"))
		if err != nil {
			abort(c, err)
			return
		}
		resp.OK(c, result)
	}
}

// ListTasks 返回历史结果分页列表。
func ListTasks(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input types.TaskListRequest
		if err := c.ShouldBindQuery(&input); err != nil {
			abort(c, errno.InvalidArgument("历史结果分页参数无效"))
			return
		}
		result, err := service.NewTaskService(svcCtx).List(input)
		if err != nil {
			abort(c, err)
			return
		}
		resp.OK(c, result)
	}
}

func RetryTask(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := service.NewTaskService(svcCtx).Retry(c.Param("taskID"))
		if err != nil {
			abort(c, err)
			return
		}
		resp.OK(c, result)
	}
}

func CreateSizeChart(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input types.SizeChartRequest
		if err := req.Bind(c, &input); err != nil {
			abort(c, err)
			return
		}
		result, err := service.NewTaskService(svcCtx).CreateSizeChart(c.Param("taskID"), input)
		if err != nil {
			abort(c, err)
			return
		}
		resp.Created(c, result)
	}
}

func DownloadOutput(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		output, err := service.NewTaskService(svcCtx).Output(c.Param("outputID"))
		if err != nil {
			abort(c, err)
			return
		}
		c.FileAttachment(output.OutputPath, output.Filename)
	}
}

func DownloadTaskArchive(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		path, err := service.NewTaskService(svcCtx).CreateArchive(c.Param("taskID"))
		if err != nil {
			abort(c, err)
			return
		}
		defer os.Remove(path)
		c.FileAttachment(path, "picflow-results.zip")
	}
}

func DeleteTask(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := service.NewTaskService(svcCtx).Delete(c.Param("taskID")); err != nil {
			abort(c, err)
			return
		}
		resp.OK(c, nil)
	}
}

func DeleteTaskAsset(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := service.NewTaskService(svcCtx).DeleteAsset(c.Param("taskID"), c.Param("assetID")); err != nil {
			abort(c, err)
			return
		}
		resp.OK(c, nil)
	}
}

func abort(c *gin.Context, err error) {
	_ = c.Error(err)
	c.Abort()
}
