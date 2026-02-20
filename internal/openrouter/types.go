package openrouter

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents a request to the chat completions endpoint.
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// ChatResponse represents a response from the chat completions endpoint.
type ChatResponse struct {
	Choices []Choice `json:"choices"`
}

// Choice represents a single completion choice.
type Choice struct {
	Message Message `json:"message"`
}

// Model represents an OpenRouter model.
type Model struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Pricing *Pricing `json:"pricing"`
}

// Pricing represents model pricing information.
type Pricing struct {
	Prompt     string `json:"prompt"`
	Completion string `json:"completion"`
}

// ModelsResponse represents the response from the models endpoint.
type ModelsResponse struct {
	Data []Model `json:"data"`
}
