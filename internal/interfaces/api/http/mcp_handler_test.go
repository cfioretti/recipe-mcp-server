package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/cfioretti/recipe-mcp-server/internal/application"
	"github.com/cfioretti/recipe-mcp-server/internal/domain"
)

type fakeProvider struct {
	generateResp *domain.RecipeDraft
	generateErr  error
	customResp   *domain.RecipeDraft
	customErr    error
}

func (f fakeProvider) GenerateRecipe(_ context.Context, _ application.GenerateRecipeCommand) (*domain.RecipeDraft, error) {
	return f.generateResp, f.generateErr
}

func (f fakeProvider) CustomizeRecipe(_ context.Context, _ application.CustomizeRecipeCommand) (*domain.RecipeDraft, error) {
	return f.customResp, f.customErr
}

func newTestRouter(provider fakeProvider) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	service := application.NewRecipeToolsService(provider)
	handler := NewMCPHandler(service)
	handler.RegisterRoutes(router)
	return router
}

func TestHandleListTools(t *testing.T) {
	router := newTestRouter(fakeProvider{})
	req := httptest.NewRequest(http.MethodGet, "/mcp/tools", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestHandleGenerateRecipeSuccess(t *testing.T) {
	router := newTestRouter(fakeProvider{
		generateResp: &domain.RecipeDraft{
			Name:    "Random Pizza",
			Dough:   map[string]float64{"flour": 1000, "water": 700},
			Topping: map[string]float64{"mozzarella": 300},
		},
	})

	body := map[string]any{
		"mode": "random",
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/mcp/tools/generate_recipe", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestHandleGenerateRecipeValidationError(t *testing.T) {
	router := newTestRouter(fakeProvider{})

	body := map[string]any{
		"mode": "prompt",
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/mcp/tools/generate_recipe", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}
