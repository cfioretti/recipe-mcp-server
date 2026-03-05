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

func TestOutputContractAndRecipeDraftContractValidation(t *testing.T) {
	contract := DefaultOutputContract()
	if err := contract.Validate(); err != nil {
		t.Fatalf("expected valid default contract, got %v", err)
	}

	recipe := RecipeDraft{
		Name:    "Contract Pizza",
		Dough:   map[string]float64{"flour": 1000, "water": 680},
		Topping: map[string]float64{"mozzarella": 250},
	}
	if err := recipe.ValidateAgainstContract(contract); err != nil {
		t.Fatalf("expected recipe to satisfy default contract, got %v", err)
	}

	invalidContract := OutputContract{RequiredDoughIngredients: []string{"flour", ""}}
	if err := invalidContract.Validate(); !errors.Is(err, ErrOutputContractInvalid) {
		t.Fatalf("expected ErrOutputContractInvalid, got %v", err)
	}

	missingWater := RecipeDraft{
		Name:    "Broken Pizza",
		Dough:   map[string]float64{"flour": 1000},
		Topping: map[string]float64{"mozzarella": 250},
	}
	if err := missingWater.ValidateAgainstContract(DefaultOutputContract()); !errors.Is(err, ErrOutputContractViolated) {
		t.Fatalf("expected ErrOutputContractViolated, got %v", err)
	}
}

func TestValidateAgainstContract_CaseInsensitive(t *testing.T) {
	contract := DefaultOutputContract()

	recipe := RecipeDraft{
		Name:    "Case Test Pizza",
		Dough:   map[string]float64{"Flour": 1000, "WATER": 680},
		Topping: map[string]float64{"mozzarella": 250},
	}
	if err := recipe.ValidateAgainstContract(contract); err != nil {
		t.Fatalf("expected case-insensitive match to pass, got %v", err)
	}

	contractWithTopping := OutputContract{
		RequiredDoughIngredients:   []string{"flour"},
		RequiredToppingIngredients: []string{"mozzarella"},
	}
	recipeMixed := RecipeDraft{
		Name:    "Mixed Case",
		Dough:   map[string]float64{"FLOUR": 500},
		Topping: map[string]float64{"Mozzarella": 200},
	}
	if err := recipeMixed.ValidateAgainstContract(contractWithTopping); err != nil {
		t.Fatalf("expected case-insensitive topping match to pass, got %v", err)
	}
}
