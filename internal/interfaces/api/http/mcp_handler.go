package http

import (
	"errors"
	stdhttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/cfioretti/recipe-mcp-server/internal/application"
	"github.com/cfioretti/recipe-mcp-server/internal/domain"
	"github.com/cfioretti/recipe-mcp-server/internal/interfaces/api/http/dto"
)

type MCPHandler struct {
	recipeToolsService *application.RecipeToolsService
}

func NewMCPHandler(recipeToolsService *application.RecipeToolsService) *MCPHandler {
	return &MCPHandler{recipeToolsService: recipeToolsService}
}

func (h *MCPHandler) RegisterRoutes(router *gin.Engine) {
	router.GET("/mcp", h.handleMCPInfo)
	router.GET("/mcp/tools", h.handleListTools)
	router.POST("/mcp/tools/generate_recipe", h.handleGenerateRecipe)
	router.POST("/mcp/tools/customize_recipe", h.handleCustomizeRecipe)
}

func (h *MCPHandler) handleMCPInfo(c *gin.Context) {
	c.JSON(stdhttp.StatusOK, gin.H{
		"name":        "recipe-mcp-server",
		"description": "MCP tool server for recipe generation and customization",
		"version":     "v1",
		"toolsPath":   "/mcp/tools",
	})
}

func (h *MCPHandler) handleListTools(c *gin.Context) {
	c.JSON(stdhttp.StatusOK, gin.H{
		"data": dto.ToolListResponse{
			Tools: h.recipeToolsService.ListTools(),
		},
	})
}

func (h *MCPHandler) handleGenerateRecipe(c *gin.Context) {
	var request dto.GenerateRecipeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		errorResponse(c, stdhttp.StatusBadRequest, err.Error())
		return
	}

	recipeDraft, err := h.recipeToolsService.GenerateRecipe(c.Request.Context(), request.ToApplication())
	if err != nil {
		errorResponse(c, mapDomainErrorToHTTPStatus(err), err.Error())
		return
	}

	c.JSON(stdhttp.StatusOK, gin.H{
		"data": dto.RecipeDraftResponse{
			RecipeDraft: *recipeDraft,
		},
	})
}

func (h *MCPHandler) handleCustomizeRecipe(c *gin.Context) {
	var request dto.CustomizeRecipeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		errorResponse(c, stdhttp.StatusBadRequest, err.Error())
		return
	}

	recipeDraft, err := h.recipeToolsService.CustomizeRecipe(c.Request.Context(), request.ToApplication())
	if err != nil {
		errorResponse(c, mapDomainErrorToHTTPStatus(err), err.Error())
		return
	}

	c.JSON(stdhttp.StatusOK, gin.H{
		"data": dto.RecipeDraftResponse{
			RecipeDraft: *recipeDraft,
		},
	})
}

func mapDomainErrorToHTTPStatus(err error) int {
	switch {
	case errors.Is(err, domain.ErrInvalidMode),
		errors.Is(err, domain.ErrPromptRequired),
		errors.Is(err, domain.ErrInvalidHydration),
		errors.Is(err, domain.ErrInvalidRecipeData):
		return stdhttp.StatusBadRequest
	default:
		return stdhttp.StatusInternalServerError
	}
}

func errorResponse(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"error": message,
	})
}
