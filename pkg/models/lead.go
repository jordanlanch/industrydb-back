package models

// LeadSearchRequest represents search parameters for leads
type LeadSearchRequest struct {
	// Full-text search
	Query       string   `query:"q"`
	Industry    string   `query:"industry" validate:"omitempty,oneof=tattoo beauty barber gym restaurant cafe bar bakery dentist pharmacy massage car_repair car_wash car_dealer clothing convenience lawyer accountant spa nail_salon"`
	SubNiche    string   `query:"sub_niche"`
	Specialties []string `query:"specialties"`
	CuisineType string   `query:"cuisine_type"`
	SportType   string   `query:"sport_type"`
	TattooStyle string   `query:"tattoo_style"`
	Country     string   `query:"country" validate:"omitempty,len=2"`
	City        string   `query:"city"`
	HasEmail       *bool    `query:"has_email"`
	HasPhone       *bool    `query:"has_phone"`
	HasWebsite     *bool    `query:"has_website"`
	HasSocialMedia *bool    `query:"has_social_media"`
	Verified       *bool    `query:"verified"`
	// Radius search parameters
	Latitude  *float64 `query:"latitude" validate:"omitempty,min=-90,max=90"`
	Longitude *float64 `query:"longitude" validate:"omitempty,min=-180,max=180"`
	Radius    *float64 `query:"radius" validate:"omitempty,min=0"`
	Unit      string   `query:"unit" validate:"omitempty,oneof=km miles"`
	// Sorting
	SortBy string `query:"sort_by" validate:"omitempty,oneof=newest quality_score distance verified relevance"`
	Page   int    `query:"page" validate:"min=1"`
	Limit  int    `query:"limit" validate:"min=1,max=100"`
}

// LeadResponse represents a single lead in API responses
type LeadResponse struct {
	ID           int               `json:"id"`
	Name         string            `json:"name"`
	Industry     string            `json:"industry"`
	SubNiche     string            `json:"sub_niche,omitempty"`
	Specialties  []string          `json:"specialties,omitempty"`
	CuisineType  string            `json:"cuisine_type,omitempty"`
	SportType    string            `json:"sport_type,omitempty"`
	TattooStyle  string            `json:"tattoo_style,omitempty"`
	Country      string            `json:"country"`
	City         string            `json:"city"`
	Address      string            `json:"address,omitempty"`
	PostalCode   string            `json:"postal_code,omitempty"`
	Phone        string            `json:"phone,omitempty"`
	Email        string            `json:"email,omitempty"`
	Website      string            `json:"website,omitempty"`
	SocialMedia  map[string]string `json:"social_media,omitempty"`
	Latitude     float64           `json:"latitude,omitempty"`
	Longitude    float64           `json:"longitude,omitempty"`
	Verified     bool              `json:"verified"`
	QualityScore int               `json:"quality_score"`
	CreatedAt    string            `json:"created_at"`
}

// LeadListResponse represents a paginated list of leads
type LeadListResponse struct {
	Data       []LeadResponse   `json:"data"`
	Pagination PaginationInfo   `json:"pagination"`
	Filters    AppliedFilters   `json:"filters"`
}

// PaginationInfo contains pagination metadata
type PaginationInfo struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

// AppliedFilters shows what filters were applied to the search
type AppliedFilters struct {
	Industry    string   `json:"industry,omitempty"`
	SubNiche    string   `json:"sub_niche,omitempty"`
	Specialties []string `json:"specialties,omitempty"`
	CuisineType string   `json:"cuisine_type,omitempty"`
	SportType   string   `json:"sport_type,omitempty"`
	TattooStyle string   `json:"tattoo_style,omitempty"`
	Country     string   `json:"country,omitempty"`
	City        string   `json:"city,omitempty"`
	HasEmail       *bool    `json:"has_email,omitempty"`
	HasPhone       *bool    `json:"has_phone,omitempty"`
	HasWebsite     *bool    `json:"has_website,omitempty"`
	HasSocialMedia *bool    `json:"has_social_media,omitempty"`
	Verified       *bool    `json:"verified,omitempty"`
}

// ExportRequest represents an export request
type ExportRequest struct {
	Format      string             `json:"format" validate:"required,oneof=csv excel"`
	Filters     LeadSearchRequest  `json:"filters"`
	MaxLeads    int                `json:"max_leads" validate:"min=1,max=10000"`
}

// ExportResponse represents an export response
type ExportResponse struct {
	ID          int    `json:"id"`
	Status      string `json:"status"`
	Format      string `json:"format"`
	LeadCount   int    `json:"lead_count"`
	FileURL     string `json:"file_url,omitempty"`
	ExpiresAt   string `json:"expires_at,omitempty"`
	CreatedAt   string `json:"created_at"`
}

// ExportListResponse represents a list of exports
type ExportListResponse struct {
	Data       []ExportResponse `json:"data"`
	Pagination PaginationInfo   `json:"pagination"`
}

// UsageInfo represents user usage statistics
type UsageInfo struct {
	UsageCount int    `json:"usage_count"`
	UsageLimit int    `json:"usage_limit"`
	Remaining  int    `json:"remaining"`
	ResetAt    string `json:"reset_at"`
	Tier       string `json:"tier"`
}

// LeadPreviewResponse represents search preview statistics (without charging credits)
type LeadPreviewResponse struct {
	EstimatedCount  int     `json:"estimated_count"`
	WithEmailCount  int     `json:"with_email_count"`
	WithEmailPct    float64 `json:"with_email_pct"`
	WithPhoneCount  int     `json:"with_phone_count"`
	WithPhonePct    float64 `json:"with_phone_pct"`
	VerifiedCount   int     `json:"verified_count"`
	VerifiedPct     float64 `json:"verified_pct"`
	QualityScoreAvg float64 `json:"quality_score_avg"`
}
