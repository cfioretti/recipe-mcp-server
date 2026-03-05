package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cfioretti/recipe-mcp-server/internal/application"
	"github.com/cfioretti/recipe-mcp-server/internal/domain"
)

const (
	defaultGeminiBaseURL = "https://generativelanguage.googleapis.com"
	defaultGeminiModel   = "gemini-2.5-flash"
)

type GeminiProvider struct {
	client      *http.Client
	baseURL     string
	model       string
	apiKey      string
	maxAttempts int
}

func NewGeminiProviderFromEnv() *GeminiProvider {
	timeoutMs := readIntFromEnv("AI_HTTP_TIMEOUT_MS", 60000)
	return &GeminiProvider{
		client: &http.Client{
			Timeout: time.Duration(timeoutMs) * time.Millisecond,
		},
		baseURL:     strings.TrimSuffix(readStringFromEnv("GEMINI_BASE_URL", defaultGeminiBaseURL), "/"),
		model:       readStringFromEnv("GEMINI_MODEL", defaultGeminiModel),
		apiKey:      readStringFromEnv("GEMINI_API_KEY", ""),
		maxAttempts: readIntFromEnv("AI_GENERATION_MAX_ATTEMPTS", 2),
	}
}

func (p *GeminiProvider) GenerateRecipe(ctx context.Context, request application.GenerateRecipeCommand) (*domain.RecipeDraft, error) {
	if p.apiKey == "" {
		return nil, errors.New("GEMINI_API_KEY is required for gemini provider")
	}
	effectiveContract := request.OutputContract.Effective()
	prompt := buildGenerationPrompt(request, effectiveContract)
	return callWithRetry(ctx, p.call, prompt, effectiveContract, p.maxAttempts, "gemini")
}

func (p *GeminiProvider) CustomizeRecipe(ctx context.Context, request application.CustomizeRecipeCommand) (*domain.RecipeDraft, error) {
	if p.apiKey == "" {
		return nil, errors.New("GEMINI_API_KEY is required for gemini provider")
	}
	prompt := buildCustomizationPrompt(request)
	return callWithRetry(ctx, p.call, prompt, domain.DefaultOutputContract(), p.maxAttempts, "gemini")
}

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error,omitempty"`
}

func (p *GeminiProvider) call(ctx context.Context, prompt string) (*domain.RecipeDraft, error) {
	payload, err := json.Marshal(geminiRequest{
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: prompt}}},
		},
	})
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", p.baseURL, p.model, p.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
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

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("gemini request failed: %s - %s", resp.Status, string(respBody))
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return nil, fmt.Errorf("gemini response parse error: %w", err)
	}
	if geminiResp.Error != nil {
		return nil, fmt.Errorf("gemini API error: %s", geminiResp.Error.Message)
	}
	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, errors.New("gemini returned empty response")
	}

	return parseRecipeDraftFromText(geminiResp.Candidates[0].Content.Parts[0].Text)
}
