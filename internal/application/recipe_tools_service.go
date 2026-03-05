package application

import (
	"context"
	"errors"

	"github.com/cfioretti/recipe-mcp-server/internal/domain"
)

var ErrEmptyProviderResponse = errors.New("provider returned empty recipe draft")

type GenerateRecipeCommand struct {
	Mode           domain.GenerationMode
	Prompt         string
	OutputContract *domain.OutputContract
}

type CustomizeRecipeCommand struct {
	Mode       domain.GenerationMode
	Prompt     string
	BaseRecipe domain.RecipeDraft
}

type RecipeGenerationProvider interface {
	GenerateRecipe(ctx context.Context, request GenerateRecipeCommand) (*domain.RecipeDraft, error)
	CustomizeRecipe(ctx context.Context, request CustomizeRecipeCommand) (*domain.RecipeDraft, error)
}

type RecipeToolsService struct {
	provider RecipeGenerationProvider
}

func NewRecipeToolsService(provider RecipeGenerationProvider) *RecipeToolsService {
	return &RecipeToolsService{provider: provider}
}

func (s *RecipeToolsService) ListTools() []domain.ToolDefinition {
	return []domain.ToolDefinition{
		{
			Name:        "list_tools",
			Description: "List available MCP recipe generation tools.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
			OutputSchema: map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
				},
			},
		},
		{
			Name:        "generate_recipe",
			Description: "Generate a recipe draft using random or prompt mode.",
			InputSchema: map[string]any{
				"type": "object",
				"required": []string{
					"mode",
				},
				"properties": map[string]any{
					"mode": map[string]any{
						"type": "string",
						"enum": []string{"random", "prompt"},
					},
					"prompt": map[string]any{
						"type": "string",
					},
					"outputContract": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"requiredDoughIngredients": map[string]any{
								"type":  "array",
								"items": map[string]any{"type": "string"},
							},
							"requiredToppingIngredients": map[string]any{
								"type":  "array",
								"items": map[string]any{"type": "string"},
							},
						},
					},
				},
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"recipeDraft": map[string]any{
						"type": "object",
					},
				},
			},
		},
		{
			Name:        "customize_recipe",
			Description: "Customize an existing recipe draft with a prompt.",
			InputSchema: map[string]any{
				"type": "object",
				"required": []string{
					"mode",
					"baseRecipe",
				},
				"properties": map[string]any{
					"mode": map[string]any{
						"type": "string",
						"enum": []string{"random", "prompt"},
					},
					"prompt": map[string]any{
						"type": "string",
					},
					"baseRecipe": map[string]any{
						"type": "object",
					},
				},
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"recipeDraft": map[string]any{
						"type": "object",
					},
				},
			},
		},
	}
}

func (s *RecipeToolsService) GenerateRecipe(ctx context.Context, request GenerateRecipeCommand) (*domain.RecipeDraft, error) {
	if err := request.Mode.Validate(request.Prompt); err != nil {
		return nil, err
	}
	effectiveContract := request.OutputContract.Effective()
	if err := effectiveContract.Validate(); err != nil {
		return nil, err
	}

	recipe, err := s.provider.GenerateRecipe(ctx, request)
	if err != nil {
		return nil, err
	}
	if recipe == nil {
		return nil, ErrEmptyProviderResponse
	}
	if err := recipe.Validate(); err != nil {
		return nil, err
	}
	if err := recipe.ValidateAgainstContract(effectiveContract); err != nil {
		return nil, err
	}

	return recipe, nil
}

func (s *RecipeToolsService) CustomizeRecipe(ctx context.Context, request CustomizeRecipeCommand) (*domain.RecipeDraft, error) {
	if err := request.Mode.Validate(request.Prompt); err != nil {
		return nil, err
	}
	if err := request.BaseRecipe.Validate(); err != nil {
		return nil, err
	}

	recipe, err := s.provider.CustomizeRecipe(ctx, request)
	if err != nil {
		return nil, err
	}
	if recipe == nil {
		return nil, ErrEmptyProviderResponse
	}
	if err := recipe.Validate(); err != nil {
		return nil, err
	}

	return recipe, nil
}
