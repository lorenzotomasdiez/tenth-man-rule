package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	APIKey     string
	OutputDir  string
	AgentCount int
	MinRounds  int
	MaxRounds  int
}

func Load() (*Config, error) {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("config: OPENROUTER_API_KEY is required")
	}

	outputDir := os.Getenv("TENTHMAN_OUTPUT_DIR")
	if outputDir == "" {
		outputDir = "output"
	}

	agentCount, err := envInt("TENTHMAN_AGENTS", 9)
	if err != nil {
		return nil, err
	}

	minRounds, err := envInt("TENTHMAN_MIN_ROUNDS", 5)
	if err != nil {
		return nil, err
	}

	maxRounds, err := envInt("TENTHMAN_MAX_ROUNDS", 15)
	if err != nil {
		return nil, err
	}

	if agentCount < 3 {
		return nil, fmt.Errorf("config: AgentCount must be >= 3, got %d", agentCount)
	}
	if minRounds < 1 {
		return nil, fmt.Errorf("config: MinRounds must be >= 1, got %d", minRounds)
	}
	if maxRounds < minRounds {
		return nil, fmt.Errorf("config: MaxRounds (%d) must be >= MinRounds (%d)", maxRounds, minRounds)
	}

	return &Config{
		APIKey:     apiKey,
		OutputDir:  outputDir,
		AgentCount: agentCount,
		MinRounds:  minRounds,
		MaxRounds:  maxRounds,
	}, nil
}

func envInt(key string, defaultVal int) (int, error) {
	s := os.Getenv(key)
	if s == "" {
		return defaultVal, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("config: invalid %s value %q: %w", key, s, err)
	}
	return v, nil
}
