package debate

import (
	"context"
	"fmt"
)

const tenthManRounds = 3

// Engine orchestrates a multi-agent debate.
type Engine struct {
	topic             string
	agents            []Agent
	llm               LLMClient
	judge             ConsensusJudge
	tenthMan          TenthManActivator
	transcript        *Transcript
	minRounds         int
	maxRounds         int
	tenthManModel     string
	consensusPosition string
	OnTurn            func(Turn)
	OnPhase           func(Phase)
}

// NewEngine creates a new debate engine.
func NewEngine(topic string, agents []Agent, llm LLMClient, judge ConsensusJudge, tenthMan TenthManActivator, minRounds, maxRounds int) *Engine {
	return &Engine{
		topic:    topic,
		agents:   agents,
		llm:      llm,
		judge:    judge,
		tenthMan: tenthMan,
		transcript: &Transcript{
			Topic: topic,
			Phase: FreeDebate,
		},
		minRounds: minRounds,
		maxRounds: maxRounds,
	}
}

// SetTenthManModel sets the model ID to use for the Tenth Man agent.
func (e *Engine) SetTenthManModel(model string) {
	e.tenthManModel = model
}

// Run executes the full debate: Phase 1 (free debate) and optionally Phase 2 (tenth man).
func (e *Engine) Run(ctx context.Context) (*Result, error) {
	if e.OnPhase != nil {
		e.OnPhase(FreeDebate)
	}

	// Phase 1: Free Debate
	var consensus *ConsensusResult
	for round := 1; round <= e.maxRounds; round++ {
		if err := e.runRound(ctx, round); err != nil {
			return nil, err
		}
		if round >= e.minRounds {
			var err error
			consensus, err = e.judge.Evaluate(ctx, e.transcript)
			if err != nil {
				return nil, fmt.Errorf("debate: consensus evaluation: %w", err)
			}
			if consensus.Detected && consensus.Score >= 7 {
				break
			}
		}
	}

	// Phase 2: Tenth Man
	if consensus != nil && consensus.Detected && consensus.Score >= 7 {
		e.transcript.Phase = TenthManPhase
		if e.OnPhase != nil {
			e.OnPhase(TenthManPhase)
		}

		model := e.tenthManModel
		if model == "" {
			model = e.agents[0].Model
		}
		tmAgent := e.tenthMan.BuildAgent(consensus.Position, len(e.agents)+1, model)
		e.agents = append(e.agents, tmAgent)
		e.consensusPosition = consensus.Position

		startRound := e.transcript.Rounds + 1
		for round := startRound; round < startRound+tenthManRounds; round++ {
			if err := e.runRound(ctx, round); err != nil {
				return nil, err
			}
		}
		var err error
		consensus, err = e.judge.Evaluate(ctx, e.transcript)
		if err != nil {
			return nil, fmt.Errorf("debate: final consensus evaluation: %w", err)
		}
	}

	return &Result{
		Transcript: e.transcript,
		Consensus:  consensus,
	}, nil
}

func (e *Engine) runRound(ctx context.Context, round int) error {
	for _, agent := range e.agents {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("debate: %w", err)
		}
		msgs := buildMessages(agent, e.topic, e.transcript, e.tenthMan, e.consensusPosition)
		resp, err := e.llm.ChatCompletion(ctx, agent.Model, msgs)
		if err != nil {
			return fmt.Errorf("debate: agent %s: %w", agent.Name, err)
		}
		content := ""
		if len(resp.Choices) > 0 {
			content = resp.Choices[0].Message.Content
		}
		turn := Turn{
			Round:   round,
			Agent:   agent,
			Content: content,
		}
		e.transcript.Turns = append(e.transcript.Turns, turn)
		if e.OnTurn != nil {
			e.OnTurn(turn)
		}
	}
	e.transcript.Rounds = round
	return nil
}
