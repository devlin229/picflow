package service

import (
	"encoding/json"
	"errors"
	"strings"

	"picflow/backend/internal/errno"
	"picflow/backend/internal/model"
	"picflow/backend/internal/svc"
	"picflow/backend/internal/types"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TemplateService 处理模板列表、创建和删除。
type TemplateService struct {
	svcCtx *svc.ServiceContext
}

func NewTemplateService(svcCtx *svc.ServiceContext) *TemplateService {
	return &TemplateService{svcCtx: svcCtx}
}

func (s *TemplateService) List() (*types.TemplateListResponse, error) {
	values, err := s.svcCtx.Templates.List()
	if err != nil {
		return nil, errno.Internal(err)
	}
	result := &types.TemplateListResponse{Items: make([]types.TemplateResponse, 0, len(values))}
	for _, value := range values {
		item, convertErr := templateResponse(value)
		if convertErr != nil {
			return nil, errno.Internal(convertErr)
		}
		result.Items = append(result.Items, item)
	}
	return result, nil
}

func (s *TemplateService) Create(request types.CreateTemplateRequest) (*types.TemplateResponse, error) {
	if strings.TrimSpace(request.Name) == "" {
		return nil, errno.InvalidArgument("模板名称不能为空")
	}
	var config map[string]any
	if err := json.Unmarshal(request.Config, &config); err != nil || len(config) == 0 {
		return nil, errno.InvalidArgument("模板配置必须是非空 JSON 对象")
	}
	if request.Type == "main_image" {
		var processRequest types.ProcessConfigRequest
		if err := json.Unmarshal(request.Config, &processRequest); err != nil {
			return nil, errno.InvalidArgument("主图模板配置无效")
		}
		if _, err := validateProcessConfig(processRequest); err != nil {
			return nil, err
		}
	}
	value := &model.Template{
		ID: uuid.NewString(), Name: strings.TrimSpace(request.Name), Type: request.Type,
		Description: strings.TrimSpace(request.Description), ConfigJSON: string(request.Config), BuiltIn: false,
	}
	if err := s.svcCtx.Templates.Create(value); err != nil {
		return nil, errno.Internal(err)
	}
	result, err := templateResponse(*value)
	return &result, err
}

func (s *TemplateService) Delete(id string) error {
	value, err := s.svcCtx.Templates.Get(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errno.NotFound("模板不存在")
	}
	if err != nil {
		return errno.Internal(err)
	}
	if value.BuiltIn {
		return errno.Forbidden("内置模板不能删除")
	}
	if err = s.svcCtx.Templates.Delete(id); err != nil {
		return errno.Internal(err)
	}
	return nil
}

func templateResponse(value model.Template) (types.TemplateResponse, error) {
	var config map[string]any
	if err := json.Unmarshal([]byte(value.ConfigJSON), &config); err != nil {
		return types.TemplateResponse{}, err
	}
	return types.TemplateResponse{
		ID: value.ID, Name: value.Name, Type: value.Type, Description: value.Description,
		Config: config, BuiltIn: value.BuiltIn,
	}, nil
}
