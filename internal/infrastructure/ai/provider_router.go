package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cfioretti/recipe-mcp-server/internal/application"
	"github.com/cfioretti/recipe-mcp-server/internal/domain"
)

const (
	defaultProvider    = "ollama"
	defaultOllamaURL   = "http://localhost:11434"
	defaultOllamaModel = "llama3.2:3b"
)

type ProviderRouter struct {
	client             *http.Client
	provider           string
	ollamaBaseURL      string
	ollamaModel        string
	maxAttempts        int
	externalAPIBaseURL string
	externalAPIKey     string
}

func NewProviderRouterFromEnv() *ProviderRouter {
	timeoutMs := readIntFromEnv("AI_HTTP_TIMEOUT_MS", 60000)
	return &ProviderRouter{
		client: &http.Client{
			Timeout: time.Duration(timeoutMs) * time.Millisecond,
		},
		provider:           strings.ToLower(readStringFromEnv("AI_PROVIDER", defaultProvider)),
		ollamaBaseURL:      strings.TrimSuffix(readStringFromEnv("OLLAMA_BASE_URL", defaultOllamaURL), "/"),
		ollamaModel:        readStringFromEnv("OLLAMA_MODEL", defaultOllamaModel),
		maxAttempts:        readIntFromEnv("AI_GENERATION_MAX_ATTEMPTS", 2),
		externalAPIBaseURL: strings.TrimSuffix(readStringFromEnv("EXTERNAL_API_BASE_URL", ""), "/"),
		externalAPIKey:     readStringFromEnv("EXTERNAL_API_KEY", ""),
	}
}

func (r *ProviderRouter) GenerateRecipe(ctx context.Context, request application.GenerateRecipeCommand) (*domain.RecipeDraft, error) {
	switch r.provider {
	case "ollama":
		return r.generateWithOllama(ctx, request)
	case "external":
		return r.generateWithExternal(ctx, request)
	case "mock":
		return r.generateWithMock(request), nil
	default:
		return nil, fmt.Errorf("unsupported AI_PROVIDER: %s", r.provider)
	}
}

func (r *ProviderRouter) CustomizeRecipe(ctx context.Context, request application.CustomizeRecipeCommand) (*domain.RecipeDraft, error) {
	switch r.provider {
	case "ollama":
		return r.customizeWithOllama(ctx, request)
	case "external":
		return r.customizeWithExternal(ctx, request)
	case "mock":
		return r.customizeWithMock(request), nil
	default:
		return nil, fmt.Errorf("unsupported AI_PROVIDER: %s", r.provider)
	}
}

type ollamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaGenerateResponse struct {
	Response string `json:"response"`
}

func (r *ProviderRouter) generateWithOllama(ctx context.Context, request application.GenerateRecipeCommand) (*domain.RecipeDraft, error) {
	prompt := buildGenerationPrompt(request)
	return r.callOllamaWithRetry(ctx, prompt)
}

func (r *ProviderRouter) customizeWithOllama(ctx context.Context, request application.CustomizeRecipeCommand) (*domain.RecipeDraft, error) {
	prompt := buildCustomizationPrompt(request)
	return r.callOllamaWithRetry(ctx, prompt)
}

func (r *ProviderRouter) callOllama(ctx context.Context, prompt string) (*domain.RecipeDraft, error) {
	payload, err := json.Marshal(ollamaGenerateRequest{
		Model:  r.ollamaModel,
		Prompt: prompt,
		Stream: false,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.ollamaBaseURL+"/api/generate", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
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

func (r *ProviderRouter) callOllamaWithRetry(ctx context.Context, initialPrompt string) (*domain.RecipeDraft, error) {
	maxAttempts := r.maxAttempts
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	currentPrompt := initialPrompt
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		draft, err := r.callOllama(ctx, currentPrompt)
		if err == nil {
			if validateErr := draft.Validate(); validateErr == nil {
				return draft, nil
			} else {
				lastErr = validateErr
				currentPrompt = buildRepairPrompt(initialPrompt, "", validateErr)
				continue
			}
		}

		lastErr = err
		// Timeout/cancellation are usually infrastructure issues; retrying with a repair
		// prompt does not help and only increases end-to-end latency.
		if isContextOrTimeoutError(err) {
			break
		}
		currentPrompt = buildRepairPrompt(initialPrompt, "", err)
	}

	return nil, fmt.Errorf("ollama generation failed after %d attempts: %w", maxAttempts, lastErr)
}

func isContextOrTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "timeout")
}

