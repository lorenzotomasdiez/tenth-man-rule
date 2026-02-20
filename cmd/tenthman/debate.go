package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/lorenzotomasdiez/tenth-man-rule/internal/debate"
	"github.com/lorenzotomasdiez/tenth-man-rule/internal/debate/consensus"
	"github.com/lorenzotomasdiez/tenth-man-rule/internal/debate/tenthman"
	"github.com/lorenzotomasdiez/tenth-man-rule/internal/models"
	"github.com/lorenzotomasdiez/tenth-man-rule/internal/openrouter"
	"github.com/lorenzotomasdiez/tenth-man-rule/internal/output"
	"github.com/spf13/cobra"
)

func newDebateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debate",
		Short: "Run a multi-agent debate on a topic",
		RunE:  runDebate,
	}
	cmd.Flags().String("topic", "", "Debate topic (required)")
	cmd.Flags().String("name", "", "Override output folder name (default: auto-slug from topic)")
	cmd.MarkFlagRequired("topic")
	return cmd
}

func runDebate(cmd *cobra.Command, args []string) error {
	topic, _ := cmd.Flags().GetString("topic")
	name, _ := cmd.Flags().GetString("name")
	apiKey, _ := cmd.Root().PersistentFlags().GetString("api-key")
	outputDir, _ := cmd.Root().PersistentFlags().GetString("output-dir")
	agentCount, _ := cmd.Root().PersistentFlags().GetInt("agents")
	minRounds, _ := cmd.Root().PersistentFlags().GetInt("min-rounds")
	maxRounds, _ := cmd.Root().PersistentFlags().GetInt("max-rounds")

	if apiKey == "" {
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	}
	if apiKey == "" {
		return fmt.Errorf("API key required: set --api-key flag or OPENROUTER_API_KEY env var")
	}
	if agentCount < 3 {
		return fmt.Errorf("agent count must be >= 3, got %d", agentCount)
	}

	// Setup context with Ctrl+C cancellation
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Create OpenRouter client
	client := openrouter.NewClient(apiKey)

	// Fetch live models, fallback to defaults
	allModels, err := client.ListModels(ctx)
	if err != nil {
		fmt.Printf("Warning: could not fetch models: %v. Using defaults.\n", err)
		allModels = models.DefaultFreeModels()
	}
	registry := models.NewRegistry(allModels)
	if len(registry.FreeModels()) == 0 {
		registry = models.NewRegistry(models.DefaultFreeModels())
	}
	selected := registry.SelectModels(agentCount + 1)

	// Build agents
	agents := make([]debate.Agent, agentCount)
	names := []string{"Alice", "Bob", "Carol", "Dave", "Eve", "Frank", "Grace", "Heidi", "Ivan"}
	for i := range agentCount {
		agentName := fmt.Sprintf("Agent-%d", i+1)
		if i < len(names) {
			agentName = names[i]
		}
		agents[i] = debate.Agent{
			ID:    i + 1,
			Name:  agentName,
			Model: selected[i].ID,
			Role:  "debater",
		}
	}

	// Create judge and tenth man activator
	judgeModel := selected[0].ID
	judge := consensus.NewJudge(client, judgeModel)
	tm := tenthman.NewActivator()

	// Setup output directory
	slug := name
	if slug == "" {
		slug = output.GenerateSlug(topic)
	}
	outDir, err := output.CreateOutputDir(outputDir, slug)
	if err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Run debate
	fmt.Printf("Debate: %s\n", topic)
	fmt.Printf("Agents: %d | Rounds: %d-%d | Output: %s\n\n", agentCount, minRounds, maxRounds, outDir)

	writer := output.NewWriter(outDir)

	engine := debate.NewEngine(topic, agents, client, judge, tm, minRounds, maxRounds)
	engine.SetTenthManModel(selected[agentCount].ID)
	engine.OnTurn = func(turn debate.Turn) {
		output.PrintTurn(turn)
		writer.Log(fmt.Sprintf("[Round %d] %s (%s): %s", turn.Round, turn.Agent.Name, turn.Agent.Model, turn.Content))
	}
	engine.OnPhase = func(phase debate.Phase) {
		output.PrintPhase(phase)
		writer.Log(fmt.Sprintf("Phase transition: %d", phase))
	}

	result, err := engine.Run(ctx)
	if err != nil {
		return fmt.Errorf("debate: %w", err)
	}

	// Write outputs
	if err := writer.WriteJSON(result.Transcript); err != nil {
		return fmt.Errorf("writing JSON: %w", err)
	}
	consensus := result.Consensus
	if consensus == nil {
		consensus = &debate.ConsensusResult{}
	}
	if err := writer.WriteMarkdown(result.Transcript, consensus); err != nil {
		return fmt.Errorf("writing markdown: %w", err)
	}

	if err := writer.WriteLog(); err != nil {
		return fmt.Errorf("writing log: %w", err)
	}

	output.PrintConsensus(consensus)
	fmt.Printf("\nDebate complete. Output saved to: %s\n", outDir)
	return nil
}
