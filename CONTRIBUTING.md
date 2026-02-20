# Contributing

Thanks for your interest in contributing to tenth-man-rule.

## Development

```bash
git clone https://github.com/lorenzotomasdiez/tenth-man-rule.git
cd tenth-man-rule
cp .env.example .env  # add your OpenRouter API key
make test             # run tests
make build            # build binary
```

## Workflow

1. Fork the repo and create a branch from `main`
2. Write a failing test for the change you want to make
3. Write the minimal code to make the test pass
4. Refactor if needed
5. Ensure `make test` passes
6. Submit a pull request

## Guidelines

- Follow TDD: write the failing test first, then the implementation
- Keep code minimal -- no over-engineering
- All non-CLI packages go in `internal/`
- Wrap errors with context: `fmt.Errorf("package: %w", err)`
- Thread `context.Context` through all API calls
- No global state -- use dependency injection

## Testing

```bash
make test          # all tests with race detector
make test-verbose  # verbose output
```

No real API calls in tests. Use `httptest.NewServer` for HTTP mocks and interfaces for dependency injection.

## Reporting Issues

Open an issue on GitHub with:
- What you expected to happen
- What actually happened
- Steps to reproduce