type externalGeneratePayload struct {
	Mode        domain.GenerationMode     `json:"mode"`
	Prompt      string                    `json:"prompt,omitempty"`
	Constraints *domain.RecipeConstraints `json:"constraints,omitempty"`
}

type externalCustomizePayload struct {
	Mode        domain.GenerationMode     `json:"mode"`
	Prompt      string                    `json:"prompt,omitempty"`
	Constraints *domain.RecipeConstraints `json:"constraints,omitempty"`
	BaseRecipe  domain.RecipeDraft        `json:"baseRecipe"`
}

func (r *ProviderRouter) generateWithExternal(ctx context.Context, request application.GenerateRecipeCommand) (*domain.RecipeDraft, error) {
	if r.externalAPIBaseURL == "" {
		return nil, errors.New("EXTERNAL_API_BASE_URL is required for external provider")
	}

	payload := externalGeneratePayload{
		Mode:        request.Mode,
		Prompt:      request.Prompt,
		Constraints: request.Constraints,
	}
	return r.callExternal(ctx, "/generate_recipe", payload)
}

func (r *ProviderRouter) customizeWithExternal(ctx context.Context, request application.CustomizeRecipeCommand) (*domain.RecipeDraft, error) {
	if r.externalAPIBaseURL == "" {
		return nil, errors.New("EXTERNAL_API_BASE_URL is required for external provider")
	}

	payload := externalCustomizePayload{
		Mode:        request.Mode,
		Prompt:      request.Prompt,
		Constraints: request.Constraints,
		BaseRecipe:  request.BaseRecipe,
	}
	return r.callExternal(ctx, "/customize_recipe", payload)
}

func (r *ProviderRouter) callExternal(ctx context.Context, path string, payload any) (*domain.RecipeDraft, error) {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.externalAPIBaseURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if r.externalAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+r.externalAPIKey)
	}

	resp, err := r.client.Do(req)
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
		return nil, fmt.Errorf("external API request failed: %s - %s", resp.Status, string(respBody))
	}

	return parseRecipeDraftFromText(string(respBody))
}

func (r *ProviderRouter) generateWithMock(request application.GenerateRecipeCommand) *domain.RecipeDraft {
	name := "Random Contemporary Pizza"
	description := "Generated with random mode."
	if request.Mode == domain.ModePrompt {
		name = "Prompted Pizza"
		description = fmt.Sprintf("Generated from prompt: %s", strings.TrimSpace(request.Prompt))
	}

	draft := &domain.RecipeDraft{
		Name:        name,
		Description: description,
		Author:      "recipe-mcp-server",
		Dough: map[string]float64{
			"flour": 1000,
			"water": 700,
			"salt":  25,
			"yeast": 2.5,
		},
		Topping: map[string]float64{
			"tomatoSauce": 250,
			"mozzarella":  300,
		},
		Steps: []string{
			"Mix ingredients.",
			"Bulk ferment.",
			"Shape and top.",
			"Bake at high temperature.",
		},
	}

	applyConstraintHints(draft, request.Constraints)
	return draft
}

func (r *ProviderRouter) customizeWithMock(request application.CustomizeRecipeCommand) *domain.RecipeDraft {
	customized := request.BaseRecipe
	if strings.TrimSpace(request.Prompt) != "" {
		customized.Description = strings.TrimSpace(request.Prompt)
	}

	if customized.Author == "" {
		customized.Author = "recipe-mcp-server"
	}

	applyConstraintHints(&customized, request.Constraints)
	return &customized
}

func applyConstraintHints(draft *domain.RecipeDraft, constraints *domain.RecipeConstraints) {
	if constraints == nil {
		return
	}

	if constraints.Vegetarian != nil && *constraints.Vegetarian {
		// Remove common non-vegetarian topping defaults if present.
		delete(draft.Topping, "anchovies")
		delete(draft.Topping, "ham")
		delete(draft.Topping, "salami")
	}

	for _, ingredient := range constraints.ExcludeIngredients {
		delete(draft.Dough, ingredient)
		delete(draft.Topping, ingredient)
	}
}

