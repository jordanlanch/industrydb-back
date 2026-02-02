package export

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/export"
	"github.com/jordanlanch/industrydb/ent/usagelog"
	"github.com/jordanlanch/industrydb/pkg/analytics"
	"github.com/jordanlanch/industrydb/pkg/leads"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/xuri/excelize/v2"
)

// Service handles export business logic
type Service struct {
	db               *ent.Client
	leadService      *leads.Service
	analyticsService *analytics.Service
	storagePath      string
}

// NewService creates a new export service
func NewService(db *ent.Client, leadService *leads.Service, analyticsService *analytics.Service, storagePath string) *Service {
	// Ensure storage directory exists
	os.MkdirAll(storagePath, 0755)

	return &Service{
		db:               db,
		leadService:      leadService,
		analyticsService: analyticsService,
		storagePath:      storagePath,
	}
}

// CreateExport creates a new export with the given filters
// organizationID is optional - pass nil for personal exports
func (s *Service) CreateExport(ctx context.Context, userID int, organizationID *int, req models.ExportRequest) (*models.ExportResponse, error) {
	// Validate format
	if req.Format != "csv" && req.Format != "excel" {
		return nil, fmt.Errorf("invalid format: must be csv or excel")
	}

	// Set max leads if not specified
	if req.MaxLeads == 0 {
		req.MaxLeads = 1000
	}
	if req.MaxLeads > 10000 {
		req.MaxLeads = 10000
	}

	// Convert filters to map
	filtersMap := make(map[string]interface{})
	filtersBytes, err := json.Marshal(req.Filters)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal filters: %w", err)
	}
	if err := json.Unmarshal(filtersBytes, &filtersMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal filters: %w", err)
	}

	// Create export record
	creator := s.db.Export.Create().
		SetUserID(userID).
		SetFormat(export.Format(req.Format)).
		SetFiltersApplied(filtersMap).
		SetLeadCount(0).
		SetStatus(export.StatusPending).
		SetExpiresAt(time.Now().Add(24 * time.Hour))

	// Set organization_id if provided
	if organizationID != nil {
		creator = creator.SetOrganizationID(*organizationID)
	}

	exp, err := creator.Save(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to create export: %w", err)
	}

	// Process export asynchronously
	go s.processExport(exp.ID, userID, req)

	return s.toExportResponse(exp), nil
}

// processExport processes the export in the background
func (s *Service) processExport(exportID, userID int, req models.ExportRequest) {
	ctx := context.Background()

	// Update status to processing
	s.db.Export.UpdateOneID(exportID).
		SetStatus(export.StatusProcessing).
		SaveX(ctx)

	// Get leads with filters
	req.Filters.Limit = req.MaxLeads
	req.Filters.Page = 1

	results, err := s.leadService.Search(ctx, req.Filters)
	if err != nil {
		s.db.Export.UpdateOneID(exportID).
			SetStatus(export.StatusFailed).
			SetErrorMessage(err.Error()).
			SaveX(ctx)
		return
	}

	// Generate filename
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("export-%d-%s.%s", exportID, timestamp, req.Format)
	filepath := filepath.Join(s.storagePath, filename)

	// Generate file based on format
	var genErr error
	if req.Format == "csv" {
		genErr = s.generateCSV(filepath, results.Data)
	} else {
		genErr = s.generateExcel(filepath, results.Data)
	}

	if genErr != nil {
		s.db.Export.UpdateOneID(exportID).
			SetStatus(export.StatusFailed).
			SetErrorMessage(genErr.Error()).
			SaveX(ctx)
		return
	}

	// Update export record
	s.db.Export.UpdateOneID(exportID).
		SetStatus(export.StatusReady).
		SetLeadCount(len(results.Data)).
		SetFilePath(filepath).
		SetFileURL(fmt.Sprintf("/api/v1/exports/%d/download", exportID)).
		SaveX(ctx)

	// Log analytics with actual lead count
	metadata := map[string]interface{}{
		"format":     req.Format,
		"max_leads":  req.MaxLeads,
		"filters":    req.Filters,
		"lead_count": len(results.Data),
		"export_id":  exportID,
	}
	if err := s.analyticsService.LogUsage(ctx, userID, usagelog.ActionExport, len(results.Data), metadata); err != nil {
		// Log error but don't fail the export
		fmt.Printf("Failed to log export analytics: %v\n", err)
	}
}

// generateCSV generates a CSV file from leads
func (s *Service) generateCSV(filepath string, leads []models.LeadResponse) error {
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"ID", "Name", "Industry", "Country", "City", "Address", "Postal Code",
		"Phone", "Email", "Website", "Latitude", "Longitude", "Verified",
		"Quality Score", "Created At",
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write data
	for _, lead := range leads {
		row := []string{
			strconv.Itoa(lead.ID),
			lead.Name,
			lead.Industry,
			lead.Country,
			lead.City,
			lead.Address,
			lead.PostalCode,
			lead.Phone,
			lead.Email,
			lead.Website,
			fmt.Sprintf("%.6f", lead.Latitude),
			fmt.Sprintf("%.6f", lead.Longitude),
			strconv.FormatBool(lead.Verified),
			strconv.Itoa(lead.QualityScore),
			lead.CreatedAt,
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write row: %w", err)
		}
	}

	return nil
}

