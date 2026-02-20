package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "tenthman",
		Short: "Multi-agent debate orchestrator using the Tenth Man Rule",
		Long:  "Orchestrates multi-agent debates, research, and analysis using free LLM models via OpenRouter. If 9 people agree, the 10th is obligated to argue the contrary position.",
	}

	root.PersistentFlags().String("api-key", "", "OpenRouter API key (overrides OPENROUTER_API_KEY env var)")
	root.PersistentFlags().String("output-dir", "output", "Output directory for results")
	root.PersistentFlags().Int("agents", 9, "Number of debate agents (minimum 3)")
	root.PersistentFlags().Int("min-rounds", 5, "Minimum debate rounds before consensus check")
	root.PersistentFlags().Int("max-rounds", 15, "Maximum debate rounds")

	root.AddCommand(newDebateCmd())
	root.AddCommand(newResearchCmd())
	root.AddCommand(newAnalyzeCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
