package output

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/lorenzotomasdiez/tenth-man-rule/internal/debate"
)

func TestGenerateSlug(t *testing.T) {
	got := GenerateSlug("AI and Machine Learning!")
	want := "ai-and-machine-learning"
	if got != want {
		t.Errorf("GenerateSlug() = %q, want %q", got, want)
	}
}

func TestGenerateSlugMaxLength(t *testing.T) {
	long := strings.Repeat("word ", 20) // 100 chars
	got := GenerateSlug(long)
	if len(got) > 50 {
		t.Errorf("GenerateSlug() length = %d, want <= 50", len(got))
	}
}

func TestCreateOutputDir(t *testing.T) {
	base := t.TempDir()
	slug := "test-topic"

	dir, err := CreateOutputDir(base, slug)
	if err != nil {
		t.Fatalf("CreateOutputDir() error = %v", err)
	}

	// Should contain slug
	if !strings.Contains(dir, slug) {
		t.Errorf("dir %q does not contain slug %q", dir, slug)
	}

	// Should match timestamp pattern
	pattern := regexp.MustCompile(`test-topic-\d{8}-\d{6}$`)
	if !pattern.MatchString(filepath.Base(dir)) {
		t.Errorf("dir base %q does not match expected pattern", filepath.Base(dir))
	}

	// Directory should exist
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("directory does not exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("path is not a directory")
	}
}

func TestWriteJSON(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(dir)

	transcript := &debate.Transcript{
		Topic: "Test Topic",
		Turns: []debate.Turn{
			{Round: 1, Agent: debate.Agent{ID: 1, Name: "Alice", Model: "model-a", Role: "debater"}, Content: "Hello"},
		},
		Phase:  debate.FreeDebate,
		Rounds: 1,
	}

	err := w.WriteJSON(transcript)
	if err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "transcript.json"))
	if err != nil {
		t.Fatalf("reading transcript.json: %v", err)
	}

	var got debate.Transcript
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if got.Topic != "Test Topic" {
		t.Errorf("Topic = %q, want %q", got.Topic, "Test Topic")
	}
	if len(got.Turns) != 1 {
		t.Errorf("Turns length = %d, want 1", len(got.Turns))
	}
}

func TestWriteMarkdown(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(dir)

	transcript := &debate.Transcript{
		Topic: "AI Regulation",
		Turns: []debate.Turn{
			{Round: 1, Agent: debate.Agent{ID: 1, Name: "Alice", Model: "model-a", Role: "debater"}, Content: "We need regulation"},
			{Round: 2, Agent: debate.Agent{ID: 2, Name: "Bob", Model: "model-b", Role: "debater"}, Content: "I agree"},
		},
		Phase:  debate.FreeDebate,
		Rounds: 2,
	}

	consensus := &debate.ConsensusResult{
		Detected:   true,
		Position:   "Regulation is needed",
		Score:      8,
		Dissenters: []string{"Charlie"},
	}

	err := w.WriteMarkdown(transcript, consensus)
	if err != nil {
		t.Fatalf("WriteMarkdown() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "report.md"))
	if err != nil {
		t.Fatalf("reading report.md: %v", err)
	}

	content := string(data)

	checks := []string{"AI Regulation", "Alice", "Bob", "Round 1", "Round 2"}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("report.md does not contain %q", check)
		}
	}
}

func TestWriteLog(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(dir)

	w.Log("round 1 started")
	w.Log("agent Alice responded: hello world")
	w.Log("consensus check: score 3")

	err := w.WriteLog()
	if err != nil {
		t.Fatalf("WriteLog() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "debate.log"))
	if err != nil {
		t.Fatalf("reading debate.log: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "round 1 started") {
		t.Error("debate.log missing log entry")
	}
	if !strings.Contains(content, "agent Alice responded") {
		t.Error("debate.log missing agent log entry")
	}
}

func TestLogWritesImmediatelyToFile(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(dir)

	w.Log("first entry")

	// File should exist and contain the entry immediately (before WriteLog)
	data, err := os.ReadFile(filepath.Join(dir, "debate.log"))
	if err != nil {
		t.Fatalf("debate.log should exist after Log(): %v", err)
	}
	if !strings.Contains(string(data), "first entry") {
		t.Error("debate.log should contain entry immediately after Log()")
	}
}

func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	return string(out)
}

func TestPrintPhaseContainsCyan(t *testing.T) {
	out := captureStdout(func() { PrintPhase(debate.FreeDebate) })
	if !strings.Contains(out, "\033[36m") {
		t.Error("PrintPhase(FreeDebate) should contain cyan ANSI code")
	}
}

func TestPrintPhaseContainsRed(t *testing.T) {
	out := captureStdout(func() { PrintPhase(debate.TenthManPhase) })
	if !strings.Contains(out, "\033[31m") {
		t.Error("PrintPhase(TenthManPhase) should contain red ANSI code")
	}
}

func TestPrintTurnContainsBoldAgentName(t *testing.T) {
	turn := debate.Turn{
		Round:   1,
		Agent:   debate.Agent{ID: 1, Name: "Alice"},
		Content: "test content",
	}
	out := captureStdout(func() { PrintTurn(turn) })
	if !strings.Contains(out, "\033[1mAlice") {
		t.Error("PrintTurn should bold the agent name")
	}
}

func TestPrintConsensusDetectedGreen(t *testing.T) {
	result := &debate.ConsensusResult{Detected: true, Position: "test", Score: 8}
	out := captureStdout(func() { PrintConsensus(result) })
	if !strings.Contains(out, "\033[32m") {
		t.Error("PrintConsensus with Detected=true should contain green ANSI code")
	}
}

func TestPrintConsensusNotDetectedRed(t *testing.T) {
	result := &debate.ConsensusResult{Detected: false, Position: "test", Score: 3}
	out := captureStdout(func() { PrintConsensus(result) })
	if !strings.Contains(out, "\033[31m") {
		t.Error("PrintConsensus with Detected=false should contain red ANSI code")
	}
}

func TestPrintTurnShowsFullContent(t *testing.T) {
	longContent := strings.Repeat("a", 500)
	turn := debate.Turn{
		Round:   1,
		Agent:   debate.Agent{ID: 1, Name: "Alice"},
		Content: longContent,
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintTurn(turn)

	w.Close()
	os.Stdout = old

	out, _ := io.ReadAll(r)
	output := string(out)

	if strings.Contains(output, "...") {
		t.Error("PrintTurn should not truncate content")
	}
	if !strings.Contains(output, longContent) {
		t.Error("PrintTurn should print full content")
	}
}
