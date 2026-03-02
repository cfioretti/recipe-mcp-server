package ai

import "testing"

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
