package consensus

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/lorenzotomasdiez/tenth-man-rule/internal/debate"
	"github.com/lorenzotomasdiez/tenth-man-rule/internal/openrouter"
)

const maxJudgeRetries = 3

var codeBlockRe = regexp.MustCompile("(?s)```(?:json)?\\s*\\n?(.*?)\\n?```")

// Judge evaluates debate transcripts for consensus using an LLM.
type Judge struct {
	llm   debate.LLMClient
	model string
}

// NewJudge creates a new consensus Judge.
func NewJudge(llm debate.LLMClient, model string) *Judge {
	return &Judge{llm: llm, model: model}
}

// Evaluate implements debate.ConsensusJudge.
func (j *Judge) Evaluate(ctx context.Context, transcript *debate.Transcript) (*debate.ConsensusResult, error) {
	system := openrouter.Message{
		Role: "system",
		Content: `You are a consensus judge. Analyze the debate transcript and return ONLY valid JSON in this exact format:
{"consensus_detected": bool, "consensus_position": "...", "agreement_score": 1-10, "dissenting_agents": ["..."]}
Do NOT include any other text, explanation, or markdown formatting. Return ONLY the JSON object.`,
	}

	var sb strings.Builder
	for _, turn := range transcript.Turns {
		fmt.Fprintf(&sb, "%s: %s\n", turn.Agent.Name, turn.Content)
	}
	user := openrouter.Message{Role: "user", Content: sb.String()}

	for attempt := range maxJudgeRetries {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("consensus: %w", err)
		}

		msgs := []openrouter.Message{system, user}
		if attempt > 0 {
			msgs = append(msgs, openrouter.Message{
				Role:    "user",
				Content: "Your previous response was not valid JSON. Return ONLY a JSON object, no markdown, no explanation.",
			})
		}

		resp, err := j.llm.ChatCompletion(ctx, j.model, msgs)
		if err != nil {
			return nil, fmt.Errorf("consensus: %w", err)
		}

		raw := resp.Choices[0].Message.Content
		result, ok := parseConsensusJSON(raw)
		if ok {
			return result, nil
		}
	}

	return &debate.ConsensusResult{}, nil
}

// parseConsensusJSON tries to extract and parse a ConsensusResult from LLM output.
func parseConsensusJSON(raw string) (*debate.ConsensusResult, bool) {
	// Try direct parse first
	var result debate.ConsensusResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &result); err == nil {
		return &result, true
	}

	// Try extracting from markdown code block
	if matches := codeBlockRe.FindStringSubmatch(raw); len(matches) > 1 {
		if err := json.Unmarshal([]byte(strings.TrimSpace(matches[1])), &result); err == nil {
			return &result, true
		}
	}

	// Try finding JSON object in text (first { to last })
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		if err := json.Unmarshal([]byte(raw[start:end+1]), &result); err == nil {
			return &result, true
		}
	}

	return nil, false
}
