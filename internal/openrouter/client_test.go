package openrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestChatCompletion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method and path
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("expected /chat/completions, got %s", r.URL.Path)
		}

		// Verify headers
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Authorization header 'Bearer test-key', got %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type 'application/json', got %q", r.Header.Get("Content-Type"))
		}

		// Verify request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		var req ChatRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to unmarshal request: %v", err)
		}
		if req.Model != "test-model" {
			t.Errorf("expected model 'test-model', got %q", req.Model)
		}
		if len(req.Messages) != 1 || req.Messages[0].Content != "hello" {
			t.Errorf("unexpected messages: %+v", req.Messages)
		}

		// Return response
		resp := ChatResponse{
			Choices: []Choice{
				{Message: Message{Role: "assistant", Content: "hi there"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClientWithBaseURL("test-key", server.URL)
	resp, err := client.ChatCompletion(context.Background(), "test-model", []Message{
		{Role: "user", Content: "hello"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(resp.Choices))
	}
	if resp.Choices[0].Message.Content != "hi there" {
		t.Errorf("expected 'hi there', got %q", resp.Choices[0].Message.Content)
	}
}

func TestListModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/models" {
			t.Errorf("expected /models, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Authorization header 'Bearer test-key', got %q", r.Header.Get("Authorization"))
		}

		resp := ModelsResponse{
			Data: []Model{
				{ID: "model-1", Name: "Model One", Pricing: &Pricing{Prompt: "0", Completion: "0"}},
				{ID: "model-2", Name: "Model Two", Pricing: nil},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClientWithBaseURL("test-key", server.URL)
	models, err := client.ListModels(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0].ID != "model-1" {
		t.Errorf("expected 'model-1', got %q", models[0].ID)
	}
	if models[1].Pricing != nil {
		t.Errorf("expected nil pricing for model-2, got %+v", models[1].Pricing)
	}
}

func TestChatCompletionErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	client := NewClientWithBaseURL("test-key", server.URL)
	client.backoffFunc = noDelay
	_, err := client.ChatCompletion(context.Background(), "test-model", []Message{
		{Role: "user", Content: "hello"},
	})
	if err == nil {
		t.Fatal("expected error for 500 status, got nil")
	}
}

func TestListModelsErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("unauthorized"))
	}))
	defer server.Close()

	client := NewClientWithBaseURL("test-key", server.URL)
	_, err := client.ListModels(context.Background())
	if err == nil {
		t.Fatal("expected error for 401 status, got nil")
	}
}

func TestNewClientSetsDefaultBaseURL(t *testing.T) {
	client := NewClient("my-key")
	if client.baseURL != "https://openrouter.ai/api/v1" {
		t.Errorf("expected default base URL, got %q", client.baseURL)
	}
	if client.apiKey != "my-key" {
		t.Errorf("expected apiKey 'my-key', got %q", client.apiKey)
	}
}

func noDelay(attempt int) time.Duration { return 0 }

func successResponse() ChatResponse {
	return ChatResponse{
		Choices: []Choice{
			{Message: Message{Role: "assistant", Content: "ok"}},
		},
	}
}

func TestChatCompletionRetries429(t *testing.T) {
	var count atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := count.Add(1)
		if n <= 2 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, "rate limited")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(successResponse())
	}))
	defer server.Close()

	client := NewClientWithBaseURL("test-key", server.URL)
	client.backoffFunc = noDelay

	resp, err := client.ChatCompletion(context.Background(), "test-model", []Message{
		{Role: "user", Content: "hello"},
	})
	if err != nil {
		t.Fatalf("expected success after retries, got error: %v", err)
	}
	if resp.Choices[0].Message.Content != "ok" {
		t.Errorf("expected 'ok', got %q", resp.Choices[0].Message.Content)
	}
	if got := count.Load(); got != 3 {
		t.Errorf("expected 3 total requests, got %d", got)
	}
}

func TestChatCompletionRetries500(t *testing.T) {
	var count atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := count.Add(1)
		if n <= 1 {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "server error")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(successResponse())
	}))
	defer server.Close()

	client := NewClientWithBaseURL("test-key", server.URL)
	client.backoffFunc = noDelay

	resp, err := client.ChatCompletion(context.Background(), "test-model", []Message{
		{Role: "user", Content: "hello"},
	})
	if err != nil {
		t.Fatalf("expected success after retry, got error: %v", err)
	}
	if resp.Choices[0].Message.Content != "ok" {
		t.Errorf("expected 'ok', got %q", resp.Choices[0].Message.Content)
	}
	if got := count.Load(); got != 2 {
		t.Errorf("expected 2 total requests, got %d", got)
	}
}

func TestChatCompletionMaxRetries(t *testing.T) {
	var count atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, "rate limited")
	}))
	defer server.Close()

	client := NewClientWithBaseURL("test-key", server.URL)
	client.backoffFunc = noDelay

	_, err := client.ChatCompletion(context.Background(), "test-model", []Message{
		{Role: "user", Content: "hello"},
	})
	if err == nil {
		t.Fatal("expected error after max retries, got nil")
	}
	if got := count.Load(); got != 4 {
		t.Errorf("expected 4 total attempts (1 + 3 retries), got %d", got)
	}
}

func TestChatCompletionNoRetryOn400(t *testing.T) {
	var count atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "bad request")
	}))
	defer server.Close()

	client := NewClientWithBaseURL("test-key", server.URL)
	client.backoffFunc = noDelay

	_, err := client.ChatCompletion(context.Background(), "test-model", []Message{
		{Role: "user", Content: "hello"},
	})
	if err == nil {
		t.Fatal("expected error for 400, got nil")
	}
	if got := count.Load(); got != 1 {
		t.Errorf("expected 1 request (no retry), got %d", got)
	}
}
