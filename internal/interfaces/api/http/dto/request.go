package dto

import (
	"github.com/cfioretti/recipe-mcp-server/internal/application"
	"github.com/cfioretti/recipe-mcp-server/internal/domain"
)

type RecipeConstraintsRequest struct {
	HydrationMin       *float64 `json:"hydrationMin,omitempty"`
	HydrationMax       *float64 `json:"hydrationMax,omitempty"`
	Vegetarian         *bool    `json:"vegetarian,omitempty"`
	IncludeIngredients []string `json:"includeIngredients,omitempty"`
	ExcludeIngredients []string `json:"excludeIngredients,omitempty"`
}

func (r *RecipeConstraintsRequest) ToDomain() *domain.RecipeConstraints {
	if r == nil {
		return nil
	}
	return &domain.RecipeConstraints{
		HydrationMin:       r.HydrationMin,
		HydrationMax:       r.HydrationMax,
		Vegetarian:         r.Vegetarian,
		IncludeIngredients: r.IncludeIngredients,
		ExcludeIngredients: r.ExcludeIngredients,
	}
}

type GenerateRecipeRequest struct {
	Mode        string                    `json:"mode" binding:"required,oneof=random prompt"`
	Prompt      string                    `json:"prompt,omitempty"`
	Constraints *RecipeConstraintsRequest `json:"constraints,omitempty"`
}

func (r GenerateRecipeRequest) ToApplication() application.GenerateRecipeCommand {
	return application.GenerateRecipeCommand{
		Mode:        domain.GenerationMode(r.Mode),
		Prompt:      r.Prompt,
		Constraints: r.Constraints.ToDomain(),
	}
}

type RecipeDraftRequest struct {
	Name        string             `json:"name" binding:"required"`
	Description string             `json:"description"`
	Author      string             `json:"author"`
	Dough       map[string]float64 `json:"dough" binding:"required"`
	Topping     map[string]float64 `json:"topping" binding:"required"`
	Steps       []string           `json:"steps"`
}

func (r RecipeDraftRequest) ToDomain() domain.RecipeDraft {
	return domain.RecipeDraft{
		Name:        r.Name,
		Description: r.Description,
		Author:      r.Author,
		Dough:       r.Dough,
		Topping:     r.Topping,
		Steps:       r.Steps,
	}
}

type CustomizeRecipeRequest struct {
	Mode        string                    `json:"mode" binding:"required,oneof=random prompt"`
	Prompt      string                    `json:"prompt,omitempty"`
	Constraints *RecipeConstraintsRequest `json:"constraints,omitempty"`
	BaseRecipe  RecipeDraftRequest        `json:"baseRecipe" binding:"required"`
}

func (r CustomizeRecipeRequest) ToApplication() application.CustomizeRecipeCommand {
	return application.CustomizeRecipeCommand{
		Mode:        domain.GenerationMode(r.Mode),
		Prompt:      r.Prompt,
		Constraints: r.Constraints.ToDomain(),
		BaseRecipe:  r.BaseRecipe.ToDomain(),
	}
}
