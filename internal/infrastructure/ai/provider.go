package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/cfioretti/recipe-mcp-server/internal/application"
	"github.com/cfioretti/recipe-mcp-server/internal/domain"
)

type callFn func(ctx context.Context, prompt string) (*domain.RecipeDraft, error)

func callWithRetry(ctx context.Context, fn callFn, initialPrompt string, outputContract domain.OutputContract, maxAttempts int, providerName string) (*domain.RecipeDraft, error) {
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	currentPrompt := initialPrompt
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		draft, err := fn(ctx, currentPrompt)
		if err == nil {
			if validateErr := draft.Validate(); validateErr == nil {
				if contractErr := draft.ValidateAgainstContract(outputContract); contractErr == nil {
					return draft, nil
				} else {
					lastErr = contractErr
					currentPrompt = buildRepairPrompt(initialPrompt, "", contractErr)
					continue
				}
			} else {
				lastErr = validateErr
				currentPrompt = buildRepairPrompt(initialPrompt, "", validateErr)
				continue
			}
		}

		lastErr = err
		if isContextOrTimeoutError(err) {
			break
		}
		currentPrompt = buildRepairPrompt(initialPrompt, "", err)
	}

	return nil, fmt.Errorf("%s generation failed after %d attempts: %w", providerName, maxAttempts, lastErr)
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

func parseRecipeDraftFromText(raw string) (*domain.RecipeDraft, error) {
	cleaned := stripMarkdownCodeFences(strings.TrimSpace(raw))

	var direct domain.RecipeDraft
	if err := json.Unmarshal([]byte(cleaned), &direct); err == nil && direct.Name != "" {
		normalizeRecipeDraftKeys(&direct)
		return &direct, nil
	}

	var wrapped struct {
		RecipeDraft domain.RecipeDraft `json:"recipeDraft"`
	}
	if err := json.Unmarshal([]byte(cleaned), &wrapped); err == nil && wrapped.RecipeDraft.Name != "" {
		normalizeRecipeDraftKeys(&wrapped.RecipeDraft)
		return &wrapped.RecipeDraft, nil
	}

	candidates := extractJSONCandidates(cleaned)
	if len(candidates) == 0 {
		return nil, errors.New("no JSON object found in provider response")
	}
	for _, candidate := range candidates {
		if err := json.Unmarshal([]byte(candidate), &direct); err == nil && direct.Name != "" {
			normalizeRecipeDraftKeys(&direct)
			return &direct, nil
		}
		if err := json.Unmarshal([]byte(candidate), &wrapped); err == nil && wrapped.RecipeDraft.Name != "" {
			normalizeRecipeDraftKeys(&wrapped.RecipeDraft)
			return &wrapped.RecipeDraft, nil
		}
	}

	return nil, errors.New("unable to parse recipe draft from provider response")
}

var ingredientAliases = map[string]string{
	"all-purpose flour":      "flour",
	"bread flour":            "flour",
	"00 flour":               "flour",
	"tipo 00 flour":          "flour",
	"plain flour":            "flour",
	"strong flour":           "flour",
	"wheat flour":            "flour",
	"tap water":              "water",
	"warm water":             "water",
	"lukewarm water":         "water",
	"cold water":             "water",
	"sea salt":               "salt",
	"kosher salt":            "salt",
	"fine salt":              "salt",
	"table salt":             "salt",
	"active dry yeast":       "yeast",
	"instant yeast":          "yeast",
	"dry yeast":              "yeast",
	"fresh yeast":            "yeast",
	"olive oil":              "olive oil",
	"extra virgin olive oil": "olive oil",
}

func normalizeRecipeDraftKeys(draft *domain.RecipeDraft) {
	draft.Dough = normalizeIngredientMap(draft.Dough)
	draft.Topping = normalizeIngredientMap(draft.Topping)
}

func normalizeIngredientMap(ingredients map[string]float64) map[string]float64 {
	normalized := make(map[string]float64, len(ingredients))
	for key, value := range ingredients {
		canonicalKey := strings.ToLower(strings.TrimSpace(key))
		if alias, ok := ingredientAliases[canonicalKey]; ok {
			canonicalKey = alias
		}
		if existing, ok := normalized[canonicalKey]; ok {
			normalized[canonicalKey] = existing + value
		} else {
			normalized[canonicalKey] = value
		}
	}
	return normalized
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

func buildGenerationPrompt(request application.GenerateRecipeCommand, outputContract domain.OutputContract) string {
	contractJSON := "{}"
	if b, err := json.Marshal(outputContract); err == nil {
		contractJSON = string(b)
	}

	modeNote := "Generate a random pizza recipe."
	if request.Mode == domain.ModePrompt {
		modeNote = "Generate a pizza recipe from this user prompt: " + strings.TrimSpace(request.Prompt)
	}

	return fmt.Sprintf(
		`%s Return ONLY valid JSON (no markdown, no prose, no code fences) matching this exact schema: {"recipeDraft":{"name":"string","description":"string","author":"string","dough":{"ingredient":number},"topping":{"ingredient":number},"steps":["string"]}}. CRITICAL RULES: 1) All ingredient key names MUST be lowercase single words (use "flour" not "All-Purpose Flour", use "water" not "Warm Water", use "salt" not "Sea Salt", use "yeast" not "Active Dry Yeast"). 2) All values must be positive numbers in grams. 3) The dough object MUST contain at minimum the keys "flour" and "water". Required output contract: %s.`,
		modeNote,
		contractJSON,
	)
}

func buildCustomizationPrompt(request application.CustomizeRecipeCommand) string {
	baseRecipeJSON := "{}"
	if b, err := json.Marshal(request.BaseRecipe); err == nil {
		baseRecipeJSON = string(b)
	}

	return fmt.Sprintf(
		"Customize this base recipe: %s. Mode=%s. User prompt=%q. Return ONLY valid JSON with schema {\"recipeDraft\":{\"name\":\"string\",\"description\":\"string\",\"author\":\"string\",\"dough\":{\"ingredient\":number},\"topping\":{\"ingredient\":number},\"steps\":[\"string\"]}}.",
		baseRecipeJSON,
		request.Mode,
		request.Prompt,
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
		`%s%s Please answer again and return ONLY valid JSON matching exactly this schema: {"recipeDraft":{"name":"string","description":"string","author":"string","dough":{"ingredient":number},"topping":{"ingredient":number},"steps":["string"]}}. CRITICAL: All ingredient keys must be lowercase single words (e.g. "flour", "water", "salt", "yeast", "mozzarella"). The dough MUST contain "flour" and "water" keys. Do not include markdown, prose, or code fences. Original task: %s`,
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
