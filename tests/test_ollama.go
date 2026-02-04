package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jordanlanch/industrydb/pkg/ai/llm"
)

func main() {
	// Create Ollama client
	client := llm.NewOllamaClient(llm.OllamaConfig{
		BaseURL:     "http://localhost:11434/v1",
		Model:       "llama3.1:8b",
		Temperature: 0.7,
		MaxTokens:   500,
	}, log.Default())

	// Test simple completion
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("🦙 Testing Ollama with Llama 3.1 8B...")
	fmt.Println()

	prompt := `Analyze this business data:
- Industry: Tattoo Studios
- Country: United States
- Total Leads: 7
- Email Coverage: 45%
- Phone Coverage: 32%
- Average Quality Score: 72/100

Provide 3 key insights and 2 recommendations. Be concise.`

	systemPrompt := "You are a business data analyst. Provide clear, actionable insights."

	start := time.Now()
	response, err := client.Complete(ctx, prompt, systemPrompt)
	duration := time.Since(start)

	if err != nil {
		log.Fatalf("❌ Error: %v", err)
	}

	fmt.Println("✅ Response received!")
	fmt.Printf("⏱️  Duration: %v\n", duration)
	fmt.Println()
	fmt.Println("📊 Analysis:")
	fmt.Println(response)
	fmt.Println()
	fmt.Println("🎉 Ollama test successful!")
}
