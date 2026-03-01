package application

import (
	"context"
	"errors"
	"testing"

	"github.com/cfioretti/recipe-mcp-server/internal/domain"
)

type stubProvider struct {
	generateResp *domain.RecipeDraft
	generateErr  error
	customResp   *domain.RecipeDraft
	customErr    error
}

func (s stubProvider) GenerateRecipe(_ context.Context, _ GenerateRecipeCommand) (*domain.RecipeDraft, error) {
	return s.generateResp, s.generateErr
}

func (s stubProvider) CustomizeRecipe(_ context.Context, _ CustomizeRecipeCommand) (*domain.RecipeDraft, error) {
	return s.customResp, s.customErr
}

func validDraft() *domain.RecipeDraft {
	return &domain.RecipeDraft{
		Name:    "Generated Pizza",
		Dough:   map[string]float64{"flour": 1000, "water": 700},
		Topping: map[string]float64{"mozzarella": 300},
	}
}

func TestListToolsHasExpectedEntries(t *testing.T) {
	service := NewRecipeToolsService(stubProvider{})
	tools := service.ListTools()
	if len(tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(tools))
	}

	expected := []string{"list_tools", "generate_recipe", "customize_recipe"}
	for i, name := range expected {
		if tools[i].Name != name {
			t.Fatalf("expected tool[%d]=%s, got %s", i, name, tools[i].Name)
		}
	}
}

func TestGenerateRecipeValidatesPromptMode(t *testing.T) {
	service := NewRecipeToolsService(stubProvider{generateResp: validDraft()})

	_, err := service.GenerateRecipe(context.Background(), GenerateRecipeCommand{
		Mode:   domain.ModePrompt,
		Prompt: "",
	})

	if !errors.Is(err, domain.ErrPromptRequired) {
		t.Fatalf("expected ErrPromptRequired, got %v", err)
	}
}

func TestGenerateRecipeHandlesNilProviderResponse(t *testing.T) {
	service := NewRecipeToolsService(stubProvider{generateResp: nil})

	_, err := service.GenerateRecipe(context.Background(), GenerateRecipeCommand{
		Mode: domain.ModeRandom,
	})

	if !errors.Is(err, ErrEmptyProviderResponse) {
		t.Fatalf("expected ErrEmptyProviderResponse, got %v", err)
	}
}

func TestCustomizeRecipeSuccess(t *testing.T) {
	service := NewRecipeToolsService(stubProvider{customResp: validDraft()})

	_, err := service.CustomizeRecipe(context.Background(), CustomizeRecipeCommand{
		Mode:       domain.ModePrompt,
		Prompt:     "more crispy",
		BaseRecipe: *validDraft(),
	})

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}
