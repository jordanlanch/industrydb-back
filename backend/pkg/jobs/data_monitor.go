package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/lead"
	"github.com/jordanlanch/industrydb/pkg/cache"
)

// IndustryCountryPair represents a combination that needs more data
type IndustryCountryPair struct {
	Industry string `json:"industry"`
	Country  string `json:"country"`
	Count    int    `json:"count"`
	Priority int    `json:"priority"` // Higher = more urgent
}

// DataMonitor manages data acquisition jobs
type DataMonitor struct {
	db     *ent.Client
	cache  *cache.Client
	logger *log.Logger
}

// NewDataMonitor creates a new data monitor instance
func NewDataMonitor(db *ent.Client, cache *cache.Client, logger *log.Logger) *DataMonitor {
	if logger == nil {
		logger = log.Default()
	}

	return &DataMonitor{
		db:     db,
		cache:  cache,
		logger: logger,
	}
}

// DetectLowDataIndustries finds industry/country combinations with < threshold leads
func (m *DataMonitor) DetectLowDataIndustries(ctx context.Context, threshold int) ([]IndustryCountryPair, error) {
	m.logger.Printf("Detecting industries with < %d leads...", threshold)

	// Get all leads
	leads, err := m.db.Lead.Query().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query leads: %w", err)
	}

	// Group by industry/country manually
	counts := make(map[string]IndustryCountryPair)
	for _, l := range leads {
		key := fmt.Sprintf("%s:%s", l.Industry, l.Country)
		pair, exists := counts[key]
		if !exists {
			pair = IndustryCountryPair{
				Industry: string(l.Industry),
				Country:  l.Country,
				Count:    0,
			}
		}
		pair.Count++
		counts[key] = pair
	}

	// Filter pairs with count < threshold
	var pairs []IndustryCountryPair
	for _, pair := range counts {
		if pair.Count < threshold {
			// Calculate priority (lower count = higher priority)
			pair.Priority = threshold - pair.Count
			pairs = append(pairs, pair)
		}
	}

	m.logger.Printf("Found %d industry/country pairs with < %d leads", len(pairs), threshold)

	return pairs, nil
}

// DetectMissingCombinations finds industry/country combinations with NO data
func (m *DataMonitor) DetectMissingCombinations(ctx context.Context) ([]IndustryCountryPair, error) {
	m.logger.Println("Detecting missing industry/country combinations...")

	// Get all possible combinations from config
	allIndustries := m.getAllIndustries()

	allCountries := []string{
		// Americas
		"US", "CA", "MX", "BR", "AR", "CL", "CO", "PE", "VE", "EC",
		// Europe
		"GB", "DE", "FR", "ES", "IT", "NL", "BE", "CH", "AT", "SE",
		"NO", "DK", "FI", "PL", "CZ", "HU", "RO", "PT", "GR", "IE",
		// Asia-Pacific
		"JP", "CN", "IN", "AU", "NZ", "SG", "MY", "TH", "VN", "PH",
		"ID", "KR", "TW", "HK",
		// Middle East
		"AE", "SA", "IL", "TR", "EG",
		// Africa
		"ZA", "NG", "KE", "MA",
		// Eastern Europe
		"RU", "UA", "BY",
		// Rest of Europe
		"BG", "HR", "SI", "SK", "LT", "LV", "EE",
	}

	// Get existing combinations
	leads, err := m.db.Lead.Query().
		Select(lead.FieldIndustry, lead.FieldCountry).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to query existing combinations: %w", err)
	}

	existingMap := make(map[string]bool)
	for _, l := range leads {
		key := fmt.Sprintf("%s:%s", string(l.Industry), l.Country)
		existingMap[key] = true
	}

	// Find missing combinations
	var missing []IndustryCountryPair
	for _, industry := range allIndustries {
		for _, country := range allCountries {
			key := fmt.Sprintf("%s:%s", industry, country)
			if !existingMap[key] {
				missing = append(missing, IndustryCountryPair{
					Industry: industry,
					Country:  country,
					Count:    0,
					Priority: 100, // High priority for completely missing data
				})
			}
		}
	}

	m.logger.Printf("Found %d missing combinations", len(missing))
	return missing, nil
}

// getAllIndustries returns list of all industry IDs
func (m *DataMonitor) getAllIndustries() []string {
	// TODO: Load from config file or database
	return []string{
		"tattoo", "beauty", "barber", "spa", "nail_salon",
		"gym", "dentist", "pharmacy", "massage",
		"restaurant", "cafe", "bar", "bakery",
		"car_repair", "car_wash", "car_dealer",
		"clothing", "convenience",
		"lawyer", "accountant",
	}
}

