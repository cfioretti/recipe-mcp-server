package http

import (
	stdhttp "net/http"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	service string
	version string
}

func NewHealthHandler(service string, version string) *HealthHandler {
	return &HealthHandler{
		service: service,
		version: version,
	}
}

func (h *HealthHandler) RegisterRoutes(router *gin.Engine) {
	router.GET("/health", h.handleHealth)
}

func (h *HealthHandler) handleHealth(c *gin.Context) {
	c.JSON(stdhttp.StatusOK, gin.H{
		"status":  "healthy",
		"service": h.service,
		"version": h.version,
	})
}
