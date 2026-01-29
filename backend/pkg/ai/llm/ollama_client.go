package llm

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sashabaranov/go-openai"
)

// OllamaClient wraps Ollama API (OpenAI compatible)
type OllamaClient struct {
	client      *openai.Client
	model       string
	temperature float32
	maxTokens   int
	logger      *log.Logger
}

// OllamaConfig for Ollama client
type OllamaConfig struct {
	BaseURL     string  // default: http://localhost:11434/v1
	Model       string  // default: llama3.1:8b
	Temperature float32 // default: 0.7
	MaxTokens   int     // default: 2000
}

// NewOllamaClient creates a new Ollama client
func NewOllamaClient(cfg OllamaConfig, logger *log.Logger) *OllamaClient {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:11434/v1"
	}
	if cfg.Model == "" {
		cfg.Model = "llama3.1:8b"
	}
	if cfg.Temperature == 0 {
		cfg.Temperature = 0.7
	}
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = 2000
	}
	if logger == nil {
		logger = log.Default()
	}

	// Create OpenAI-compatible client pointing to Ollama
	config := openai.DefaultConfig("ollama") // API key not needed for Ollama
	config.BaseURL = cfg.BaseURL

	client := openai.NewClientWithConfig(config)

	logger.Printf("🦙 Ollama client initialized (model: %s, url: %s)", cfg.Model, cfg.BaseURL)

	return &OllamaClient{
		client:      client,
		model:       cfg.Model,
		temperature: cfg.Temperature,
		maxTokens:   cfg.MaxTokens,
		logger:      logger,
	}
}

// Chat sends a chat completion request to Ollama
func (c *OllamaClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	c.logger.Printf("🦙 Ollama Chat: %d messages, model: %s", len(req.Messages), c.model)

	// Convert messages
	messages := make([]openai.ChatCompletionMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Set defaults
	temperature := req.Temperature
	if temperature == 0 {
		temperature = c.temperature
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = c.maxTokens
	}

	// Create request
	chatReq := openai.ChatCompletionRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}

	// Execute request with timeout
	start := time.Now()
	resp, err := c.client.CreateChatCompletion(ctx, chatReq)
	duration := time.Since(start)

	if err != nil {
		c.logger.Printf("❌ Ollama Chat failed: %v (duration: %v)", err, duration)
		return nil, fmt.Errorf("ollama chat failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from ollama")
	}

	c.logger.Printf("✅ Ollama Chat completed: %d tokens (duration: %v)", resp.Usage.TotalTokens, duration)

	return &ChatResponse{
		Message:      resp.Choices[0].Message.Content,
		TokensUsed:   resp.Usage.TotalTokens,
		FinishReason: string(resp.Choices[0].FinishReason),
	}, nil
}

// Complete sends a simple completion request (helper for single prompts)
func (c *OllamaClient) Complete(ctx context.Context, prompt string, systemPrompt ...string) (string, error) {
	messages := []ChatMessage{}

	// Add system prompt if provided
	if len(systemPrompt) > 0 && systemPrompt[0] != "" {
		messages = append(messages, ChatMessage{
			Role:    "system",
			Content: systemPrompt[0],
		})
	}

	// Add user prompt
	messages = append(messages, ChatMessage{
		Role:    "user",
		Content: prompt,
	})

	resp, err := c.Chat(ctx, ChatRequest{
		Messages: messages,
	})

	if err != nil {
		return "", err
	}

	return resp.Message, nil
}

// StreamChat sends a streaming chat completion request
func (c *OllamaClient) StreamChat(ctx context.Context, req ChatRequest) (<-chan string, <-chan error) {
	responseChan := make(chan string, 100)
	errorChan := make(chan error, 1)

	go func() {
		defer close(responseChan)
		defer close(errorChan)

		// Convert messages
		messages := make([]openai.ChatCompletionMessage, len(req.Messages))
		for i, msg := range req.Messages {
			messages[i] = openai.ChatCompletionMessage{
				Role:    msg.Role,
				Content: msg.Content,
			}
		}

		// Create stream request
		stream, err := c.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
			Model:       c.model,
			Messages:    messages,
			Temperature: req.Temperature,
			MaxTokens:   req.MaxTokens,
			Stream:      true,
		})

		if err != nil {
			errorChan <- fmt.Errorf("failed to create stream: %w", err)
			return
		}
		defer stream.Close()

		// Read stream
		for {
			response, err := stream.Recv()
			if err != nil {
				if err.Error() == "EOF" {
					break
				}
				errorChan <- fmt.Errorf("stream error: %w", err)
				return
			}

			if len(response.Choices) > 0 {
				content := response.Choices[0].Delta.Content
				if content != "" {
					responseChan <- content
				}
			}
		}

		c.logger.Printf("✅ Ollama Stream completed")
	}()

	return responseChan, errorChan
}

// CountTokens estimates the number of tokens in a text
func (c *OllamaClient) CountTokens(text string) int {
	// Rough estimate: ~4 characters per token
	return len(text) / 4
}
