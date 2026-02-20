package consensus

import (
	"context"
	"errors"
	"testing"

	"github.com/lorenzotomasdiez/tenth-man-rule/internal/debate"
	"github.com/lorenzotomasdiez/tenth-man-rule/internal/openrouter"
)

type mockLLM struct {
	response *openrouter.ChatResponse
	err      error
}

func (m *mockLLM) ChatCompletion(_ context.Context, _ string, _ []openrouter.Message) (*openrouter.ChatResponse, error) {
	return m.response, m.err
}

func chatResponse(content string) *openrouter.ChatResponse {
	return &openrouter.ChatResponse{
		Choices: []openrouter.Choice{{Message: openrouter.Message{Role: "assistant", Content: content}}},
	}
}

func sampleTranscript() *debate.Transcript {
	return &debate.Transcript{
		Topic: "test topic",
		Turns: []debate.Turn{
			{Round: 1, Agent: debate.Agent{ID: 1, Name: "Alice"}, Content: "I agree"},
			{Round: 1, Agent: debate.Agent{ID: 2, Name: "Bob"}, Content: "I also agree"},
		},
	}
}

func TestJudgeDetectsConsensus(t *testing.T) {
	llm := &mockLLM{response: chatResponse(`{"consensus_detected": true, "consensus_position": "everyone agrees", "agreement_score": 8, "dissenting_agents": []}`)}
	judge := NewJudge(llm, "test-model")

	result, err := judge.Evaluate(context.Background(), sampleTranscript())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Detected {
		t.Error("expected consensus detected")
	}
	if result.Score != 8 {
		t.Errorf("expected score 8, got %d", result.Score)
	}
	if result.Position != "everyone agrees" {
		t.Errorf("expected position 'everyone agrees', got %q", result.Position)
	}
}

func TestJudgeNoConsensus(t *testing.T) {
	llm := &mockLLM{response: chatResponse(`{"consensus_detected": false, "consensus_position": "", "agreement_score": 3, "dissenting_agents": ["Alice"]}`)}
	judge := NewJudge(llm, "test-model")

	result, err := judge.Evaluate(context.Background(), sampleTranscript())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Detected {
		t.Error("expected no consensus")
	}
	if result.Score != 3 {
		t.Errorf("expected score 3, got %d", result.Score)
	}
	if len(result.Dissenters) != 1 || result.Dissenters[0] != "Alice" {
		t.Errorf("expected dissenters [Alice], got %v", result.Dissenters)
	}
}

func TestJudgeMalformedJSON(t *testing.T) {
	llm := &mockLLM{response: chatResponse("this is not json at all")}
	judge := NewJudge(llm, "test-model")

	result, err := judge.Evaluate(context.Background(), sampleTranscript())
	if err != nil {
		t.Fatalf("expected no error on malformed JSON, got: %v", err)
	}
	if result.Detected {
		t.Error("expected Detected false on malformed JSON")
	}
	if result.Score != 0 {
		t.Errorf("expected score 0 on malformed JSON, got %d", result.Score)
	}
}

func TestJudgeExtractsJSONFromMarkdownCodeBlock(t *testing.T) {
	response := "Here is my analysis:\n```json\n{\"consensus_detected\": true, \"consensus_position\": \"we agree\", \"agreement_score\": 9, \"dissenting_agents\": []}\n```\nThat's my evaluation."
	llm := &mockLLM{response: chatResponse(response)}
	judge := NewJudge(llm, "test-model")

	result, err := judge.Evaluate(context.Background(), sampleTranscript())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Detected {
		t.Error("expected consensus detected")
	}
	if result.Score != 9 {
		t.Errorf("expected score 9, got %d", result.Score)
	}
}

