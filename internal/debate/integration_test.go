package debate

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/lorenzotomasdiez/tenth-man-rule/internal/openrouter"
)

// simulatedLLM provides realistic varied responses based on agent/round context.
type simulatedLLM struct {
	callCount int
}

func (m *simulatedLLM) ChatCompletion(_ context.Context, model string, msgs []openrouter.Message) (*openrouter.ChatResponse, error) {
	m.callCount++
	systemPrompt := ""
	if len(msgs) > 0 {
		systemPrompt = msgs[0].Content
	}

	var response string
	if strings.Contains(systemPrompt, "OBLIGATED") {
		// Tenth Man response
		response = "I must challenge the consensus. While the majority argues for strict regulation, " +
			"there are significant risks: regulatory capture by incumbents, stifling innovation in " +
			"developing nations, and the impossibility of regulating open-source models. The consensus " +
			"ignores these critical failure modes."
	} else if strings.Contains(systemPrompt, "engage with the Tenth Man") {
		// Phase 2 debater response
		response = "The Tenth Man raises valid points about regulatory capture, but the risks of " +
			"unregulated AI deployment far outweigh the costs of imperfect regulation. We can " +
			"design adaptive frameworks that avoid the pitfalls mentioned."
	} else {
		// Phase 1 debater responses — converge toward consensus
		responses := []string{
			"AI regulation is essential. Without guardrails, we risk deploying systems that cause widespread harm.",
			"I agree that regulation is needed. The EU AI Act provides a reasonable framework to build upon.",
			"The evidence strongly supports regulatory intervention. Self-regulation has failed in every tech sector.",
			"We must act now. The pace of AI development demands immediate regulatory frameworks.",
			"Regulation is clearly the responsible path forward. The question is implementation, not whether to regulate.",
		}
		response = responses[m.callCount%len(responses)]
	}

	return &openrouter.ChatResponse{
		Choices: []openrouter.Choice{{Message: openrouter.Message{Role: "assistant", Content: response}}},
	}, nil
}

// simulatedJudge provides realistic consensus detection.
type simulatedJudge struct {
	callCount int
}

func (j *simulatedJudge) Evaluate(_ context.Context, transcript *Transcript) (*ConsensusResult, error) {
	j.callCount++

	// Count unique positions in recent turns
	if transcript.Rounds < 5 {
		return &ConsensusResult{Detected: false, Score: 3, Position: ""}, nil
	}

	// After round 5, detect consensus
	return &ConsensusResult{
		Detected:   true,
		Position:   "AI regulation is essential and should be implemented through adaptive frameworks similar to the EU AI Act",
		Score:      8,
		Dissenters: nil,
	}, nil
}

// simulatedTenthMan provides realistic tenth man activation.
type simulatedTenthMan struct {
	activated         bool
	consensusReceived string
}

func (m *simulatedTenthMan) BuildAgent(consensusPosition string, agentID int, model string) Agent {
	m.activated = true
	m.consensusReceived = consensusPosition
	return Agent{ID: agentID, Name: "The Tenth Man", Model: model, Role: "tenth-man"}
}

func (m *simulatedTenthMan) SystemPrompt(consensusPosition string) string {
	return fmt.Sprintf(
		"You are The Tenth Man. The group has reached consensus: %s. "+
			"You are OBLIGATED to argue the contrary position with genuine analytical rigor.",
		consensusPosition,
	)
}

