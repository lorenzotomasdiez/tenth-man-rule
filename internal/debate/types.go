package debate

import (
	"context"

	"github.com/lorenzotomasdiez/tenth-man-rule/internal/openrouter"
)

// Phase represents the current phase of the debate.
type Phase int

const (
	FreeDebate Phase = iota
	TenthManPhase
)

// Agent represents a debate participant.
type Agent struct {
	ID    int
	Name  string
	Model string // OpenRouter model ID
	Role  string // "debater" or "tenth-man"
}

// Turn represents a single agent's contribution in a round.
type Turn struct {
	Round   int
	Agent   Agent
	Content string
}

// Transcript holds the full state of a debate.
type Transcript struct {
	Topic  string
	Turns  []Turn
	Phase  Phase
	Rounds int
}

// LLMClient interface so we can mock the OpenRouter client.
type LLMClient interface {
	ChatCompletion(ctx context.Context, model string, messages []openrouter.Message) (*openrouter.ChatResponse, error)
}

// ConsensusResult is used by the consensus judge.
type ConsensusResult struct {
	Detected   bool     `json:"consensus_detected"`
	Position   string   `json:"consensus_position"`
	Score      int      `json:"agreement_score"`
	Dissenters []string `json:"dissenting_agents"`
}

// ConsensusJudge interface so we can mock consensus detection.
type ConsensusJudge interface {
	Evaluate(ctx context.Context, transcript *Transcript) (*ConsensusResult, error)
}

// TenthManActivator interface for building tenth man agent.
type TenthManActivator interface {
	BuildAgent(consensusPosition string, agentID int, model string) Agent
	SystemPrompt(consensusPosition string) string
}

// Result holds the complete output of a debate run.
type Result struct {
	Transcript *Transcript
	Consensus  *ConsensusResult
}
