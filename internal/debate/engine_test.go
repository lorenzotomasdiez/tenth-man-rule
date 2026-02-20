package debate

import (
	"context"
	"fmt"
	"testing"

	"github.com/lorenzotomasdiez/tenth-man-rule/internal/openrouter"
)

// mockLLM returns canned responses, rotating through them.
type mockLLM struct {
	responses []string
	callCount int
}

func (m *mockLLM) ChatCompletion(_ context.Context, _ string, _ []openrouter.Message) (*openrouter.ChatResponse, error) {
	resp := m.responses[m.callCount%len(m.responses)]
	m.callCount++
	return &openrouter.ChatResponse{
		Choices: []openrouter.Choice{{Message: openrouter.Message{Role: "assistant", Content: resp}}},
	}, nil
}

// mockJudge returns consensus only when transcript.Rounds >= consensusAtRound.
type mockJudge struct {
	consensusAtRound int
	callCount        int
}

func (m *mockJudge) Evaluate(_ context.Context, transcript *Transcript) (*ConsensusResult, error) {
	m.callCount++
	if transcript.Rounds >= m.consensusAtRound {
		return &ConsensusResult{
			Detected:   true,
			Position:   "the consensus position",
			Score:      8,
			Dissenters: nil,
		}, nil
	}
	return &ConsensusResult{
		Detected: false,
		Score:    3,
	}, nil
}

// mockTenthMan builds a tenth man agent.
type mockTenthMan struct {
	buildCalled bool
}

func (m *mockTenthMan) BuildAgent(consensusPosition string, agentID int, model string) Agent {
	m.buildCalled = true
	return Agent{ID: agentID, Name: "Tenth Man", Model: model, Role: "tenth-man"}
}

func (m *mockTenthMan) SystemPrompt(_ string) string {
	return "You must argue the contrary."
}

func makeAgents(n int) []Agent {
	agents := make([]Agent, n)
	for i := range n {
		agents[i] = Agent{
			ID:    i + 1,
			Name:  fmt.Sprintf("Agent-%d", i+1),
			Model: fmt.Sprintf("model-%d", i+1),
			Role:  "debater",
		}
	}
	return agents
}

func TestEngineRunsMinimumRounds(t *testing.T) {
	agents := makeAgents(3)
	llm := &mockLLM{responses: []string{"I think X", "I think Y", "I think Z"}}
	judge := &mockJudge{consensusAtRound: 999} // never consensus
	tm := &mockTenthMan{}

	e := NewEngine("test topic", agents, llm, judge, tm, 5, 7)
	result, err := e.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Transcript.Rounds != 7 {
		t.Errorf("expected 7 rounds, got %d", result.Transcript.Rounds)
	}
	if result.Transcript.Phase != FreeDebate {
		t.Errorf("expected FreeDebate phase, got %d", result.Transcript.Phase)
	}
	// 3 agents * 7 rounds = 21 turns
	if len(result.Transcript.Turns) != 21 {
		t.Errorf("expected 21 turns, got %d", len(result.Transcript.Turns))
	}
	if tm.buildCalled {
		t.Error("tenth man should not be activated when no consensus")
	}
}

func TestEngineTransitionsToTenthManPhase(t *testing.T) {
	agents := makeAgents(3)
	llm := &mockLLM{responses: []string{"response"}}
	judge := &mockJudge{consensusAtRound: 5} // consensus at round 5
	tm := &mockTenthMan{}

	e := NewEngine("test topic", agents, llm, judge, tm, 5, 10)
	result, err := e.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Transcript.Phase != TenthManPhase {
		t.Errorf("expected TenthManPhase, got %d", result.Transcript.Phase)
	}
	if !tm.buildCalled {
		t.Error("expected tenth man to be activated")
	}
	// Phase 1: 5 rounds * 3 agents = 15 turns
	// Phase 2: 3 rounds * 4 agents (3 + tenth man) = 12 turns
	// Total: 27 turns
	expectedTurns := 27
	if len(result.Transcript.Turns) != expectedTurns {
		t.Errorf("expected %d turns, got %d", expectedTurns, len(result.Transcript.Turns))
	}
	// Total rounds: 5 (phase 1) + 3 (phase 2) = 8
	if result.Transcript.Rounds != 8 {
		t.Errorf("expected 8 rounds, got %d", result.Transcript.Rounds)
	}
}

