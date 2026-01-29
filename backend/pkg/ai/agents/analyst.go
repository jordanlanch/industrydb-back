package agents

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/lead"
	"github.com/jordanlanch/industrydb/pkg/ai/llm"
)

// AnalystAgent provides data analysis and insights
type AnalystAgent struct {
	llm    llm.LLMClient
	db     *ent.Client
	logger *log.Logger
}

// NewAnalystAgent creates a new analyst agent
func NewAnalystAgent(llmClient llm.LLMClient, db *ent.Client, logger *log.Logger) *AnalystAgent {
	if logger == nil {
		logger = log.Default()
	}

	return &AnalystAgent{
		llm:    llmClient,
		db:     db,
		logger: logger,
	}
}

// AnalysisRequest represents a request for data analysis
type AnalysisRequest struct {
	Industry string   `json:"industry,omitempty"`
	Country  string   `json:"country,omitempty"`
	Question string   `json:"question,omitempty"`
	Filters  []string `json:"filters,omitempty"`
}

// AnalysisResponse represents the analysis result
type AnalysisResponse struct {
	Summary         string   `json:"summary"`
	KeyInsights     []string `json:"key_insights"`
	Recommendations []string `json:"recommendations"`
	Metrics         map[string]interface{} `json:"metrics"`
	RawAnalysis     string   `json:"raw_analysis,omitempty"`
}

// Analyze performs data analysis and returns insights
func (a *AnalystAgent) Analyze(ctx context.Context, req AnalysisRequest) (*AnalysisResponse, error) {
	a.logger.Printf("🔍 Analyst: Analyzing data (industry: %s, country: %s)", req.Industry, req.Country)

	// 1. Gather data from database
	metrics, err := a.gatherMetrics(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to gather metrics: %w", err)
	}

	// 2. Generate data summary for LLM
	dataSummary := a.generateDataSummary(metrics, req)

	// 3. Create prompt
	prompt := llm.AnalyzeLeadsPrompt(
		a.getIndustryName(req.Industry),
		req.Country,
		metrics["total_leads"].(int),
		dataSummary,
	)

	// Add user question if provided
	if req.Question != "" {
		prompt += fmt.Sprintf("\n\nUser Question: %s", req.Question)
	}

	// 4. Get analysis from LLM
	analysis, err := a.llm.Complete(ctx, prompt, llm.AnalystSystemPrompt)
	if err != nil {
		return nil, fmt.Errorf("llm analysis failed: %w", err)
	}

	// 5. Parse and structure the response
	response := a.parseAnalysis(analysis, metrics)

	a.logger.Printf("✅ Analyst: Analysis completed (%d insights, %d recommendations)",
		len(response.KeyInsights), len(response.Recommendations))

	return response, nil
}

// gatherMetrics collects relevant metrics from the database
func (a *AnalystAgent) gatherMetrics(ctx context.Context, req AnalysisRequest) (map[string]interface{}, error) {
	metrics := make(map[string]interface{})

	// Build query based on filters
	query := a.db.Lead.Query()

	if req.Industry != "" {
		query = query.Where(lead.IndustryEQ(lead.Industry(req.Industry)))
	}

	if req.Country != "" {
		query = query.Where(lead.CountryEQ(req.Country))
	}

	// Total leads
	totalLeads, err := query.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count leads: %w", err)
	}
	metrics["total_leads"] = totalLeads

	// Get all leads for more detailed analysis
	leads, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch leads: %w", err)
	}

	// Calculate additional metrics
	withEmail := 0
	withPhone := 0
	withWebsite := 0
	avgQualityScore := 0.0

	for _, l := range leads {
		if l.Email != "" {
			withEmail++
		}
		if l.Phone != "" {
			withPhone++
		}
		if l.Website != "" {
			withWebsite++
		}
		avgQualityScore += float64(l.QualityScore)
	}

	if totalLeads > 0 {
		avgQualityScore /= float64(totalLeads)
		metrics["email_coverage"] = float64(withEmail) / float64(totalLeads) * 100
		metrics["phone_coverage"] = float64(withPhone) / float64(totalLeads) * 100
		metrics["website_coverage"] = float64(withWebsite) / float64(totalLeads) * 100
	}

	metrics["avg_quality_score"] = avgQualityScore
	metrics["with_email"] = withEmail
	metrics["with_phone"] = withPhone
	metrics["with_website"] = withWebsite

	// Group by city (top 5)
	cityMap := make(map[string]int)
	for _, l := range leads {
		cityMap[l.City]++
	}
	metrics["top_cities"] = a.getTopN(cityMap, 5)

	// Group by industry if not filtered
	if req.Industry == "" {
		industryMap := make(map[string]int)
		for _, l := range leads {
			industryMap[string(l.Industry)]++
		}
		metrics["top_industries"] = a.getTopN(industryMap, 5)
	}

	return metrics, nil
}

