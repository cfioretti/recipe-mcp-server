package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOllamaProvider_Call_ParsesValidResponse(t *testing.T) {
	recipeJSON := `{"recipeDraft":{"name":"Ollama Pizza","description":"AI generated","author":"Ollama","dough":{"flour":1000,"water":650,"salt":20},"topping":{"mozzarella":250,"tomato":150},"steps":["Mix","Rise","Bake"]}}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Fatalf("expected path /api/generate, got %s", r.URL.Path)
		}

		var req ollamaGenerateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if req.Model != "test-model" {
			t.Fatalf("expected model 'test-model', got %q", req.Model)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ollamaGenerateResponse{Response: recipeJSON})
	}))
	defer server.Close()

	provider := &OllamaProvider{
		client:  server.Client(),
		baseURL: server.URL,
		model:   "test-model",
	}

	draft, err := provider.call(context.Background(), "Generate a pizza recipe")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if draft.Name != "Ollama Pizza" {
		t.Fatalf("expected name 'Ollama Pizza', got %q", draft.Name)
	}
	if draft.Dough["flour"] != 1000 {
		t.Fatalf("expected flour=1000, got %v", draft.Dough["flour"])
	}
}

func TestOllamaProvider_Call_HandlesErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"model 'unknown' not found"}`))
	}))
	defer server.Close()

	provider := &OllamaProvider{
		client:  server.Client(),
		baseURL: server.URL,
		model:   "unknown",
	}

	draft, err := provider.call(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for missing model")
	}
	if draft != nil {
		t.Fatal("expected nil draft on error")
	}
}
