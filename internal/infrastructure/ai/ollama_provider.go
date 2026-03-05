package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cfioretti/recipe-mcp-server/internal/application"
	"github.com/cfioretti/recipe-mcp-server/internal/domain"
)

const (
	defaultOllamaURL   = "http://localhost:11434"
	defaultOllamaModel = "llama3.2:3b"
)

type OllamaProvider struct {
	client      *http.Client
	baseURL     string
	model       string
	maxAttempts int
}

func NewOllamaProviderFromEnv() *OllamaProvider {
	timeoutMs := readIntFromEnv("AI_HTTP_TIMEOUT_MS", 60000)
	return &OllamaProvider{
		client: &http.Client{
			Timeout: time.Duration(timeoutMs) * time.Millisecond,
		},
		baseURL:     strings.TrimSuffix(readStringFromEnv("OLLAMA_BASE_URL", defaultOllamaURL), "/"),
		model:       readStringFromEnv("OLLAMA_MODEL", defaultOllamaModel),
		maxAttempts: readIntFromEnv("AI_GENERATION_MAX_ATTEMPTS", 2),
	}
}

func (p *OllamaProvider) GenerateRecipe(ctx context.Context, request application.GenerateRecipeCommand) (*domain.RecipeDraft, error) {
	effectiveContract := request.OutputContract.Effective()
	prompt := buildGenerationPrompt(request, effectiveContract)
	return callWithRetry(ctx, p.call, prompt, effectiveContract, p.maxAttempts, "ollama")
}

func (p *OllamaProvider) CustomizeRecipe(ctx context.Context, request application.CustomizeRecipeCommand) (*domain.RecipeDraft, error) {
	prompt := buildCustomizationPrompt(request)
	return callWithRetry(ctx, p.call, prompt, domain.DefaultOutputContract(), p.maxAttempts, "ollama")
}

type ollamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaGenerateResponse struct {
	Response string `json:"response"`
}

func (p *OllamaProvider) call(ctx context.Context, prompt string) (*domain.RecipeDraft, error) {
	payload, err := json.Marshal(ollamaGenerateRequest{
		Model:  p.model,
		Prompt: prompt,
		Stream: false,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/generate", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama request failed: %s - %s", resp.Status, string(body))
	}

	var ollamaResp ollamaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, err
	}

	return parseRecipeDraftFromText(ollamaResp.Response)
}
