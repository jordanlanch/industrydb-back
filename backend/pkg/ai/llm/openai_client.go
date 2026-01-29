package llm

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sashabaranov/go-openai"
)

// OpenAIClient wraps the OpenAI API client
type OpenAIClient struct {
	client      *openai.Client
	model       string
	temperature float32
	maxTokens   int
	logger      *log.Logger
}

// Config for OpenAI client
type Config struct {
	APIKey      string
	Model       string  // default: gpt-4-turbo-preview
	Temperature float32 // default: 0.7
	MaxTokens   int     // default: 2000
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(cfg Config, logger *log.Logger) *OpenAIClient {
	if cfg.Model == "" {
		cfg.Model = "gpt-4-turbo-preview"
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

	client := openai.NewClient(cfg.APIKey)

	return &OpenAIClient{
		client:      client,
		model:       cfg.Model,
		temperature: cfg.Temperature,
		maxTokens:   cfg.MaxTokens,
		logger:      logger,
	}
}

// ChatMessage represents a chat message
type ChatMessage struct {
	Role    string `json:"role"`    // system, user, assistant
	Content string `json:"content"`
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Messages    []ChatMessage `json:"messages"`
	Temperature float32       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	Message      string `json:"message"`
	TokensUsed   int    `json:"tokens_used"`
	FinishReason string `json:"finish_reason"`
}

// Chat sends a chat completion request to OpenAI
func (c *OpenAIClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	c.logger.Printf("🤖 OpenAI Chat: %d messages, model: %s", len(req.Messages), c.model)

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
		c.logger.Printf("❌ OpenAI Chat failed: %v (duration: %v)", err, duration)
		return nil, fmt.Errorf("openai chat failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from openai")
	}

	c.logger.Printf("✅ OpenAI Chat completed: %d tokens (duration: %v)", resp.Usage.TotalTokens, duration)

	return &ChatResponse{
		Message:      resp.Choices[0].Message.Content,
		TokensUsed:   resp.Usage.TotalTokens,
		FinishReason: string(resp.Choices[0].FinishReason),
	}, nil
}

// Complete sends a simple completion request (helper for single prompts)
func (c *OpenAIClient) Complete(ctx context.Context, prompt string, systemPrompt ...string) (string, error) {
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
// Returns a channel that emits response chunks
func (c *OpenAIClient) StreamChat(ctx context.Context, req ChatRequest) (<-chan string, <-chan error) {
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
					// Stream finished normally
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

		c.logger.Printf("✅ OpenAI Stream completed")
	}()

	return responseChan, errorChan
}

// CountTokens estimates the number of tokens in a text
// This is a rough estimate, not exact
func (c *OpenAIClient) CountTokens(text string) int {
	// Rough estimate: ~4 characters per token
	return len(text) / 4
}
