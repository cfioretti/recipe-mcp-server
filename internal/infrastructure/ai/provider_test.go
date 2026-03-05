package ai

import (
	"testing"
)

func TestParseRecipeDraftFromText_WithWrappedJSONInMarkdown(t *testing.T) {
	raw := "```json\n{\"recipeDraft\":{\"name\":\"Margherita\",\"description\":\"Classic\",\"author\":\"AI\",\"dough\":{\"flour\":500},\"topping\":{\"tomato\":120},\"steps\":[\"Mix\",\"Bake\"]}}\n```"

	draft, err := parseRecipeDraftFromText(raw)
	if err != nil {
		t.Fatalf("expected parse to succeed, got error: %v", err)
	}
	if draft.Name != "Margherita" {
		t.Fatalf("expected name Margherita, got %q", draft.Name)
	}
}

func TestParseRecipeDraftFromText_WithLeadingTextAndTrailingJSON(t *testing.T) {
	raw := "Here is your recipe in JSON format:\n" +
		"{\"recipeDraft\":{\"name\":\"Diavola\",\"description\":\"Spicy\",\"author\":\"AI\",\"dough\":{\"flour\":500},\"topping\":{\"salami\":80},\"steps\":[\"Prep\",\"Bake\"]}}\n" +
		"Some additional text"

	draft, err := parseRecipeDraftFromText(raw)
	if err != nil {
		t.Fatalf("expected parse to succeed, got error: %v", err)
	}
	if draft.Name != "Diavola" {
		t.Fatalf("expected name Diavola, got %q", draft.Name)
	}
}

func TestParseRecipeDraftFromText_NormalizesKeysToLowercase(t *testing.T) {
	raw := `{"recipeDraft":{"name":"Pizza","description":"Test","author":"AI","dough":{"Flour":500,"Water":300,"Salt":10},"topping":{"Mozzarella":200},"steps":["Bake"]}}`

	draft, err := parseRecipeDraftFromText(raw)
	if err != nil {
		t.Fatalf("expected parse to succeed, got error: %v", err)
	}
	if _, ok := draft.Dough["flour"]; !ok {
		t.Fatalf("expected lowercase key 'flour' in dough, got keys: %v", draft.Dough)
	}
	if _, ok := draft.Dough["water"]; !ok {
		t.Fatalf("expected lowercase key 'water' in dough, got keys: %v", draft.Dough)
	}
	if _, ok := draft.Topping["mozzarella"]; !ok {
		t.Fatalf("expected lowercase key 'mozzarella' in topping, got keys: %v", draft.Topping)
	}
}

func TestParseRecipeDraftFromText_NormalizesAliasesToCanonical(t *testing.T) {
	raw := `{"name":"Pizza","description":"Test","author":"AI","dough":{"All-Purpose Flour":500,"Warm Water":300},"topping":{"tomato":120},"steps":["Bake"]}`

	draft, err := parseRecipeDraftFromText(raw)
	if err != nil {
		t.Fatalf("expected parse to succeed, got error: %v", err)
	}
	if _, ok := draft.Dough["flour"]; !ok {
		t.Fatalf("expected alias 'All-Purpose Flour' resolved to 'flour', got keys: %v", draft.Dough)
	}
	if _, ok := draft.Dough["water"]; !ok {
		t.Fatalf("expected alias 'Warm Water' resolved to 'water', got keys: %v", draft.Dough)
	}
}

func TestNormalizeIngredientMap_MergesDuplicateKeys(t *testing.T) {
	input := map[string]float64{
		"Flour":       300,
		"bread flour": 200,
		"Water":       400,
	}

	result := normalizeIngredientMap(input)
	if result["flour"] != 500 {
		t.Fatalf("expected merged flour=500, got %v", result["flour"])
	}
	if result["water"] != 400 {
		t.Fatalf("expected water=400, got %v", result["water"])
	}
}
