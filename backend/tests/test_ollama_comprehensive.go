package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/jordanlanch/industrydb/pkg/ai/llm"
)

type Lead struct {
	Name         string  `json:"name"`
	Industry     string  `json:"industry"`
	Country      string  `json:"country"`
	City         string  `json:"city"`
	Email        string  `json:"email"`
	Phone        string  `json:"phone"`
	Website      string  `json:"website"`
	QualityScore int     `json:"quality_score"`
}

func main() {
	fmt.Println("🧪 COMPREHENSIVE OLLAMA/MCP TESTING")
	fmt.Println("====================================")
	fmt.Println()

	// Initialize Ollama
	client := llm.NewOllamaClient(llm.OllamaConfig{
		BaseURL:     "http://localhost:11434/v1",
		Model:       "llama3.1:8b",
		Temperature: 0.7,
		MaxTokens:   500,
	}, log.Default())

	fmt.Println("✅ Ollama initialized")
	fmt.Println()

	// Load all JSON files
	dataDir := "../data/test_all_industries"
	industries, err := loadLeadsFromJSON(dataDir)
	if err != nil {
		log.Fatalf("❌ Failed to load leads: %v", err)
	}

	fmt.Printf("📊 Loaded %d industries\n", len(industries))
	fmt.Println()

	// Test each industry
	totalStart := time.Now()
	results := make([]TestResult, 0)

	for i, industry := range industries {
		fmt.Printf("[%d/%d] Testing: %s (%d leads, %s)\n",
			i+1, len(industries), industry.Name, industry.LeadCount, industry.Country)

		result := testIndustryAnalysis(client, industry)
		results = append(results, result)

		if result.Success {
			fmt.Printf("  ✅ %v | Quality: %s\n", result.Duration, result.Quality)
			fmt.Printf("  💡 Insights: %d | Tokens: %d\n", result.InsightCount, result.TokensUsed)
		} else {
			fmt.Printf("  ❌ Error: %s\n", result.Error)
		}
		fmt.Println()

		// Delay between tests
		time.Sleep(2 * time.Second)
	}

	totalDuration := time.Since(totalStart)

	// Print summary
	printSummary(results, totalDuration)
}

type IndustryData struct {
	Name      string
	Country   string
	Leads     []Lead
	LeadCount int
}

type TestResult struct {
	Industry     string
	Country      string
	LeadCount    int
	Duration     time.Duration
	Success      bool
	Quality      string
	InsightCount int
	TokensUsed   int
	Error        string
}

func loadLeadsFromJSON(dataDir string) ([]IndustryData, error) {
	industries := make([]IndustryData, 0)

	// Walk through all JSON files
	err := filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(path) == ".json" {
			// Read file
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			// Parse JSON
			var leads []Lead
			if err := json.Unmarshal(data, &leads); err != nil {
				return err
			}

			if len(leads) == 0 {
				return nil
			}

			// Extract industry and country from path
			// Format: dataDir/industry/COUNTRY.json
			industry := filepath.Base(filepath.Dir(path))
			country := filepath.Base(path)
			country = country[:len(country)-5] // Remove .json

			industries = append(industries, IndustryData{
				Name:      industry,
				Country:   country,
				Leads:     leads,
				LeadCount: len(leads),
			})
		}

		return nil
	})

	return industries, err
}

func testIndustryAnalysis(client *llm.OllamaClient, industry IndustryData) TestResult {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// Generate data summary
	summary := generateDataSummary(industry.Leads)

	// Create analysis prompt
	prompt := fmt.Sprintf(`Analyze this business data:

Industry: %s
Country: %s
Total Leads: %d

%s

Provide:
1. Three key insights (concise bullet points)
2. One strategic recommendation

Be brief and actionable.`, industry.Name, industry.Country, industry.LeadCount, summary)

	systemPrompt := "You are a business data analyst. Provide clear, actionable insights in bullet format."

	start := time.Now()
	response, err := client.Complete(ctx, prompt, systemPrompt)
	duration := time.Since(start)

	if err != nil {
		return TestResult{
			Industry:  industry.Name,
			Country:   industry.Country,
			LeadCount: industry.LeadCount,
			Duration:  duration,
			Success:   false,
			Error:     err.Error(),
		}
	}

	// Analyze response quality
	quality := analyzeResponseQuality(response)
	insightCount := countInsights(response)
	tokensUsed := client.CountTokens(response)

	return TestResult{
		Industry:     industry.Name,
		Country:      industry.Country,
		LeadCount:    industry.LeadCount,
		Duration:     duration,
		Success:      true,
		Quality:      quality,
		InsightCount: insightCount,
		TokensUsed:   tokensUsed,
	}
}

