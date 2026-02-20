package models

import (
	"github.com/lorenzotomasdiez/tenth-man-rule/internal/openrouter"
)

// Registry holds a filtered list of free models.
type Registry struct {
	free []openrouter.Model
}

// NewRegistry creates a registry, keeping only free models (Prompt == "0" and Completion == "0").
// Models with nil Pricing are excluded.
func NewRegistry(models []openrouter.Model) *Registry {
	var free []openrouter.Model
	for _, m := range models {
		if m.Pricing == nil {
			continue
		}
		if m.Pricing.Prompt == "0" && m.Pricing.Completion == "0" {
			free = append(free, m)
		}
	}
	return &Registry{free: free}
}

// FreeModels returns all free models in the registry.
func (r *Registry) FreeModels() []openrouter.Model {
	return r.free
}

// SelectModels returns n models from the free list, cycling if n > available.
func (r *Registry) SelectModels(n int) []openrouter.Model {
	if len(r.free) == 0 {
		return nil
	}
	selected := make([]openrouter.Model, n)
	for i := range n {
		selected[i] = r.free[i%len(r.free)]
	}
	return selected
}

// DefaultFreeModels returns a hardcoded fallback list of known free models.
func DefaultFreeModels() []openrouter.Model {
	return []openrouter.Model{
		{ID: "qwen/qwen3-235b-a22b:free", Name: "Qwen3 235B A22B", Pricing: &openrouter.Pricing{Prompt: "0", Completion: "0"}},
		{ID: "google/gemma-3n-e2b-it:free", Name: "Gemma 3n 2B", Pricing: &openrouter.Pricing{Prompt: "0", Completion: "0"}},
		{ID: "nvidia/nemotron-nano-9b-v2:free", Name: "Nemotron Nano 9B V2", Pricing: &openrouter.Pricing{Prompt: "0", Completion: "0"}},
		{ID: "qwen/qwen3-coder:free", Name: "Qwen3 Coder 480B A35B", Pricing: &openrouter.Pricing{Prompt: "0", Completion: "0"}},
		{ID: "openai/gpt-oss-120b:free", Name: "GPT OSS 120B", Pricing: &openrouter.Pricing{Prompt: "0", Completion: "0"}},
	}
}