// generateExcel generates an Excel file from leads
func (s *Service) generateExcel(filepath string, leads []models.LeadResponse) error {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Leads"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return fmt.Errorf("failed to create sheet: %w", err)
	}

	// Set header style
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
	})
	if err != nil {
		return fmt.Errorf("failed to create style: %w", err)
	}

	// Write header
	headers := []string{
		"ID", "Name", "Industry", "Country", "City", "Address", "Postal Code",
		"Phone", "Email", "Website", "Latitude", "Longitude", "Verified",
		"Quality Score", "Created At",
	}

	for i, header := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetName, cell, header)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	// Write data
	for rowIdx, lead := range leads {
		row := rowIdx + 2 // Start from row 2 (after header)
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), lead.ID)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), lead.Name)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), lead.Industry)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), lead.Country)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), lead.City)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), lead.Address)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), lead.PostalCode)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), lead.Phone)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), lead.Email)
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), lead.Website)
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", row), lead.Latitude)
		f.SetCellValue(sheetName, fmt.Sprintf("L%d", row), lead.Longitude)
		f.SetCellValue(sheetName, fmt.Sprintf("M%d", row), lead.Verified)
		f.SetCellValue(sheetName, fmt.Sprintf("N%d", row), lead.QualityScore)
		f.SetCellValue(sheetName, fmt.Sprintf("O%d", row), lead.CreatedAt)
	}

	// Auto-fit columns
	for i := 0; i < len(headers); i++ {
		col := string(rune('A' + i))
		f.SetColWidth(sheetName, col, col, 15)
	}

	// Set active sheet
	f.SetActiveSheet(index)

	// Save file
	if err := f.SaveAs(filepath); err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}

// GetExport retrieves an export by ID
func (s *Service) GetExport(ctx context.Context, userID, exportID int) (*models.ExportResponse, error) {
	exp, err := s.db.Export.Query().
		Where(export.IDEQ(exportID), export.UserIDEQ(userID)).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("export not found")
		}
		return nil, fmt.Errorf("failed to get export: %w", err)
	}

	return s.toExportResponse(exp), nil
}

// ListExports lists all exports for a user or organization
// organizationID is optional - pass nil to list personal exports
func (s *Service) ListExports(ctx context.Context, userID int, organizationID *int, page, limit int) (*models.ExportListResponse, error) {
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// Build query with user_id filter
	query := s.db.Export.Query().Where(export.UserIDEQ(userID))

	// Add organization filter if provided
	if organizationID != nil {
		query = query.Where(export.OrganizationIDEQ(*organizationID))
	} else {
		// Only show personal exports (no organization)
		query = query.Where(export.OrganizationIDIsNil())
	}

	// Get total count
	total, err := query.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count exports: %w", err)
	}

	// Get paginated results
	offset := (page - 1) * limit
	totalPages := (total + limit - 1) / limit

	exports, err := query.
		Order(ent.Desc(export.FieldCreatedAt)).
		Limit(limit).
		Offset(offset).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to list exports: %w", err)
	}

	// Convert to response
	exportResponses := make([]models.ExportResponse, len(exports))
	for i, exp := range exports {
		exportResponses[i] = *s.toExportResponse(exp)
	}

	return &models.ExportListResponse{
		Data: exportResponses,
		Pagination: models.PaginationInfo{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
			HasNext:    page < totalPages,
			HasPrev:    page > 1,
		},
	}, nil
}

// GetFilePath returns the file path for an export
func (s *Service) GetFilePath(ctx context.Context, userID, exportID int) (string, error) {
	exp, err := s.db.Export.Query().
		Where(export.IDEQ(exportID), export.UserIDEQ(userID)).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return "", fmt.Errorf("export not found")
		}
		return "", fmt.Errorf("failed to get export: %w", err)
	}

	if exp.Status != export.StatusReady {
		return "", fmt.Errorf("export not ready: status is %s", exp.Status)
	}

	if !exp.ExpiresAt.IsZero() && time.Now().After(exp.ExpiresAt) {
		return "", fmt.Errorf("export has expired")
	}

	if exp.FilePath == "" {
		return "", fmt.Errorf("file path not set")
	}

	return exp.FilePath, nil
}

// toExportResponse converts an Ent export to a response model
func (s *Service) toExportResponse(exp *ent.Export) *models.ExportResponse {
	response := &models.ExportResponse{
		ID:        exp.ID,
		Status:    string(exp.Status),
		Format:    string(exp.Format),
		LeadCount: exp.LeadCount,
		CreatedAt: exp.CreatedAt.Format(time.RFC3339),
	}

	if exp.FileURL != "" {
		response.FileURL = exp.FileURL
	}

	if !exp.ExpiresAt.IsZero() {
		response.ExpiresAt = exp.ExpiresAt.Format(time.RFC3339)
	}

	return response
}
