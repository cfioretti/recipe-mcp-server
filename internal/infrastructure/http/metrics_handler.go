package http

import (
	stdhttp "net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricsHandler struct {
	handler stdhttp.Handler
}

func NewMetricsHandler() *MetricsHandler {
	return &MetricsHandler{
		handler: promhttp.Handler(),
	}
}

func (h *MetricsHandler) RegisterRoutes(router *gin.Engine) {
	router.GET("/metrics", h.handleMetrics)
}

func (h *MetricsHandler) handleMetrics(c *gin.Context) {
	h.handler.ServeHTTP(c.Writer, c.Request)
}
