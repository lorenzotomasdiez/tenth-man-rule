package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/lorenzotomasdiez/tenth-man-rule/internal/debate"
	"github.com/lorenzotomasdiez/tenth-man-rule/internal/debate/consensus"
	"github.com/lorenzotomasdiez/tenth-man-rule/internal/debate/tenthman"
	"github.com/lorenzotomasdiez/tenth-man-rule/internal/models"
	"github.com/lorenzotomasdiez/tenth-man-rule/internal/openrouter"
	"github.com/lorenzotomasdiez/tenth-man-rule/internal/output"
)

func TestE2EFullDebateWithMockServer(t *testing.T) {
	var requestCount atomic.Int32

	// Mock OpenRouter server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := requestCount.Add(1)

		var req openrouter.ChatRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Verify auth header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-key-123" {
			t.Errorf("bad auth header: %s", auth)
		}

		// Generate contextual responses
		systemPrompt := ""
		if len(req.Messages) > 0 {
			systemPrompt = req.Messages[0].Content
		}

		var content string
		switch {
		case strings.Contains(systemPrompt, "consensus judge"):
			// Consensus judge call
			if count > 20 { // After enough rounds
				content = `{"consensus_detected": true, "consensus_position": "Space exploration is crucial", "agreement_score": 8, "dissenting_agents": []}`
			} else {
				content = `{"consensus_detected": false, "consensus_position": "", "agreement_score": 3, "dissenting_agents": ["Agent-1"]}`
			}
		case strings.Contains(systemPrompt, "OBLIGATED"):
			content = "As the Tenth Man, I must point out that space exploration diverts resources from pressing terrestrial problems."
		case strings.Contains(systemPrompt, "engage with the Tenth Man"):
			content = "While the Tenth Man raises resource concerns, the technological spillovers from space programs more than justify the investment."
		default:
			content = "Space exploration drives innovation and ensures humanity's long-term survival. We should invest heavily."
		}

		resp := openrouter.ChatResponse{
			Choices: []openrouter.Choice{{Message: openrouter.Message{Role: "assistant", Content: content}}},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Build the full pipeline with real components
	client := openrouter.NewClientWithBaseURL("test-key-123", server.URL)

	registry := models.NewRegistry(models.DefaultFreeModels())
	selected := registry.SelectModels(4) // 3 agents + 1 tenth man

	agents := []debate.Agent{
		{ID: 1, Name: "Ava", Model: selected[0].ID, Role: "debater"},
		{ID: 2, Name: "Ben", Model: selected[1].ID, Role: "debater"},
		{ID: 3, Name: "Cal", Model: selected[2].ID, Role: "debater"},
	}

	judge := consensus.NewJudge(client, selected[0].ID)
	tm := tenthman.NewActivator()

	// Setup output
	outDir := t.TempDir()
	slug := output.GenerateSlug("Space exploration investment")
	dir, err := output.CreateOutputDir(outDir, slug)
	if err != nil {
		t.Fatalf("CreateOutputDir: %v", err)
	}

	writer := output.NewWriter(dir)

	// Run engine with real components
	engine := debate.NewEngine("Should we invest more in space exploration?", agents, client, judge, tm, 5, 10)
	engine.SetTenthManModel(selected[3].ID)
	engine.OnTurn = func(turn debate.Turn) {
		output.PrintTurn(turn)
		writer.Log(turn.Content)
	}
	engine.OnPhase = func(phase debate.Phase) {
		output.PrintPhase(phase)
		writer.Log("phase transition")
	}

	result, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("debate failed: %v", err)
	}

	// Write outputs
	if err := writer.WriteJSON(result.Transcript); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	consensusResult := result.Consensus
	if consensusResult == nil {
		consensusResult = &debate.ConsensusResult{}
	}
	if err := writer.WriteMarkdown(result.Transcript, consensusResult); err != nil {
		t.Fatalf("WriteMarkdown: %v", err)
	}
	if err := writer.WriteLog(); err != nil {
		t.Fatalf("WriteLog: %v", err)
	}

	// Verify outputs exist
	for _, name := range []string{"transcript.json", "report.md", "debate.log"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("missing output file %s: %v", name, err)
		}
	}

	// Verify transcript structure
	if result.Transcript.Rounds < 5 {
		t.Errorf("expected at least 5 rounds, got %d", result.Transcript.Rounds)
	}
	if len(result.Transcript.Turns) == 0 {
		t.Error("no turns recorded")
	}

	// Verify JSON is valid
	jsonData, _ := os.ReadFile(filepath.Join(dir, "transcript.json"))
	var parsed debate.Transcript
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if parsed.Topic != "Should we invest more in space exploration?" {
		t.Errorf("wrong topic in JSON: %s", parsed.Topic)
	}

	// Verify markdown has content
	mdData, _ := os.ReadFile(filepath.Join(dir, "report.md"))
	md := string(mdData)
	if !strings.Contains(md, "space exploration") {
		t.Error("markdown missing topic content")
	}

	// Verify log has entries
	logData, _ := os.ReadFile(filepath.Join(dir, "debate.log"))
	if len(logData) == 0 {
		t.Error("debate.log is empty")
	}

	t.Logf("E2E complete: %d rounds, %d turns, %d API calls",
		result.Transcript.Rounds, len(result.Transcript.Turns), requestCount.Load())
}