func TestJudgeExtractsJSONFromCodeBlockNoLang(t *testing.T) {
	response := "```\n{\"consensus_detected\": false, \"consensus_position\": \"\", \"agreement_score\": 4, \"dissenting_agents\": [\"Bob\"]}\n```"
	llm := &mockLLM{response: chatResponse(response)}
	judge := NewJudge(llm, "test-model")

	result, err := judge.Evaluate(context.Background(), sampleTranscript())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Detected {
		t.Error("expected no consensus")
	}
	if result.Score != 4 {
		t.Errorf("expected score 4, got %d", result.Score)
	}
}

func TestJudgeExtractsJSONFromPreambleText(t *testing.T) {
	response := "Based on my analysis of the debate, here is the result:\n{\"consensus_detected\": true, \"consensus_position\": \"regulation needed\", \"agreement_score\": 7, \"dissenting_agents\": [\"Charlie\"]}\nEnd of evaluation."
	llm := &mockLLM{response: chatResponse(response)}
	judge := NewJudge(llm, "test-model")

	result, err := judge.Evaluate(context.Background(), sampleTranscript())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Detected {
		t.Error("expected consensus detected")
	}
	if result.Score != 7 {
		t.Errorf("expected score 7, got %d", result.Score)
	}
	if result.Position != "regulation needed" {
		t.Errorf("expected position 'regulation needed', got %q", result.Position)
	}
}

func TestJudgeRetriesOnMalformedJSON(t *testing.T) {
	callCount := 0
	llm := &retryMockLLM{
		responses: []*openrouter.ChatResponse{
			chatResponse("I can't produce valid JSON sorry"),
			chatResponse("Still not valid {broken"),
			chatResponse(`{"consensus_detected": true, "consensus_position": "finally works", "agreement_score": 8, "dissenting_agents": []}`),
		},
		callCount: &callCount,
	}
	judge := NewJudge(llm, "test-model")

	result, err := judge.Evaluate(context.Background(), sampleTranscript())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Detected {
		t.Error("expected consensus detected after retry")
	}
	if result.Score != 8 {
		t.Errorf("expected score 8, got %d", result.Score)
	}
	if callCount != 3 {
		t.Errorf("expected 3 LLM calls (2 retries), got %d", callCount)
	}
}

func TestJudgeRetriesExhaustedReturnsDefault(t *testing.T) {
	callCount := 0
	llm := &retryMockLLM{
		responses: []*openrouter.ChatResponse{
			chatResponse("not json 1"),
			chatResponse("not json 2"),
			chatResponse("not json 3"),
		},
		callCount: &callCount,
	}
	judge := NewJudge(llm, "test-model")

	result, err := judge.Evaluate(context.Background(), sampleTranscript())
	if err != nil {
		t.Fatalf("expected no error after retries exhausted, got: %v", err)
	}
	if result.Detected {
		t.Error("expected Detected false when all retries fail")
	}
	if result.Score != 0 {
		t.Errorf("expected score 0, got %d", result.Score)
	}
	if callCount != 3 {
		t.Errorf("expected 3 LLM calls, got %d", callCount)
	}
}

type retryMockLLM struct {
	responses []*openrouter.ChatResponse
	callCount *int
}

func (m *retryMockLLM) ChatCompletion(_ context.Context, _ string, _ []openrouter.Message) (*openrouter.ChatResponse, error) {
	idx := *m.callCount
	*m.callCount++
	if idx < len(m.responses) {
		return m.responses[idx], nil
	}
	return m.responses[len(m.responses)-1], nil
}

func TestJudgeLLMError(t *testing.T) {
	llm := &mockLLM{err: errors.New("api down")}
	judge := NewJudge(llm, "test-model")

	_, err := judge.Evaluate(context.Background(), sampleTranscript())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, llm.err) {
		t.Errorf("expected wrapped original error, got: %v", err)
	}
	expected := "consensus:"
	if len(err.Error()) < len(expected) || err.Error()[:len(expected)] != expected {
		t.Errorf("expected error prefix %q, got: %v", expected, err)
	}
}
