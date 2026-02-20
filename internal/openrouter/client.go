package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const maxRetries = 3

// Client is an OpenRouter API client.
type Client struct {
	httpClient  *http.Client
	apiKey      string
	baseURL     string
	backoffFunc func(attempt int) time.Duration
}

func defaultBackoff(attempt int) time.Duration {
	return time.Duration(1<<uint(attempt)) * time.Second
}

// NewClient creates a new Client with the default OpenRouter base URL.
func NewClient(apiKey string) *Client {
	return &Client{
		httpClient:  &http.Client{},
		apiKey:      apiKey,
		baseURL:     "https://openrouter.ai/api/v1",
		backoffFunc: defaultBackoff,
	}
}

// NewClientWithBaseURL creates a new Client with a custom base URL (for testing).
func NewClientWithBaseURL(apiKey, baseURL string) *Client {
	return &Client{
		httpClient:  &http.Client{},
		apiKey:      apiKey,
		baseURL:     baseURL,
		backoffFunc: defaultBackoff,
	}
}

// ChatCompletion sends a chat completion request with retry for transient failures.
func (c *Client) ChatCompletion(ctx context.Context, model string, messages []Message) (*ChatResponse, error) {
	reqBody := ChatRequest{
		Model:    model,
		Messages: messages,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("openrouter: %w", err)
	}

	resp, err := c.doWithRetry(ctx, func(ctx context.Context) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")
		return c.httpClient.Do(req)
	})
	if err != nil {
		return nil, fmt.Errorf("openrouter: %w", err)
	}
	defer resp.Body.Close()

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("openrouter: %w", err)
	}
	return &chatResp, nil
}

func isRetryable(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode >= 500
}

func (c *Client) doWithRetry(ctx context.Context, do func(context.Context) (*http.Response, error)) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := c.backoffFunc(attempt - 1)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		resp, err := do(ctx)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusOK {
			return resp, nil
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if !isRetryable(resp.StatusCode) {
			return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
		}

		// Respect Retry-After header on 429 (additional wait on top of backoff)
		if resp.StatusCode == http.StatusTooManyRequests {
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if secs, parseErr := strconv.Atoi(ra); parseErr == nil {
					raDelay := time.Duration(secs) * time.Second
					// Skip if backoffFunc signals zero delays (test mode)
					if raDelay > 0 && c.backoffFunc(0) > 0 {
						select {
						case <-ctx.Done():
							return nil, ctx.Err()
						case <-time.After(raDelay):
						}
					}
				}
			}
		}

		lastErr = fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil, lastErr
}

// ListModels retrieves available models from OpenRouter.
func (c *Client) ListModels(ctx context.Context) ([]Model, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("openrouter: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openrouter: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openrouter: unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	var modelsResp ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("openrouter: %w", err)
	}
	return modelsResp.Data, nil
}
