package domain

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidMode            = errors.New("invalid mode")
	ErrPromptRequired         = errors.New("prompt is required when mode is prompt")
	ErrInvalidRecipeData      = errors.New("recipe draft is invalid")
	ErrOutputContractInvalid  = errors.New("output contract is invalid")
	ErrOutputContractViolated = errors.New("recipe draft does not satisfy expected output format")
)

type GenerationMode string

const (
	ModeRandom GenerationMode = "random"
	ModePrompt GenerationMode = "prompt"
)

type RecipeDraft struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Author      string             `json:"author"`
	Dough       map[string]float64 `json:"dough"`
	Topping     map[string]float64 `json:"topping"`
	Steps       []string           `json:"steps"`
}

type OutputContract struct {
	RequiredDoughIngredients   []string `json:"requiredDoughIngredients,omitempty"`
	RequiredToppingIngredients []string `json:"requiredToppingIngredients,omitempty"`
}

func DefaultOutputContract() OutputContract {
	return OutputContract{
		RequiredDoughIngredients: []string{"flour", "water"},
	}
}

func (c *OutputContract) Effective() OutputContract {
	if c == nil {
		return DefaultOutputContract()
	}

	effective := *c
	if len(effective.RequiredDoughIngredients) == 0 {
		effective.RequiredDoughIngredients = DefaultOutputContract().RequiredDoughIngredients
	}
	return effective
}

func (c OutputContract) Validate() error {
	for _, ingredient := range c.RequiredDoughIngredients {
		if strings.TrimSpace(ingredient) == "" {
			return fmt.Errorf("%w: required dough ingredient cannot be empty", ErrOutputContractInvalid)
		}
	}
	for _, ingredient := range c.RequiredToppingIngredients {
		if strings.TrimSpace(ingredient) == "" {
			return fmt.Errorf("%w: required topping ingredient cannot be empty", ErrOutputContractInvalid)
		}
	}
	return nil
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

func (r RecipeDraft) ValidateAgainstContract(contract OutputContract) error {
	if err := contract.Validate(); err != nil {
		return err
	}

	for _, ingredient := range contract.RequiredDoughIngredients {
		if !hasKeyCaseInsensitive(r.Dough, ingredient) {
			return fmt.Errorf("%w: missing required dough ingredient %q", ErrOutputContractViolated, ingredient)
		}
	}
	for _, ingredient := range contract.RequiredToppingIngredients {
		if !hasKeyCaseInsensitive(r.Topping, ingredient) {
			return fmt.Errorf("%w: missing required topping ingredient %q", ErrOutputContractViolated, ingredient)
		}
	}

	return nil
}

func hasKeyCaseInsensitive(m map[string]float64, key string) bool {
	target := strings.ToLower(strings.TrimSpace(key))
	for k := range m {
		if strings.ToLower(strings.TrimSpace(k)) == target {
			return true
		}
	}
	return false
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
