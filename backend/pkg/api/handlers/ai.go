package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/jordanlanch/industrydb/pkg/ai/agents"
	"github.com/labstack/echo/v4"
)

// AIHandler handles AI-powered endpoints
type AIHandler struct {
	analyst *agents.AnalystAgent
}

// NewAIHandler creates a new AI handler
func NewAIHandler(analyst *agents.AnalystAgent) *AIHandler {
	return &AIHandler{
		analyst: analyst,
	}
}

// AnalyzeData analyzes lead data and provides insights
// POST /api/v1/ai/analyze
func (h *AIHandler) AnalyzeData(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 60*time.Second)
	defer cancel()

	// Parse request
	var req agents.AnalysisRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request body",
		})
	}

	// Validate
	if req.Industry == "" && req.Country == "" && req.Question == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "At least one of industry, country, or question must be provided",
		})
	}

	// Execute analysis
	result, err := h.analyst.Analyze(ctx, req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Analysis failed",
			"details": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, result)
}

// GetInsights gets quick insights for dashboard
// GET /api/v1/ai/insights
func (h *AIHandler) GetInsights(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
	defer cancel()

	// Get query parameters
	industry := c.QueryParam("industry")
	country := c.QueryParam("country")

	// Execute analysis with a simple question
	result, err := h.analyst.Analyze(ctx, agents.AnalysisRequest{
		Industry: industry,
		Country:  country,
		Question: "What are the key trends and opportunities in this data?",
	})

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to generate insights",
		})
	}

	// Return simplified response for dashboard
	return c.JSON(http.StatusOK, map[string]interface{}{
		"summary":         result.Summary,
		"key_insights":    result.KeyInsights,
		"recommendations": result.Recommendations,
		"metrics":         result.Metrics,
	})
}

// AnalyzeIndustryTrends analyzes trends for a specific industry
// GET /api/v1/ai/trends/:industry
func (h *AIHandler) AnalyzeIndustryTrends(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 60*time.Second)
	defer cancel()

	industry := c.Param("industry")
	if industry == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Industry parameter is required",
		})
	}

	// Execute analysis
	result, err := h.analyst.Analyze(ctx, agents.AnalysisRequest{
		Industry: industry,
		Question: "Analyze the trends, geographic distribution, and quality of leads in this industry. What opportunities exist?",
	})

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Trend analysis failed",
		})
	}

	return c.JSON(http.StatusOK, result)
}

// CompareIndustries compares two or more industries
// POST /api/v1/ai/compare
func (h *AIHandler) CompareIndustries(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 90*time.Second)
	defer cancel()

	// Parse request
	var req struct {
		Industries []string `json:"industries" validate:"required,min=2"`
		Country    string   `json:"country,omitempty"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request body",
		})
	}

	if len(req.Industries) < 2 {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "At least 2 industries are required for comparison",
		})
	}

	// Analyze each industry
	results := make(map[string]*agents.AnalysisResponse)
	for _, industry := range req.Industries {
		result, err := h.analyst.Analyze(ctx, agents.AnalysisRequest{
			Industry: industry,
			Country:  req.Country,
			Question: "Provide key metrics and characteristics of this industry segment",
		})

		if err != nil {
			// Continue with other industries even if one fails
			continue
		}

		results[industry] = result
	}

	if len(results) == 0 {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to analyze any of the provided industries",
		})
	}

	// Return comparison results
	return c.JSON(http.StatusOK, map[string]interface{}{
		"industries": results,
		"count":      len(results),
	})
}
