package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	httpHandlers "github.com/cfioretti/recipe-mcp-server/internal/infrastructure/http"
)

const serviceName = "recipe-mcp-server"

func main() {
	port := readPort()
	version := readVersion()
	router := setupRouter(version)
	startServerWithGracefulShutdown(router, port)
}

func setupRouter(version string) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	metricsHandler := httpHandlers.NewMetricsHandler()
	metricsHandler.RegisterRoutes(router)

	healthHandler := httpHandlers.NewHealthHandler(serviceName, version)
	healthHandler.RegisterRoutes(router)

	mcpHandler := httpHandlers.NewMCPHandler()
	mcpHandler.RegisterRoutes(router)

	return router
}

func readVersion() string {
	version := os.Getenv("APP_VERSION")
	if version == "" {
		return "dev"
	}

	return version
}

func readPort() int {
	raw := os.Getenv("MCP_SERVER_PORT")
	if raw == "" {
		return 8080
	}

	port, err := strconv.Atoi(raw)
	if err != nil || port <= 0 {
		log.Printf("invalid MCP_SERVER_PORT=%q, falling back to 8080", raw)
		return 8080
	}

	return port
}

func startServerWithGracefulShutdown(router *gin.Engine, port int) {
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("%s listening on :%d", serviceName, port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server failed: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("shutdown signal received")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}
