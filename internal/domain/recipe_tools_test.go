package domain

import (
	"errors"
	"testing"
)

func TestGenerationModeValidate(t *testing.T) {
	tests := []struct {
		name    string
		mode    GenerationMode
		prompt  string
		wantErr error
	}{
		{name: "random mode is valid", mode: ModeRandom, prompt: "", wantErr: nil},
		{name: "prompt mode requires prompt", mode: ModePrompt, prompt: "", wantErr: ErrPromptRequired},
		{name: "prompt mode with prompt is valid", mode: ModePrompt, prompt: "high hydration", wantErr: nil},
		{name: "unknown mode is invalid", mode: "unexpected", prompt: "", wantErr: ErrInvalidMode},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.mode.Validate(tc.prompt)
			if tc.wantErr == nil && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected error %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestRecipeConstraintsValidate(t *testing.T) {
	min := 80.0
	max := 70.0
	constraints := &RecipeConstraints{
		HydrationMin: &min,
		HydrationMax: &max,
	}

	if err := constraints.Validate(); !errors.Is(err, ErrInvalidHydration) {
		t.Fatalf("expected ErrInvalidHydration, got %v", err)
	}
}

func TestRecipeDraftValidate(t *testing.T) {
	valid := RecipeDraft{
		Name:    "Test Pizza",
		Dough:   map[string]float64{"flour": 1000, "water": 700},
		Topping: map[string]float64{"mozzarella": 300},
	}

	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid recipe, got error %v", err)
	}

	invalid := RecipeDraft{
		Name:    "",
		Dough:   map[string]float64{},
		Topping: map[string]float64{},
	}
	if err := invalid.Validate(); !errors.Is(err, ErrInvalidRecipeData) {
		t.Fatalf("expected ErrInvalidRecipeData, got %v", err)
	}
}
