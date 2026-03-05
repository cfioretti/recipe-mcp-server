package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGeminiProvider_Call_ParsesValidResponse(t *testing.T) {
	recipeJSON := `{"recipeDraft":{"name":"Gemini Pizza","description":"AI generated","author":"Gemini","dough":{"flour":1000,"water":650,"salt":20},"topping":{"mozzarella":250,"tomato":150},"steps":["Mix","Rise","Bake"]}}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}

		var req geminiRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if len(req.Contents) == 0 || len(req.Contents[0].Parts) == 0 {
			t.Fatal("expected non-empty contents")
		}

		w.Header().Set("Content-Type", "application/json")
		resp := geminiResponse{
			Candidates: []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			}{
				{Content: struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				}{Parts: []struct {
					Text string `json:"text"`
				}{{Text: recipeJSON}}}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := &GeminiProvider{
		client:  server.Client(),
		baseURL: server.URL,
		model:   "gemini-2.0-flash",
		apiKey:  "test-key",
	}

	draft, err := provider.call(context.Background(), "Generate a pizza recipe")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if draft.Name != "Gemini Pizza" {
		t.Fatalf("expected name 'Gemini Pizza', got %q", draft.Name)
	}
	if draft.Dough["flour"] != 1000 {
		t.Fatalf("expected flour=1000, got %v", draft.Dough["flour"])
	}
}

func TestGeminiProvider_Call_HandlesErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"API key not valid","code":400}}`))
	}))
	defer server.Close()

	provider := &GeminiProvider{
		client:  server.Client(),
		baseURL: server.URL,
		model:   "gemini-2.0-flash",
		apiKey:  "bad-key",
	}

	draft, err := provider.call(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for bad API key")
	}
	if draft != nil {
		t.Fatal("expected nil draft on error")
	}
}

func TestGeminiProvider_Call_HandlesEmptyCandidates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[]}`))
	}))
	defer server.Close()

	provider := &GeminiProvider{
		client:  server.Client(),
		baseURL: server.URL,
		model:   "gemini-2.0-flash",
		apiKey:  "test-key",
	}

	draft, err := provider.call(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for empty candidates")
	}
	if draft != nil {
		t.Fatal("expected nil draft on empty candidates")
	}
}