func parseRecipeDraftFromText(raw string) (*domain.RecipeDraft, error) {
	cleaned := stripMarkdownCodeFences(strings.TrimSpace(raw))

	var direct domain.RecipeDraft
	if err := json.Unmarshal([]byte(cleaned), &direct); err == nil && direct.Name != "" {
		return &direct, nil
	}

	var wrapped struct {
		RecipeDraft domain.RecipeDraft `json:"recipeDraft"`
	}
	if err := json.Unmarshal([]byte(cleaned), &wrapped); err == nil && wrapped.RecipeDraft.Name != "" {
		return &wrapped.RecipeDraft, nil
	}

	candidates := extractJSONCandidates(cleaned)
	if len(candidates) == 0 {
		return nil, errors.New("no JSON object found in provider response")
	}
	for _, candidate := range candidates {
		if err := json.Unmarshal([]byte(candidate), &direct); err == nil && direct.Name != "" {
			return &direct, nil
		}
		if err := json.Unmarshal([]byte(candidate), &wrapped); err == nil && wrapped.RecipeDraft.Name != "" {
			return &wrapped.RecipeDraft, nil
		}
	}

	return nil, errors.New("unable to parse recipe draft from provider response")
}

func stripMarkdownCodeFences(input string) string {
	trimmed := strings.TrimSpace(input)
	if strings.HasPrefix(trimmed, "```") && strings.HasSuffix(trimmed, "```") {
		trimmed = strings.TrimPrefix(trimmed, "```json")
		trimmed = strings.TrimPrefix(trimmed, "```")
		trimmed = strings.TrimSuffix(trimmed, "```")
	}
	return strings.TrimSpace(trimmed)
}

func extractJSONCandidates(input string) []string {
	var candidates []string
	start := -1
	depth := 0

	for i, r := range input {
		switch r {
		case '{':
			if depth == 0 {
				start = i
			}
			depth++
		case '}':
			if depth == 0 {
				continue
			}
			depth--
			if depth == 0 && start >= 0 {
				candidates = append(candidates, input[start:i+1])
				start = -1
			}
		}
	}
	return candidates
}

func buildGenerationPrompt(request application.GenerateRecipeCommand) string {
	constraintsJSON := "{}"
	if request.Constraints != nil {
		if b, err := json.Marshal(request.Constraints); err == nil {
			constraintsJSON = string(b)
		}
	}

	modeNote := "Generate a random pizza recipe."
	if request.Mode == domain.ModePrompt {
		modeNote = "Generate a pizza recipe from this user prompt: " + strings.TrimSpace(request.Prompt)
	}

	return fmt.Sprintf(
		"%s Return ONLY valid JSON matching this schema: {\"recipeDraft\":{\"name\":\"string\",\"description\":\"string\",\"author\":\"string\",\"dough\":{\"ingredient\":number},\"topping\":{\"ingredient\":number},\"steps\":[\"string\"]}}. Respect optional constraints: %s.",
		modeNote,
		constraintsJSON,
	)
}

func buildCustomizationPrompt(request application.CustomizeRecipeCommand) string {
	baseRecipeJSON := "{}"
	if b, err := json.Marshal(request.BaseRecipe); err == nil {
		baseRecipeJSON = string(b)
	}

	constraintsJSON := "{}"
	if request.Constraints != nil {
		if b, err := json.Marshal(request.Constraints); err == nil {
			constraintsJSON = string(b)
		}
	}

	return fmt.Sprintf(
		"Customize this base recipe: %s. Mode=%s. User prompt=%q. Constraints=%s. Return ONLY valid JSON with schema {\"recipeDraft\":{\"name\":\"string\",\"description\":\"string\",\"author\":\"string\",\"dough\":{\"ingredient\":number},\"topping\":{\"ingredient\":number},\"steps\":[\"string\"]}}.",
		baseRecipeJSON,
		request.Mode,
		request.Prompt,
		constraintsJSON,
	)
}

func buildRepairPrompt(originalPrompt string, previousResponse string, validationErr error) string {
	repairInstructions := "Your previous output could not be parsed/validated as recipe draft JSON."
	if validationErr != nil {
		repairInstructions = fmt.Sprintf("Your previous output is invalid: %s.", validationErr.Error())
	}
	previousOutputClause := ""
	if strings.TrimSpace(previousResponse) != "" {
		previousOutputClause = fmt.Sprintf(" Previous invalid output was: %s.", previousResponse)
	}

	return fmt.Sprintf(
		"%s%s Please answer again and return ONLY valid JSON matching exactly this schema: {\"recipeDraft\":{\"name\":\"string\",\"description\":\"string\",\"author\":\"string\",\"dough\":{\"ingredient\":number},\"topping\":{\"ingredient\":number},\"steps\":[\"string\"]}}. Do not include markdown, prose, or code fences. Original task: %s",
		repairInstructions,
		previousOutputClause,
		originalPrompt,
	)
}

func readStringFromEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func readIntFromEnv(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
