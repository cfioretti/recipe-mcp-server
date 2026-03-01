package domain

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidMode       = errors.New("invalid mode")
	ErrPromptRequired    = errors.New("prompt is required when mode is prompt")
	ErrInvalidHydration  = errors.New("hydration range is invalid")
	ErrInvalidRecipeData = errors.New("recipe draft is invalid")
)

type GenerationMode string

const (
	ModeRandom GenerationMode = "random"
	ModePrompt GenerationMode = "prompt"
)

type RecipeConstraints struct {
	HydrationMin       *float64 `json:"hydrationMin,omitempty"`
	HydrationMax       *float64 `json:"hydrationMax,omitempty"`
	Vegetarian         *bool    `json:"vegetarian,omitempty"`
	IncludeIngredients []string `json:"includeIngredients,omitempty"`
	ExcludeIngredients []string `json:"excludeIngredients,omitempty"`
}

func (c *RecipeConstraints) Validate() error {
	if c == nil {
		return nil
	}

	if c.HydrationMin != nil && c.HydrationMax != nil && *c.HydrationMin > *c.HydrationMax {
		return ErrInvalidHydration
	}

	return nil
}

type RecipeDraft struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Author      string             `json:"author"`
	Dough       map[string]float64 `json:"dough"`
	Topping     map[string]float64 `json:"topping"`
	Steps       []string           `json:"steps"`
}

func (r RecipeDraft) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidRecipeData)
	}
	if len(r.Dough) == 0 {
		return fmt.Errorf("%w: dough must contain ingredients", ErrInvalidRecipeData)
	}
	if len(r.Topping) == 0 {
		return fmt.Errorf("%w: topping must contain ingredients", ErrInvalidRecipeData)
	}
	return nil
}

func (m GenerationMode) Validate(prompt string) error {
	switch m {
	case ModeRandom:
		return nil
	case ModePrompt:
		if strings.TrimSpace(prompt) == "" {
			return ErrPromptRequired
		}
		return nil
	default:
		return ErrInvalidMode
	}
}

type ToolDefinition struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	InputSchema  any    `json:"inputSchema"`
	OutputSchema any    `json:"outputSchema"`
}
