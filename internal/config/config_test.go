package config

import (
	"os"
	"path/filepath"
	"testing"
)

func clearEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"OPENROUTER_API_KEY",
		"TENTHMAN_OUTPUT_DIR",
		"TENTHMAN_AGENTS",
		"TENTHMAN_MIN_ROUNDS",
		"TENTHMAN_MAX_ROUNDS",
	} {
		t.Setenv(key, "")
		os.Unsetenv(key)
	}
}

func TestLoad_MissingAPIKey(t *testing.T) {
	clearEnv(t)
	_, err := Load()
	if err == nil {
		t.Fatal("expected error when OPENROUTER_API_KEY is missing")
	}
}

func TestLoad_Defaults(t *testing.T) {
	clearEnv(t)
	t.Setenv("OPENROUTER_API_KEY", "test-key")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIKey != "test-key" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "test-key")
	}
	if cfg.OutputDir != "output" {
		t.Errorf("OutputDir = %q, want %q", cfg.OutputDir, "output")
	}
	if cfg.AgentCount != 9 {
		t.Errorf("AgentCount = %d, want %d", cfg.AgentCount, 9)
	}
	if cfg.MinRounds != 5 {
		t.Errorf("MinRounds = %d, want %d", cfg.MinRounds, 5)
	}
	if cfg.MaxRounds != 15 {
		t.Errorf("MaxRounds = %d, want %d", cfg.MaxRounds, 15)
	}
}

func TestLoad_CustomEnvVars(t *testing.T) {
	clearEnv(t)
	t.Setenv("OPENROUTER_API_KEY", "my-key")
	t.Setenv("TENTHMAN_OUTPUT_DIR", "results")
	t.Setenv("TENTHMAN_AGENTS", "5")
	t.Setenv("TENTHMAN_MIN_ROUNDS", "3")
	t.Setenv("TENTHMAN_MAX_ROUNDS", "10")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIKey != "my-key" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "my-key")
	}
	if cfg.OutputDir != "results" {
		t.Errorf("OutputDir = %q, want %q", cfg.OutputDir, "results")
	}
	if cfg.AgentCount != 5 {
		t.Errorf("AgentCount = %d, want %d", cfg.AgentCount, 5)
	}
	if cfg.MinRounds != 3 {
		t.Errorf("MinRounds = %d, want %d", cfg.MinRounds, 3)
	}
	if cfg.MaxRounds != 10 {
		t.Errorf("MaxRounds = %d, want %d", cfg.MaxRounds, 10)
	}
}

func TestLoad_AgentCountTooLow(t *testing.T) {
	clearEnv(t)
	t.Setenv("OPENROUTER_API_KEY", "test-key")
	t.Setenv("TENTHMAN_AGENTS", "2")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when AgentCount < 3")
	}
}

func TestLoad_MinRoundsTooLow(t *testing.T) {
	clearEnv(t)
	t.Setenv("OPENROUTER_API_KEY", "test-key")
	t.Setenv("TENTHMAN_MIN_ROUNDS", "0")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when MinRounds < 1")
	}
}

func TestLoad_MaxRoundsLessThanMinRounds(t *testing.T) {
	clearEnv(t)
	t.Setenv("OPENROUTER_API_KEY", "test-key")
	t.Setenv("TENTHMAN_MIN_ROUNDS", "5")
	t.Setenv("TENTHMAN_MAX_ROUNDS", "3")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when MaxRounds < MinRounds")
	}
}

func TestLoad_InvalidAgentCount(t *testing.T) {
	clearEnv(t)
	t.Setenv("OPENROUTER_API_KEY", "test-key")
	t.Setenv("TENTHMAN_AGENTS", "notanumber")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for non-numeric TENTHMAN_AGENTS")
	}
}

func TestLoadDotEnv_SetsVarsFromFile(t *testing.T) {
	clearEnv(t)
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	os.WriteFile(envFile, []byte("OPENROUTER_API_KEY=from-dotenv\nTENTHMAN_OUTPUT_DIR=dotenv-output\n"), 0644)

	err := LoadDotEnv(envFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIKey != "from-dotenv" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "from-dotenv")
	}
	if cfg.OutputDir != "dotenv-output" {
		t.Errorf("OutputDir = %q, want %q", cfg.OutputDir, "dotenv-output")
	}
}

func TestLoadDotEnv_EnvVarsTakePrecedence(t *testing.T) {
	clearEnv(t)
	t.Setenv("OPENROUTER_API_KEY", "from-env")

	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	os.WriteFile(envFile, []byte("OPENROUTER_API_KEY=from-dotenv\n"), 0644)

	err := LoadDotEnv(envFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIKey != "from-env" {
		t.Errorf("APIKey = %q, want %q (env var should take precedence)", cfg.APIKey, "from-env")
	}
}

func TestLoadDotEnv_MissingFileIsNotError(t *testing.T) {
	err := LoadDotEnv("/nonexistent/.env")
	if err != nil {
		t.Fatalf("missing .env file should not be an error, got: %v", err)
	}
}