func TestIntegrationFullDebateFlow(t *testing.T) {
	agents := makeAgents(5)
	llm := &simulatedLLM{}
	judge := &simulatedJudge{}
	tm := &simulatedTenthMan{}

	engine := NewEngine("Should AI be regulated?", agents, llm, judge, tm, 5, 10)
	engine.SetTenthManModel("test-model-tenth")

	// Track callbacks
	var turnCount int
	var phasesSeen []Phase
	engine.OnTurn = func(turn Turn) {
		turnCount++
	}
	engine.OnPhase = func(phase Phase) {
		phasesSeen = append(phasesSeen, phase)
	}

	result, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("debate failed: %v", err)
	}

	// Verify two-phase structure
	if len(phasesSeen) != 2 {
		t.Fatalf("expected 2 phase transitions, got %d", len(phasesSeen))
	}
	if phasesSeen[0] != FreeDebate {
		t.Errorf("first phase should be FreeDebate, got %d", phasesSeen[0])
	}
	if phasesSeen[1] != TenthManPhase {
		t.Errorf("second phase should be TenthManPhase, got %d", phasesSeen[1])
	}

	// Verify tenth man was activated
	if !tm.activated {
		t.Error("tenth man was not activated")
	}
	if tm.consensusReceived == "" {
		t.Error("tenth man did not receive consensus position")
	}

	// Verify transcript structure
	transcript := result.Transcript
	if transcript.Topic != "Should AI be regulated?" {
		t.Errorf("wrong topic: %s", transcript.Topic)
	}
	if transcript.Phase != TenthManPhase {
		t.Errorf("final phase should be TenthManPhase, got %d", transcript.Phase)
	}

	// Phase 1: 5 rounds * 5 agents = 25 turns
	// Phase 2: 3 rounds * 6 agents (5 + tenth man) = 18 turns
	// Total: 43 turns
	expectedTurns := 43
	if len(transcript.Turns) != expectedTurns {
		t.Errorf("expected %d turns, got %d", expectedTurns, len(transcript.Turns))
	}
	if turnCount != expectedTurns {
		t.Errorf("OnTurn called %d times, expected %d", turnCount, expectedTurns)
	}

	// Verify tenth man turns exist and have the right model
	tenthManTurns := 0
	for _, turn := range transcript.Turns {
		if turn.Agent.Role == "tenth-man" {
			tenthManTurns++
			if turn.Agent.Model != "test-model-tenth" {
				t.Errorf("tenth man model = %q, want test-model-tenth", turn.Agent.Model)
			}
			if !strings.Contains(turn.Content, "challenge the consensus") {
				t.Errorf("tenth man response doesn't contain contrarian content: %s", turn.Content)
			}
		}
	}
	if tenthManTurns != 3 {
		t.Errorf("expected 3 tenth man turns (one per phase 2 round), got %d", tenthManTurns)
	}

	// Verify phase 2 debater responses engage with tenth man
	phase2DebaterTurns := 0
	for _, turn := range transcript.Turns {
		if turn.Round > 5 && turn.Agent.Role == "debater" {
			phase2DebaterTurns++
			if !strings.Contains(turn.Content, "Tenth Man") {
				t.Errorf("phase 2 debater response should engage with Tenth Man: %s", turn.Content)
			}
		}
	}
	if phase2DebaterTurns != 15 { // 3 rounds * 5 original agents
		t.Errorf("expected 15 phase 2 debater turns, got %d", phase2DebaterTurns)
	}

	// Verify consensus result
	if result.Consensus == nil {
		t.Fatal("expected consensus result")
	}
	if !result.Consensus.Detected {
		t.Error("expected consensus detected in final evaluation")
	}

	// Verify LLM was called for every turn
	if llm.callCount != expectedTurns {
		t.Errorf("LLM called %d times, expected %d", llm.callCount, expectedTurns)
	}

	// Verify judge was called appropriately
	// Round 5 consensus check + final Phase 2 evaluation = 2 calls
	if judge.callCount != 2 {
		t.Errorf("judge called %d times, expected 2", judge.callCount)
	}
}

func TestIntegrationNoConsensusReached(t *testing.T) {
	agents := makeAgents(4)
	llm := &simulatedLLM{}
	// Judge that never detects consensus
	judge := &mockJudge{consensusAtRound: 999}
	tm := &simulatedTenthMan{}

	engine := NewEngine("Is the universe deterministic?", agents, llm, judge, tm, 3, 5)

	var phasesSeen []Phase
	engine.OnPhase = func(phase Phase) {
		phasesSeen = append(phasesSeen, phase)
	}

	result, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("debate failed: %v", err)
	}

	// Should only see FreeDebate phase
	if len(phasesSeen) != 1 {
		t.Fatalf("expected 1 phase transition, got %d", len(phasesSeen))
	}
	if phasesSeen[0] != FreeDebate {
		t.Errorf("should stay in FreeDebate, got %d", phasesSeen[0])
	}

	// Tenth man should NOT be activated
	if tm.activated {
		t.Error("tenth man should not be activated without consensus")
	}

	// 5 rounds * 4 agents = 20 turns
	if len(result.Transcript.Turns) != 20 {
		t.Errorf("expected 20 turns, got %d", len(result.Transcript.Turns))
	}

	// Consensus should exist but not be detected
	if result.Consensus == nil {
		t.Fatal("expected consensus result even without detection")
	}
	if result.Consensus.Detected {
		t.Error("consensus should not be detected")
	}
}
