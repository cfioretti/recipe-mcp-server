package dto

import "github.com/cfioretti/recipe-mcp-server/internal/domain"

type ToolListResponse struct {
	Tools []domain.ToolDefinition `json:"tools"`
}

type RecipeDraftResponse struct {
	RecipeDraft domain.RecipeDraft `json:"recipeDraft"`
}
