package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jordanlanch/industrydb/config"
	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/lead"
	"github.com/jordanlanch/industrydb/pkg/ai/agents"
	"github.com/jordanlanch/industrydb/pkg/ai/llm"
	"github.com/jordanlanch/industrydb/pkg/database"
)

func main() {
	fmt.Println("🧪 COMPREHENSIVE AI TESTING - ALL INDUSTRIES")
	fmt.Println("============================================================")
	fmt.Println()

	// Load config
	cfg := config.Load()

	// Connect to database
	db, err := database.NewClient(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("❌ Database connection failed: %v", err)
	}
	defer db.Close()

	fmt.Println("✅ Database connected")

	// Initialize Ollama client
	llmClient := llm.NewOllamaClient(llm.OllamaConfig{
		BaseURL:     "http://localhost:11434/v1",
		Model:       "llama3.1:8b",
		Temperature: 0.7,
		MaxTokens:   500,
	}, log.Default())

	fmt.Println("✅ Ollama client initialized")
	fmt.Println()

	// Create analyst agent
	analyst := agents.NewAnalystAgent(llmClient, db.Ent, log.Default())

	// Get all industries from database
	industries, err := getIndustries(db.Ent)
	if err != nil {
		log.Fatalf("❌ Failed to get industries: %v", err)
	}

	fmt.Printf("📊 Found %d industries in database\n", len(industries))
	fmt.Println()

	// Test each industry
	results := make(map[string]TestResult)
	totalStart := time.Now()

	for i, industry := range industries {
		fmt.Printf("[%d/%d] Testing industry: %s\n", i+1, len(industries), industry)

		result := testIndustry(analyst, industry)
		results[industry] = result

		fmt.Printf("  ✅ Duration: %v | Leads: %d | Insights: %d\n",
			result.Duration, result.LeadCount, len(result.Insights))
		fmt.Println()

		// Small delay to avoid overwhelming Ollama
		time.Sleep(2 * time.Second)
	}

	totalDuration := time.Since(totalStart)

	// Print summary
	printSummary(results, totalDuration)
}

type TestResult struct {
	Industry    string
	LeadCount   int
	Duration    time.Duration
	Success     bool
	Insights    []string
	Recommendations []string
	Error       string
}

func getIndustries(client *ent.Client) ([]string, error) {
	ctx := context.Background()

	// Get unique industries from leads
	leads, err := client.Lead.Query().
		Select(lead.FieldIndustry).
		All(ctx)

	if err != nil {
		return nil, err
	}

	// Deduplicate
	industryMap := make(map[string]bool)
	for _, l := range leads {
		industryMap[string(l.Industry)] = true
	}

	industries := make([]string, 0, len(industryMap))
	for industry := range industryMap {
		industries = append(industries, industry)
	}

	return industries, nil
}

func testIndustry(analyst *agents.AnalystAgent, industry string) TestResult {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	start := time.Now()

	// Run analysis
	response, err := analyst.Analyze(ctx, agents.AnalysisRequest{
		Industry: industry,
		Question: "Provide 3 key insights about this industry data. Be concise.",
	})

	duration := time.Since(start)

	if err != nil {
		return TestResult{
			Industry: industry,
			Duration: duration,
			Success:  false,
			Error:    err.Error(),
		}
	}

	leadCount := 0
	if total, ok := response.Metrics["total_leads"].(int); ok {
		leadCount = total
	}

	return TestResult{
		Industry:        industry,
		LeadCount:       leadCount,
		Duration:        duration,
		Success:         true,
		Insights:        response.KeyInsights,
		Recommendations: response.Recommendations,
	}
}

func printSummary(results map[string]TestResult, totalDuration time.Duration) {
	fmt.Println()
	fmt.Println("==========================================================)
	fmt.Println("📊 TEST SUMMARY")
	fmt.Println("==========================================================)
	fmt.Println()

	successCount := 0
	failCount := 0
	totalLeads := 0
	totalInsights := 0
	var totalTestDuration time.Duration

	for _, result := range results {
		if result.Success {
			successCount++
			totalLeads += result.LeadCount
			totalInsights += len(result.Insights)
			totalTestDuration += result.Duration
		} else {
			failCount++
		}
	}

	fmt.Printf("✅ Successful Tests: %d\n", successCount)
	fmt.Printf("❌ Failed Tests: %d\n", failCount)
	fmt.Printf("📈 Total Leads Analyzed: %d\n", totalLeads)
	fmt.Printf("💡 Total Insights Generated: %d\n", totalInsights)
	fmt.Printf("⏱️  Average Response Time: %v\n", totalTestDuration/time.Duration(successCount))
	fmt.Printf("⏱️  Total Test Duration: %v\n", totalDuration)
	fmt.Println()

	// Top performers
	fmt.Println("🏆 TOP INDUSTRIES BY LEAD COUNT:")
	type industryCount struct {
		name  string
		count int
	}
	var sorted []industryCount
	for _, result := range results {
		if result.Success {
			sorted = append(sorted, industryCount{result.Industry, result.LeadCount})
		}
	}
	// Simple bubble sort
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].count < sorted[j].count {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	for i := 0; i < 5 && i < len(sorted); i++ {
		fmt.Printf("  %d. %s: %d leads\n", i+1, sorted[i].name, sorted[i].count)
	}
	fmt.Println()

	// Failed tests
	if failCount > 0 {
		fmt.Println("❌ FAILED TESTS:")
		for _, result := range results {
			if !result.Success {
				fmt.Printf("  - %s: %s\n", result.Industry, result.Error)
			}
		}
		fmt.Println()
	}

	fmt.Println("🎉 Testing complete!")
}
