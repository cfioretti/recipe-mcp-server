package dto

import (
	"github.com/cfioretti/recipe-mcp-server/internal/application"
	"github.com/cfioretti/recipe-mcp-server/internal/domain"
)

type GenerateRecipeRequest struct {
	Mode           string                 `json:"mode" binding:"required,oneof=random prompt"`
	Prompt         string                 `json:"prompt,omitempty"`
	OutputContract *OutputContractRequest `json:"outputContract,omitempty"`
}

type OutputContractRequest struct {
	RequiredDoughIngredients   []string `json:"requiredDoughIngredients,omitempty"`
	RequiredToppingIngredients []string `json:"requiredToppingIngredients,omitempty"`
}

func (r *OutputContractRequest) ToDomain() *domain.OutputContract {
	if r == nil {
		return nil
	}
	return &domain.OutputContract{
		RequiredDoughIngredients:   r.RequiredDoughIngredients,
		RequiredToppingIngredients: r.RequiredToppingIngredients,
	}
}

func (r GenerateRecipeRequest) ToApplication() application.GenerateRecipeCommand {
	return application.GenerateRecipeCommand{
		Mode:           domain.GenerationMode(r.Mode),
		Prompt:         r.Prompt,
		OutputContract: r.OutputContract.ToDomain(),
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
	Mode       string             `json:"mode" binding:"required,oneof=random prompt"`
	Prompt     string             `json:"prompt,omitempty"`
	BaseRecipe RecipeDraftRequest `json:"baseRecipe" binding:"required"`
}

func (r CustomizeRecipeRequest) ToApplication() application.CustomizeRecipeCommand {
	return application.CustomizeRecipeCommand{
		Mode:       domain.GenerationMode(r.Mode),
		Prompt:     r.Prompt,
		BaseRecipe: r.BaseRecipe.ToDomain(),
	}
}
