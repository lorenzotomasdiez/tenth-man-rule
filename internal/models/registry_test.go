package models

import (
	"testing"

	"github.com/lorenzotomasdiez/tenth-man-rule/internal/openrouter"
)

func TestNewRegistryFiltersFreeModels(t *testing.T) {
	models := []openrouter.Model{
		{ID: "free-model", Name: "Free", Pricing: &openrouter.Pricing{Prompt: "0", Completion: "0"}},
		{ID: "paid-model", Name: "Paid", Pricing: &openrouter.Pricing{Prompt: "0.01", Completion: "0.02"}},
		{ID: "half-free", Name: "HalfFree", Pricing: &openrouter.Pricing{Prompt: "0", Completion: "0.01"}},
	}

	r := NewRegistry(models)
	free := r.FreeModels()

	if len(free) != 1 {
		t.Fatalf("expected 1 free model, got %d", len(free))
	}
	if free[0].ID != "free-model" {
		t.Fatalf("expected free-model, got %s", free[0].ID)
	}
}

func TestNewRegistryExcludesNilPricing(t *testing.T) {
	models := []openrouter.Model{
		{ID: "no-pricing", Name: "NoPricing", Pricing: nil},
		{ID: "free-model", Name: "Free", Pricing: &openrouter.Pricing{Prompt: "0", Completion: "0"}},
	}

	r := NewRegistry(models)
	free := r.FreeModels()

	if len(free) != 1 {
		t.Fatalf("expected 1 free model, got %d", len(free))
	}
	if free[0].ID != "free-model" {
		t.Fatalf("expected free-model, got %s", free[0].ID)
	}
}

func TestSelectModelsLessThanAvailable(t *testing.T) {
	models := []openrouter.Model{
		{ID: "a", Name: "A", Pricing: &openrouter.Pricing{Prompt: "0", Completion: "0"}},
		{ID: "b", Name: "B", Pricing: &openrouter.Pricing{Prompt: "0", Completion: "0"}},
		{ID: "c", Name: "C", Pricing: &openrouter.Pricing{Prompt: "0", Completion: "0"}},
	}

	r := NewRegistry(models)
	selected := r.SelectModels(2)

	if len(selected) != 2 {
		t.Fatalf("expected 2 models, got %d", len(selected))
	}
	if selected[0].ID == selected[1].ID {
		t.Fatal("expected distinct models")
	}
}

func TestSelectModelsWrapsAround(t *testing.T) {
	models := []openrouter.Model{
		{ID: "a", Name: "A", Pricing: &openrouter.Pricing{Prompt: "0", Completion: "0"}},
		{ID: "b", Name: "B", Pricing: &openrouter.Pricing{Prompt: "0", Completion: "0"}},
	}

	r := NewRegistry(models)
	selected := r.SelectModels(5)

	if len(selected) != 5 {
		t.Fatalf("expected 5 models, got %d", len(selected))
	}
	// Should cycle: a, b, a, b, a
	if selected[0].ID != "a" || selected[1].ID != "b" || selected[2].ID != "a" {
		t.Fatalf("expected wrap-around pattern, got %v", selected)
	}
}

func TestDefaultFreeModelsNonEmpty(t *testing.T) {
	defaults := DefaultFreeModels()
	if len(defaults) == 0 {
		t.Fatal("expected non-empty default free models list")
	}
}
