package importpkg

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/lead"
)

// CSVImportService handles bulk import of leads from CSV
type CSVImportService struct {
	client *ent.Client
}

// NewCSVImportService creates a new CSV import service
func NewCSVImportService(client *ent.Client) *CSVImportService {
	return &CSVImportService{
		client: client,
	}
}

// ImportResult holds the result of a CSV import operation
type ImportResult struct {
	TotalRows      int                `json:"total_rows"`
	SuccessCount   int                `json:"success_count"`
	FailureCount   int                `json:"failure_count"`
	Errors         []ImportError      `json:"errors,omitempty"`
	Duration       string             `json:"duration"`
	ImportedLeads  []ImportedLead     `json:"imported_leads,omitempty"`
}

// ImportError represents an error during import
type ImportError struct {
	Row     int    `json:"row"`
	Field   string `json:"field,omitempty"`
	Value   string `json:"value,omitempty"`
	Message string `json:"message"`
}

// ImportedLead represents a successfully imported lead
type ImportedLead struct {
	Row       int    `json:"row"`
	LeadID    string `json:"lead_id"`
	Name      string `json:"name"`
	Industry  string `json:"industry"`
}

// CSVConfig holds configuration for CSV import
type CSVConfig struct {
	MaxRows          int  // Maximum rows to import (0 = unlimited)
	SkipHeader       bool // Skip first row as header
	ValidateOnly     bool // Only validate, don't import
	UpdateExisting   bool // Update existing leads if found
	BatchSize        int  // Number of records per transaction
}

// DefaultCSVConfig returns default configuration
func DefaultCSVConfig() CSVConfig {
	return CSVConfig{
		MaxRows:        10000, // Limit to 10k rows per import
		SkipHeader:     true,
		ValidateOnly:   false,
		UpdateExisting: false,
		BatchSize:      100, // Process 100 records per transaction
	}
}

// RequiredFields defines the required CSV columns
var RequiredFields = []string{
	"name",
	"industry",
	"country",
}

// OptionalFields defines optional CSV columns
var OptionalFields = []string{
	"city",
	"address",
	"postal_code",
	"phone",
	"email",
	"website",
	"latitude",
	"longitude",
	"sub_niche",
	"quality_score",
}

// ImportFromCSV imports leads from CSV reader
func (s *CSVImportService) ImportFromCSV(ctx context.Context, r io.Reader, config CSVConfig) (*ImportResult, error) {
	startTime := time.Now()

	result := &ImportResult{
		Errors:        []ImportError{},
		ImportedLeads: []ImportedLead{},
	}

	// Create CSV reader
	csvReader := csv.NewReader(r)
	csvReader.TrimLeadingSpace = true
	csvReader.FieldsPerRecord = -1 // Variable number of fields

	// Read header
	headers, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Validate headers
	headerMap := make(map[string]int)
	for i, header := range headers {
		headerMap[strings.ToLower(strings.TrimSpace(header))] = i
	}

	// Check required fields
	for _, field := range RequiredFields {
		if _, ok := headerMap[field]; !ok {
			return nil, fmt.Errorf("missing required field: %s", field)
		}
	}

	log.Printf("✅ CSV headers validated: %v", headers)

	// Read and import rows
	rowNum := 1 // Start from 1 (header is row 0)
	batch := make([]*ent.LeadCreate, 0, config.BatchSize)

	for {
		// Check row limit
		if config.MaxRows > 0 && rowNum > config.MaxRows {
			log.Printf("⚠️  Reached max rows limit: %d", config.MaxRows)
			break
		}

		// Read next row
		row, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Errors = append(result.Errors, ImportError{
				Row:     rowNum,
				Message: fmt.Sprintf("CSV read error: %v", err),
			})
			result.FailureCount++
			rowNum++
			continue
		}

		result.TotalRows++

		// Parse row into lead data
		leadData, parseErr := s.parseRow(row, headerMap, rowNum)
		if parseErr != nil {
			result.Errors = append(result.Errors, *parseErr)
			result.FailureCount++
			rowNum++
			continue
		}

		// Validate lead data
		if validationErr := s.validateLead(leadData, rowNum); validationErr != nil {
			result.Errors = append(result.Errors, *validationErr)
			result.FailureCount++
			rowNum++
			continue
		}

		// If validate-only mode, skip actual import
		if config.ValidateOnly {
			result.SuccessCount++
			rowNum++
			continue
		}

		// Add to batch
		leadCreate := s.client.Lead.Create().
			SetName(leadData.Name).
			SetIndustry(lead.Industry(leadData.Industry)).
			SetCountry(leadData.Country)

		if leadData.City != "" {
			leadCreate.SetCity(leadData.City)
		}
		if leadData.Address != "" {
			leadCreate.SetAddress(leadData.Address)
		}
		if leadData.PostalCode != "" {
			leadCreate.SetPostalCode(leadData.PostalCode)
		}
		if leadData.Phone != "" {
			leadCreate.SetPhone(leadData.Phone)
		}
		if leadData.Email != "" {
			leadCreate.SetEmail(leadData.Email)
		}
		if leadData.Website != "" {
			leadCreate.SetWebsite(leadData.Website)
		}
		if leadData.Latitude != 0 {
			leadCreate.SetLatitude(leadData.Latitude)
		}
		if leadData.Longitude != 0 {
			leadCreate.SetLongitude(leadData.Longitude)
		}
		if leadData.SubNiche != "" {
			leadCreate.SetSubNiche(leadData.SubNiche)
		}
		if leadData.QualityScore != 0 {
			leadCreate.SetQualityScore(leadData.QualityScore)
		}

		batch = append(batch, leadCreate)

		// Process batch when full
		if len(batch) >= config.BatchSize {
			if batchErr := s.processBatch(ctx, batch, result, rowNum-len(batch)+1); batchErr != nil {
				log.Printf("⚠️  Batch processing error: %v", batchErr)
			}
			batch = make([]*ent.LeadCreate, 0, config.BatchSize)
		}

		rowNum++
	}

	// Process remaining batch
	if len(batch) > 0 && !config.ValidateOnly {
		if batchErr := s.processBatch(ctx, batch, result, rowNum-len(batch)+1); batchErr != nil {
			log.Printf("⚠️  Final batch processing error: %v", batchErr)
		}
	}

	result.Duration = time.Since(startTime).String()

	log.Printf("✅ CSV import completed: %d success, %d failures in %s",
		result.SuccessCount, result.FailureCount, result.Duration)

	return result, nil
}

