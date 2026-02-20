package tenthman_test

import (
	"strings"
	"testing"

	"github.com/lorenzotomasdiez/tenth-man-rule/internal/debate/tenthman"
)

func TestBuildAgent(t *testing.T) {
	a := tenthman.NewActivator()
	agent := a.BuildAgent("AI is beneficial", 10, "meta-llama/llama-3-8b-instruct:free")

	if agent.ID != 10 {
		t.Errorf("expected ID 10, got %d", agent.ID)
	}
	if agent.Name != "The Tenth Man" {
		t.Errorf("expected Name 'The Tenth Man', got %q", agent.Name)
	}
	if agent.Role != "tenth-man" {
		t.Errorf("expected Role 'tenth-man', got %q", agent.Role)
	}
	if agent.Model != "meta-llama/llama-3-8b-instruct:free" {
		t.Errorf("expected Model 'meta-llama/llama-3-8b-instruct:free', got %q", agent.Model)
	}
}

func TestSystemPromptContainsConsensusPosition(t *testing.T) {
	a := tenthman.NewActivator()
	prompt := a.SystemPrompt("AI regulation is necessary")

	if !strings.Contains(prompt, "AI regulation is necessary") {
		t.Error("expected prompt to contain the consensus position")
	}
}

func TestSystemPromptContainsContraryMandate(t *testing.T) {
	a := tenthman.NewActivator()
	prompt := a.SystemPrompt("some consensus")

	if !strings.Contains(prompt, "OBLIGATED") {
		t.Error("expected prompt to contain 'OBLIGATED'")
	}
	if !strings.Contains(prompt, "contrary") {
		t.Error("expected prompt to contain 'contrary'")
	}
}
