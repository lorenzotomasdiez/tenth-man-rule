package output

import (
	"fmt"

	"github.com/lorenzotomasdiez/tenth-man-rule/internal/debate"
)

const (
	ansiReset   = "\033[0m"
	ansiBold    = "\033[1m"
	ansiRed     = "\033[31m"
	ansiGreen   = "\033[32m"
	ansiYellow  = "\033[33m"
	AnsiMagenta = "\033[35m"
	ansiCyan    = "\033[36m"
)

// Colorize wraps s with an ANSI color code and reset.
func Colorize(color, s string) string { return color + s + ansiReset }

// Bold wraps s with ANSI bold and reset.
func Bold(s string) string { return ansiBold + s + ansiReset }

// PrintTurn prints a formatted turn to stdout.
func PrintTurn(turn debate.Turn) {
	fmt.Printf("%s %s: %s\n",
		Colorize(ansiYellow, fmt.Sprintf("[Round %d]", turn.Round)),
		Bold(turn.Agent.Name),
		turn.Content,
	)
}

// PrintPhase prints a phase transition banner.
func PrintPhase(phase debate.Phase) {
	name := "Free Debate"
	color := ansiCyan
	if phase == debate.TenthManPhase {
		name = "Tenth Man"
		color = ansiRed
	}
	fmt.Printf("\n%s\n\n", Colorize(ansiBold+color, "=== Phase: "+name+" ==="))
}

// PrintConsensus prints the consensus summary.
func PrintConsensus(result *debate.ConsensusResult) {
	detected := "No"
	detectedColor := ansiRed
	if result.Detected {
		detected = "Yes"
		detectedColor = ansiGreen
	}
	fmt.Printf("Consensus Detected: %s\n", Colorize(ansiBold+detectedColor, detected))
	fmt.Printf("Position: %s\n", result.Position)
	fmt.Printf("Agreement Score: %s\n", Colorize(ansiYellow, fmt.Sprintf("%d/10", result.Score)))
	if len(result.Dissenters) > 0 {
		fmt.Printf("Dissenters: %v\n", result.Dissenters)
	}
}