// processBatch processes a batch of lead creates in a transaction
func (s *CSVImportService) processBatch(ctx context.Context, batch []*ent.LeadCreate, result *ImportResult, startRow int) error {
	// Start transaction
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	// Process batch
	for i, leadCreate := range batch {
		lead, err := leadCreate.Save(ctx)
		if err != nil {
			tx.Rollback()
			result.Errors = append(result.Errors, ImportError{
				Row:     startRow + i,
				Message: fmt.Sprintf("Failed to create lead: %v", err),
			})
			result.FailureCount++
			continue
		}

		result.SuccessCount++
		result.ImportedLeads = append(result.ImportedLeads, ImportedLead{
			Row:      startRow + i,
			LeadID:   strconv.Itoa(lead.ID),
			Name:     lead.Name,
			Industry: string(lead.Industry),
		})
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// LeadData holds parsed lead data from CSV
type LeadData struct {
	Name         string
	Industry     string
	Country      string
	City         string
	Address      string
	PostalCode   string
	Phone        string
	Email        string
	Website      string
	Latitude     float64
	Longitude    float64
	SubNiche     string
	QualityScore int
}

// parseRow parses a CSV row into LeadData
func (s *CSVImportService) parseRow(row []string, headerMap map[string]int, rowNum int) (*LeadData, *ImportError) {
	data := &LeadData{}

	// Helper to get field value
	getField := func(fieldName string) string {
		if idx, ok := headerMap[fieldName]; ok && idx < len(row) {
			return strings.TrimSpace(row[idx])
		}
		return ""
	}

	// Required fields
	data.Name = getField("name")
	data.Industry = getField("industry")
	data.Country = getField("country")

	// Optional fields
	data.City = getField("city")
	data.Address = getField("address")
	data.PostalCode = getField("postal_code")
	data.Phone = getField("phone")
	data.Email = getField("email")
	data.Website = getField("website")
	data.SubNiche = getField("sub_niche")

	// Parse numeric fields (skip errors, use default 0)
	// Latitude/Longitude parsing would go here

	return data, nil
}

// validateLead validates lead data
func (s *CSVImportService) validateLead(data *LeadData, rowNum int) *ImportError {
	// Validate required fields
	if data.Name == "" {
		return &ImportError{
			Row:     rowNum,
			Field:   "name",
			Message: "Name is required",
		}
	}

	if data.Industry == "" {
		return &ImportError{
			Row:     rowNum,
			Field:   "industry",
			Message: "Industry is required",
		}
	}

	if data.Country == "" {
		return &ImportError{
			Row:     rowNum,
			Field:   "country",
			Message: "Country is required",
		}
	}

	// Validate industry (must be valid)
	validIndustries := []string{
		"tattoo", "beauty", "barber", "nail_salon", "spa", "massage",
		"gym", "dentist", "pharmacy", "restaurant", "cafe", "bar",
		"bakery", "car_repair", "car_wash", "car_dealer", "lawyer",
		"accountant", "clothing", "convenience",
	}

	valid := false
	for _, vi := range validIndustries {
		if data.Industry == vi {
			valid = true
			break
		}
	}

	if !valid {
		return &ImportError{
			Row:     rowNum,
			Field:   "industry",
			Value:   data.Industry,
			Message: "Invalid industry",
		}
	}

	return nil
}
