package llm

import "context"

// LLMClient is the interface for LLM clients (OpenAI, Ollama, etc.)
type LLMClient interface {
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	Complete(ctx context.Context, prompt string, systemPrompt ...string) (string, error)
	StreamChat(ctx context.Context, req ChatRequest) (<-chan string, <-chan error)
	CountTokens(text string) int
}

// Ensure implementations satisfy the interface
var _ LLMClient = (*OpenAIClient)(nil)
var _ LLMClient = (*OllamaClient)(nil)