// generateDataSummary creates a human-readable summary of the metrics
func (a *AnalystAgent) generateDataSummary(metrics map[string]interface{}, req AnalysisRequest) string {
	var summary strings.Builder

	summary.WriteString(fmt.Sprintf("Total Leads: %d\n", metrics["total_leads"].(int)))
	summary.WriteString(fmt.Sprintf("Average Quality Score: %.1f/100\n", metrics["avg_quality_score"].(float64)))
	summary.WriteString(fmt.Sprintf("\nContact Information Coverage:\n"))
	summary.WriteString(fmt.Sprintf("- Email: %.1f%% (%d leads)\n", metrics["email_coverage"].(float64), metrics["with_email"].(int)))
	summary.WriteString(fmt.Sprintf("- Phone: %.1f%% (%d leads)\n", metrics["phone_coverage"].(float64), metrics["with_phone"].(int)))
	summary.WriteString(fmt.Sprintf("- Website: %.1f%% (%d leads)\n", metrics["website_coverage"].(float64), metrics["with_website"].(int)))

	if topCities, ok := metrics["top_cities"].([]string); ok && len(topCities) > 0 {
		summary.WriteString(fmt.Sprintf("\nTop Cities: %s\n", strings.Join(topCities, ", ")))
	}

	if topIndustries, ok := metrics["top_industries"].([]string); ok && len(topIndustries) > 0 {
		summary.WriteString(fmt.Sprintf("\nTop Industries: %s\n", strings.Join(topIndustries, ", ")))
	}

	return summary.String()
}

// getTopN returns top N items from a map sorted by value
func (a *AnalystAgent) getTopN(m map[string]int, n int) []string {
	type kv struct {
		Key   string
		Value int
	}

	var sorted []kv
	for k, v := range m {
		sorted = append(sorted, kv{k, v})
	}

	// Simple bubble sort (fine for small n)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].Value < sorted[j].Value {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	result := make([]string, 0, n)
	for i := 0; i < n && i < len(sorted); i++ {
		result = append(result, sorted[i].Key)
	}

	return result
}

// parseAnalysis parses the LLM response into structured format
func (a *AnalystAgent) parseAnalysis(analysis string, metrics map[string]interface{}) *AnalysisResponse {
	// Parse sections from the analysis
	sections := strings.Split(analysis, "\n\n")

	response := &AnalysisResponse{
		RawAnalysis:     analysis,
		KeyInsights:     []string{},
		Recommendations: []string{},
		Metrics:         metrics,
	}

	// Extract insights and recommendations using simple parsing
	lines := strings.Split(analysis, "\n")
	inInsights := false
	inRecommendations := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Detect section headers
		lower := strings.ToLower(line)
		if strings.Contains(lower, "key") && (strings.Contains(lower, "finding") || strings.Contains(lower, "insight") || strings.Contains(lower, "trend")) {
			inInsights = true
			inRecommendations = false
			continue
		}
		if strings.Contains(lower, "recommendation") || strings.Contains(lower, "suggest") {
			inInsights = false
			inRecommendations = true
			continue
		}

		// Extract bullet points
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") || strings.HasPrefix(line, "• ") {
			item := strings.TrimLeft(line, "-*• ")
			item = strings.TrimSpace(item)

			if inInsights && item != "" {
				response.KeyInsights = append(response.KeyInsights, item)
			} else if inRecommendations && item != "" {
				response.Recommendations = append(response.Recommendations, item)
			}
		}
	}

	// Generate summary (first paragraph or section)
	if len(sections) > 0 {
		response.Summary = strings.TrimSpace(sections[0])
	}

	// Ensure we have at least some content
	if len(response.KeyInsights) == 0 {
		response.KeyInsights = []string{"Analysis completed - see raw analysis for details"}
	}

	if len(response.Recommendations) == 0 {
		response.Recommendations = []string{"Continue monitoring data quality and lead engagement"}
	}

	return response
}

// getIndustryName converts industry code to display name
func (a *AnalystAgent) getIndustryName(code string) string {
	names := map[string]string{
		"tattoo":     "Tattoo Studios",
		"beauty":     "Beauty Salons",
		"barber":     "Barber Shops",
		"gym":        "Gyms & Fitness Centers",
		"restaurant": "Restaurants",
		"cafe":       "Cafes & Coffee Shops",
		"spa":        "Spas & Wellness Centers",
		"massage":    "Massage Therapy",
	}

	if name, ok := names[code]; ok {
		return name
	}
	return code
}
