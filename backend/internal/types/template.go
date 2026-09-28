package types

import "encoding/json"

type CreateTemplateRequest struct {
	Name        string          `json:"name" binding:"required,max=80"`
	Type        string          `json:"type" binding:"required,oneof=main_image size_chart"`
	Description string          `json:"description" binding:"max=255"`
	Config      json.RawMessage `json:"config" binding:"required"`
}

type TemplateResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Config      any    `json:"config"`
	BuiltIn     bool   `json:"built_in"`
}

type TemplateListResponse struct {
	Items []TemplateResponse `json:"items"`
}