// TriggerDataFetch triggers Python script to fetch data for specific industry/country
func (m *DataMonitor) TriggerDataFetch(ctx context.Context, industry, country string, limit int) error {
	m.logger.Printf("Triggering data fetch for %s/%s (limit: %d)", industry, country, limit)

	// Check if fetch is already in progress
	inProgress, err := m.IsFetchInProgress(ctx, industry, country)
	if err != nil {
		m.logger.Printf("Warning: failed to check fetch status: %v", err)
	} else if inProgress {
		m.logger.Printf("Fetch already in progress for %s/%s, skipping", industry, country)
		return nil
	}

	// Mark as in progress
	if err := m.MarkFetchInProgress(ctx, industry, country); err != nil {
		m.logger.Printf("Warning: failed to mark fetch as in progress: %v", err)
	}

	// Get project root (assuming we're in backend/pkg/jobs/)
	projectRoot := filepath.Join("..", "..", "..")
	scriptPath := filepath.Join(projectRoot, "scripts", "data-acquisition", "simple_fetch.py")

	// Build command
	cmd := exec.CommandContext(ctx,
		"python3",
		scriptPath,
		industry,
		"--country", country,
		"--limit", fmt.Sprintf("%d", limit),
		"--output", filepath.Join(projectRoot, "data", "auto_fetch"),
	)

	// Execute in background
	go func() {
		startTime := time.Now()

		output, err := cmd.CombinedOutput()
		duration := time.Since(startTime)

		// Clear fetch status
		clearCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		m.ClearFetchStatus(clearCtx, industry, country)

		if err != nil {
			m.logger.Printf("❌ Data fetch failed for %s/%s: %v (duration: %v)",
				industry, country, err, duration)
			m.logger.Printf("Output: %s", string(output))
			return
		}

		m.logger.Printf("✅ Data fetch completed for %s/%s (duration: %v)",
			industry, country, duration)

		// TODO: Trigger import to database
		// For now, just log success
	}()

	return nil
}

// TriggerDataFetchBatch triggers multiple fetches in batch
func (m *DataMonitor) TriggerDataFetchBatch(ctx context.Context, pairs []IndustryCountryPair, limit int, maxConcurrent int) error {
	m.logger.Printf("Triggering batch fetch for %d pairs (max concurrent: %d)", len(pairs), maxConcurrent)

	semaphore := make(chan struct{}, maxConcurrent)
	errChan := make(chan error, len(pairs))

	for _, pair := range pairs {
		semaphore <- struct{}{} // Acquire semaphore

		go func(p IndustryCountryPair) {
			defer func() { <-semaphore }() // Release semaphore

			if err := m.TriggerDataFetch(ctx, p.Industry, p.Country, limit); err != nil {
				errChan <- fmt.Errorf("%s/%s: %w", p.Industry, p.Country, err)
			}

			// Rate limiting between requests
			time.Sleep(2 * time.Second)
		}(pair)
	}

	// Wait for all goroutines to finish
	for i := 0; i < maxConcurrent; i++ {
		semaphore <- struct{}{}
	}

	close(errChan)

	// Collect errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		m.logger.Printf("Batch fetch completed with %d errors", len(errors))
		return fmt.Errorf("batch fetch had %d errors", len(errors))
	}

	m.logger.Printf("Batch fetch completed successfully")
	return nil
}

// GetPopulationStats returns statistics about data population
func (m *DataMonitor) GetPopulationStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total leads
	totalLeads, err := m.db.Lead.Query().Count(ctx)
	if err != nil {
		return nil, err
	}
	stats["total_leads"] = totalLeads

	// Get all leads for grouping
	leads, err := m.db.Lead.Query().
		Select(lead.FieldIndustry, lead.FieldCountry).
		All(ctx)
	if err != nil {
		return nil, err
	}

	// Group by industry
	industryStats := make(map[string]int)
	for _, l := range leads {
		industryStats[string(l.Industry)]++
	}
	stats["top_industries"] = industryStats

	// Group by country
	countryStats := make(map[string]int)
	for _, l := range leads {
		countryStats[l.Country]++
	}
	stats["top_countries"] = countryStats

	// Coverage (unique combinations)
	combinationsMap := make(map[string]bool)
	for _, l := range leads {
		key := fmt.Sprintf("%s:%s", string(l.Industry), l.Country)
		combinationsMap[key] = true
	}
	stats["total_combinations"] = len(combinationsMap)

	return stats, nil
}

// CacheKey generates cache key for data fetch tracking
func (m *DataMonitor) CacheKey(industry, country string) string {
	return fmt.Sprintf("data_fetch:%s:%s", industry, country)
}

// MarkFetchInProgress marks a fetch as in progress (to avoid duplicates)
func (m *DataMonitor) MarkFetchInProgress(ctx context.Context, industry, country string) error {
	key := m.CacheKey(industry, country)

	// Set with 1 hour expiration
	value := map[string]interface{}{
		"status":     "in_progress",
		"started_at": time.Now().Unix(),
	}

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return m.cache.Set(ctx, key, string(data), time.Hour)
}

// IsFetchInProgress checks if a fetch is already in progress
func (m *DataMonitor) IsFetchInProgress(ctx context.Context, industry, country string) (bool, error) {
	key := m.CacheKey(industry, country)

	value, err := m.cache.Get(ctx, key)
	if err != nil {
		if err.Error() == "redis: nil" {
			return false, nil
		}
		return false, err
	}

	return value != "", nil
}

// ClearFetchStatus clears fetch status from cache
func (m *DataMonitor) ClearFetchStatus(ctx context.Context, industry, country string) error {
	key := m.CacheKey(industry, country)
	return m.cache.Delete(ctx, key)
}
