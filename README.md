# tenth-man-rule

A Go CLI that orchestrates multi-agent debates using free LLM models via the [OpenRouter](https://openrouter.ai) API.

Named after the intelligence doctrine originating from the failures documented in *The Pentagon Papers* and later institutionalized by Israeli military intelligence after the Yom Kippur War: **if 9 people agree, the 10th is obligated to argue the contrary position** -- not as token opposition, but with genuine analytical rigor.

## How It Works

```
Phase 1: Free Debate                    Phase 2: Tenth Man
┌─────────────────────────┐            ┌─────────────────────────┐
│  9 agents debate the    │            │  10th agent activated   │
│  topic (min 5 rounds)   │──consensus──▶  with contrarian       │
│                         │  detected  │  mandate (3 rounds)    │
│  Consensus judge eval-  │            │                         │
│  uates after each round │            │  Original 9 must        │
│  (score >= 7 triggers)  │            │  engage with counter-   │
└─────────────────────────┘            │  arguments              │
                                       └─────────────────────────┘
```

Each agent uses a different free model from OpenRouter. The consensus judge uses structured JSON output to detect agreement. When consensus is found, the Tenth Man builds the strongest possible case against it.

## Quick Start

```bash
# Clone and build
git clone https://github.com/lorenzotomasdiez/tenth-man-rule.git
cd tenth-man-rule
go build -o tenthman ./cmd/tenthman

# Set your OpenRouter API key (free, no credit card needed)
export OPENROUTER_API_KEY=sk-or-v1-your-key-here

# Run a debate
./tenthman debate --topic "Should AI be regulated?"
```

Get a free API key at [openrouter.ai/keys](https://openrouter.ai/keys).

## Usage

```bash
# Basic debate
./tenthman debate --topic "Is remote work better for software teams?"

# Custom agent count and rounds
./tenthman debate --topic "Should we colonize Mars?" --agents 5 --min-rounds 3 --max-rounds 8

# Custom output folder name
./tenthman debate --topic "Nuclear energy policy" --name "nuclear-debate"

# Pass API key as flag
./tenthman debate --topic "topic" --api-key sk-or-v1-...
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--topic` | (required) | The debate topic |
| `--agents` | `9` | Number of debate agents (min 3) |
| `--min-rounds` | `5` | Minimum rounds before consensus check |
| `--max-rounds` | `15` | Maximum debate rounds |
| `--output-dir` | `output` | Base directory for results |
| `--name` | auto-slug | Override output folder name |
| `--api-key` | `$OPENROUTER_API_KEY` | OpenRouter API key |

### Modes

| Command | Status | Description |
|---------|--------|-------------|
| `debate` | Available | Multi-agent structured debate with Tenth Man |
| `research` | Coming soon | Deep investigation with contrarian stress-testing |
| `analyze` | Coming soon | Document/decision counter-analysis |

## Output

Each run creates a timestamped folder with three files:

```
output/should-ai-be-regulated-20260220-143052/
  transcript.json   # Structured JSON: rounds, agents, positions, consensus scores
  report.md         # Human-readable markdown report
  debate.log        # Raw debug log
```

## Architecture

```
cmd/tenthman/              CLI entrypoint (Cobra)
internal/
  config/                  Configuration (env vars, defaults, validation)
  openrouter/              OpenRouter API client (retry, rate-limit)
  models/                  Free model registry and selection
  debate/                  Debate engine (phases, rounds, transcript)
    consensus/             LLM consensus detection (JSON extraction, retry)
    tenthman/              Tenth Man agent and contrarian prompts
  output/                  Terminal, markdown, JSON, and log writers
```

## The Debate Flow

**Phase 1 -- Free Debate** (minimum 5 rounds):
- N agents debate sequentially, each speaking once per round
- After the minimum round threshold, a consensus judge evaluates the transcript
- The judge returns `{ consensus_detected, consensus_position, agreement_score, dissenting_agents }`
- If `agreement_score >= 7`, Phase 2 activates

**Phase 2 -- Tenth Man** (3 rounds):
- A new agent is introduced with an explicit contrarian mandate
- The Tenth Man must build the strongest possible case *against* the consensus
- The original agents must directly engage with the Tenth Man's arguments
- Final consensus is re-evaluated

## Development

```bash
make build          # Build binary
make test           # Run all tests with race detector
make test-verbose   # Verbose test output
make lint           # Run golangci-lint
make fmt            # Format code
make clean          # Remove binary and output
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Requirements

- Go 1.25+
- An [OpenRouter API key](https://openrouter.ai/keys) (free tier works -- the tool uses exclusively free models)

## License

[MIT](LICENSE)