func generateDataSummary(leads []Lead) string {
	withEmail := 0
	withPhone := 0
	withWebsite := 0
	totalQuality := 0

	for _, lead := range leads {
		if lead.Email != "" {
			withEmail++
		}
		if lead.Phone != "" {
			withPhone++
		}
		if lead.Website != "" {
			withWebsite++
		}
		totalQuality += lead.QualityScore
	}

	avgQuality := 0
	if len(leads) > 0 {
		avgQuality = totalQuality / len(leads)
	}

	return fmt.Sprintf(`Data Coverage:
- Email: %d%% (%d leads)
- Phone: %d%% (%d leads)
- Website: %d%% (%d leads)
- Avg Quality Score: %d/100`,
		(withEmail*100)/len(leads), withEmail,
		(withPhone*100)/len(leads), withPhone,
		(withWebsite*100)/len(leads), withWebsite,
		avgQuality)
}

func analyzeResponseQuality(response string) string {
	length := len(response)
	if length < 100 {
		return "Poor"
	} else if length < 300 {
		return "Fair"
	} else if length < 600 {
		return "Good"
	}
	return "Excellent"
}

func countInsights(response string) int {
	// Simple counting of bullet points or numbered items
	count := 0
	lines := []rune(response)
	for i, r := range lines {
		if r == '-' || r == '•' || r == '*' {
			if i == 0 || lines[i-1] == '\n' {
				count++
			}
		}
	}
	return count
}

func printSummary(results []TestResult, totalDuration time.Duration) {
	fmt.Println()
	fmt.Println("============================================================")
	fmt.Println("📊 TEST SUMMARY")
	fmt.Println("============================================================")
	fmt.Println()

	successCount := 0
	failCount := 0
	totalLeads := 0
	totalInsights := 0
	totalTokens := 0
	var totalTestDuration time.Duration

	for _, result := range results {
		if result.Success {
			successCount++
			totalLeads += result.LeadCount
			totalInsights += result.InsightCount
			totalTokens += result.TokensUsed
			totalTestDuration += result.Duration
		} else {
			failCount++
		}
	}

	fmt.Printf("✅ Successful Tests: %d\n", successCount)
	fmt.Printf("❌ Failed Tests: %d\n", failCount)
	fmt.Printf("📈 Total Leads Analyzed: %d\n", totalLeads)
	fmt.Printf("💡 Total Insights Generated: %d\n", totalInsights)
	fmt.Printf("🔢 Total Tokens Used: %d\n", totalTokens)
	fmt.Printf("⏱️  Average Response Time: %v\n", totalTestDuration/time.Duration(successCount))
	fmt.Printf("⏱️  Total Test Duration: %v\n", totalDuration)
	fmt.Println()

	// Quality distribution
	qualityMap := make(map[string]int)
	for _, result := range results {
		if result.Success {
			qualityMap[result.Quality]++
		}
	}

	fmt.Println("📊 RESPONSE QUALITY DISTRIBUTION:")
	for quality, count := range qualityMap {
		fmt.Printf("  %s: %d tests\n", quality, count)
	}
	fmt.Println()

	// Failed tests
	if failCount > 0 {
		fmt.Println("❌ FAILED TESTS:")
		for _, result := range results {
			if !result.Success {
				fmt.Printf("  - %s (%s): %s\n", result.Industry, result.Country, result.Error)
			}
		}
		fmt.Println()
	}

	fmt.Println("🎉 Ollama/MCP testing complete!")
	fmt.Println()
	fmt.Println("💰 Cost: $0 (100% free & local)")
	fmt.Println("🔒 Privacy: 100% (data never left server)")
	fmt.Println("🦙 Model: Llama 3.1 8B (Open Source)")
}