func TestEngineRespectsContextCancellation(t *testing.T) {
	agents := makeAgents(3)
	callCount := 0
	llm := &mockLLM{responses: []string{"response"}}

	// Wrap mockLLM to cancel after first agent's call
	ctx, cancel := context.WithCancel(context.Background())
	cancellingLLM := &cancellingMockLLM{inner: llm, cancel: cancel, cancelAfter: 1, callCount: &callCount}

	judge := &mockJudge{consensusAtRound: 999}
	tm := &mockTenthMan{}

	e := NewEngine("test topic", agents, cancellingLLM, judge, tm, 5, 10)
	_, err := e.Run(ctx)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

type cancellingMockLLM struct {
	inner       *mockLLM
	cancel      context.CancelFunc
	cancelAfter int
	callCount   *int
}

func (m *cancellingMockLLM) ChatCompletion(ctx context.Context, model string, msgs []openrouter.Message) (*openrouter.ChatResponse, error) {
	*m.callCount++
	resp, err := m.inner.ChatCompletion(ctx, model, msgs)
	if *m.callCount >= m.cancelAfter {
		m.cancel()
	}
	return resp, err
}

func TestEngineReturnsFinalConsensus(t *testing.T) {
	agents := makeAgents(3)
	llm := &mockLLM{responses: []string{"response"}}
	judge := &mockJudge{consensusAtRound: 5}
	tm := &mockTenthMan{}

	e := NewEngine("test topic", agents, llm, judge, tm, 5, 10)
	result, err := e.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Consensus == nil {
		t.Fatal("expected final consensus result, got nil")
	}
	// Final consensus is re-evaluated after Phase 2
	if result.Consensus.Score == 0 {
		t.Error("expected non-zero consensus score")
	}
}

func TestEngineNoConsensusResult(t *testing.T) {
	agents := makeAgents(3)
	llm := &mockLLM{responses: []string{"response"}}
	judge := &mockJudge{consensusAtRound: 999}
	tm := &mockTenthMan{}

	e := NewEngine("test topic", agents, llm, judge, tm, 5, 7)
	result, err := e.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No consensus was reached, but last evaluation result should still be stored
	if result.Consensus == nil {
		t.Fatal("expected consensus result even when not detected")
	}
	if result.Consensus.Detected {
		t.Error("expected consensus not detected")
	}
}

func TestEngineTenthManGetsRealModel(t *testing.T) {
	agents := makeAgents(3)
	llm := &mockLLM{responses: []string{"response"}}
	judge := &mockJudge{consensusAtRound: 5}
	tm := &mockTenthMan{}

	e := NewEngine("test topic", agents, llm, judge, tm, 5, 10)
	e.SetTenthManModel("real-model-for-tenth")
	result, err := e.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find the tenth man's turns and verify model
	for _, turn := range result.Transcript.Turns {
		if turn.Agent.Role == "tenth-man" {
			if turn.Agent.Model != "real-model-for-tenth" {
				t.Errorf("expected tenth man model 'real-model-for-tenth', got %q", turn.Agent.Model)
			}
			return
		}
	}
	t.Error("no tenth man turns found")
}

func TestEngineTenthManUsesContraryPrompt(t *testing.T) {
	agents := makeAgents(3)
	captureLLM := &capturingMockLLM{responses: []string{"response"}}
	judge := &mockJudge{consensusAtRound: 5}
	tm := &mockTenthMan{}

	e := NewEngine("test topic", agents, captureLLM, judge, tm, 5, 10)
	e.SetTenthManModel("tm-model")
	_, err := e.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find messages sent to tenth man (after round 5)
	found := false
	for _, call := range captureLLM.calls {
		if call.model == "tm-model" {
			// System prompt should contain contrarian mandate, not generic debater prompt
			systemMsg := call.messages[0].Content
			if systemMsg == fmt.Sprintf("You are %s, a debate participant. The topic is: %s. Provide your analysis and perspective. Be concise but thorough.", "Tenth Man", "test topic") {
				t.Error("tenth man should NOT get the generic debater prompt")
			}
			if systemMsg != "You must argue the contrary." {
				t.Errorf("expected tenth man contrarian prompt, got %q", systemMsg)
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("no LLM calls found for tenth man model")
	}
}

func TestEnginePhase2PromptsEngageWithTenthMan(t *testing.T) {
	agents := makeAgents(3)
	captureLLM := &capturingMockLLM{responses: []string{"response"}}
	judge := &mockJudge{consensusAtRound: 5}
	tm := &mockTenthMan{}

	e := NewEngine("test topic", agents, captureLLM, judge, tm, 5, 10)
	e.SetTenthManModel("tm-model")
	_, err := e.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find Phase 2 calls for original agents (not tenth man)
	// They should have a modified system prompt mentioning the Tenth Man
	for _, call := range captureLLM.calls {
		if call.model != "tm-model" && call.round > 5 {
			systemMsg := call.messages[0].Content
			if systemMsg == fmt.Sprintf("You are %s, a debate participant. The topic is: %s. Provide your analysis and perspective. Be concise but thorough.", call.agentName, "test topic") {
				t.Errorf("Phase 2 agent %s should have modified prompt mentioning Tenth Man engagement", call.agentName)
			}
		}
	}
}

func TestEngineCallsOnTurnCallback(t *testing.T) {
	agents := makeAgents(2)
	llm := &mockLLM{responses: []string{"hello", "world"}}
	judge := &mockJudge{consensusAtRound: 999}
	tm := &mockTenthMan{}

	e := NewEngine("test topic", agents, llm, judge, tm, 1, 1)
	var turns []Turn
	e.OnTurn = func(turn Turn) {
		turns = append(turns, turn)
	}
	_, err := e.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(turns) != 2 {
		t.Fatalf("expected 2 OnTurn callbacks, got %d", len(turns))
	}
	if turns[0].Content != "hello" {
		t.Errorf("expected first turn content 'hello', got %q", turns[0].Content)
	}
}

func TestEngineCallsOnPhaseCallback(t *testing.T) {
	agents := makeAgents(3)
	llm := &mockLLM{responses: []string{"response"}}
	judge := &mockJudge{consensusAtRound: 5}
	tm := &mockTenthMan{}

	e := NewEngine("test topic", agents, llm, judge, tm, 5, 10)
	e.SetTenthManModel("tm-model")
	var phases []Phase
	e.OnPhase = func(phase Phase) {
		phases = append(phases, phase)
	}
	_, err := e.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(phases) != 2 {
		t.Fatalf("expected 2 OnPhase callbacks, got %d", len(phases))
	}
	if phases[0] != FreeDebate {
		t.Errorf("expected first phase FreeDebate, got %d", phases[0])
	}
	if phases[1] != TenthManPhase {
		t.Errorf("expected second phase TenthManPhase, got %d", phases[1])
	}
}

// capturingMockLLM records all calls for inspection.
type llmCall struct {
	model     string
	agentName string
	messages  []openrouter.Message
	round     int
}

type capturingMockLLM struct {
	responses []string
	callCount int
	calls     []llmCall
}

func (m *capturingMockLLM) ChatCompletion(_ context.Context, model string, msgs []openrouter.Message) (*openrouter.ChatResponse, error) {
	// Extract agent name from system prompt (rough heuristic)
	agentName := ""
	round := 0
	if len(msgs) > 0 {
		agentName = model // fallback
	}
	// Count user messages to estimate round
	for _, msg := range msgs {
		if msg.Role == "user" && msg.Content != "It's your turn to speak. Provide your perspective on the topic." {
			round++
		}
	}
	m.calls = append(m.calls, llmCall{model: model, agentName: agentName, messages: msgs, round: round})
	resp := m.responses[m.callCount%len(m.responses)]
	m.callCount++
	return &openrouter.ChatResponse{
		Choices: []openrouter.Choice{{Message: openrouter.Message{Role: "assistant", Content: resp}}},
	}, nil
}

func TestEngineBuildsTranscript(t *testing.T) {
	agents := makeAgents(2)
	llm := &mockLLM{responses: []string{"alpha", "beta"}}
	judge := &mockJudge{consensusAtRound: 999}
	tm := &mockTenthMan{}

	e := NewEngine("climate change", agents, llm, judge, tm, 1, 1)
	result, err := e.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	transcript := result.Transcript
	if transcript.Topic != "climate change" {
		t.Errorf("expected topic 'climate change', got %q", transcript.Topic)
	}
	if len(transcript.Turns) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(transcript.Turns))
	}

	turn0 := transcript.Turns[0]
	if turn0.Round != 1 {
		t.Errorf("turn 0: expected round 1, got %d", turn0.Round)
	}
	if turn0.Agent.Name != "Agent-1" {
		t.Errorf("turn 0: expected Agent-1, got %s", turn0.Agent.Name)
	}
	if turn0.Content != "alpha" {
		t.Errorf("turn 0: expected 'alpha', got %q", turn0.Content)
	}

	turn1 := transcript.Turns[1]
	if turn1.Round != 1 {
		t.Errorf("turn 1: expected round 1, got %d", turn1.Round)
	}
	if turn1.Agent.Name != "Agent-2" {
		t.Errorf("turn 1: expected Agent-2, got %s", turn1.Agent.Name)
	}
	if turn1.Content != "beta" {
		t.Errorf("turn 1: expected 'beta', got %q", turn1.Content)
	}
}
