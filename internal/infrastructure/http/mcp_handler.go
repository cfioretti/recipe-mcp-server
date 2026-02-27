package http

import (
	stdhttp "net/http"

	"github.com/gin-gonic/gin"
)

type MCPHandler struct{}

func NewMCPHandler() *MCPHandler {
	return &MCPHandler{}
}

func (h *MCPHandler) RegisterRoutes(router *gin.Engine) {
	router.GET("/mcp", h.handlePlaceholder)
}

func (h *MCPHandler) handlePlaceholder(c *gin.Context) {
	c.JSON(stdhttp.StatusOK, gin.H{
		"message": "MCP server skeleton ready. Tool contract will be added in next step.",
	})
}
