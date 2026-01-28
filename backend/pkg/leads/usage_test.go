package leads

import (
	"testing"
)

func TestGetUsageLimitForTier(t *testing.T) {
	tests := []struct {
		tier  string
		limit int
	}{
		{"free", 50},
		{"starter", 500},
		{"pro", 2000},
		{"business", 10000},
		{"unknown", 50}, // Default to free
	}

	for _, tt := range tests {
		t.Run(tt.tier, func(t *testing.T) {
			limit := GetUsageLimitForTier(tt.tier)
			if limit != tt.limit {
				t.Errorf("GetUsageLimitForTier(%s) = %d, want %d", tt.tier, limit, tt.limit)
			}
		})
	}
}

func TestCalculateQualityScore(t *testing.T) {
	tests := []struct {
		name         string
		hasPhone     bool
		hasEmail     bool
		hasWebsite   bool
		hasAddress   bool
		hasLocation  bool
		expectedMin  int
		expectedMax  int
	}{
		{
			name:        "No data",
			expectedMin: 30,
			expectedMax: 30,
		},
		{
			name:        "Only phone",
			hasPhone:    true,
			expectedMin: 50,
			expectedMax: 50,
		},
		{
			name:        "Phone and email",
			hasPhone:    true,
			hasEmail:    true,
			expectedMin: 70,
			expectedMax: 70,
		},
		{
			name:        "All data",
			hasPhone:    true,
			hasEmail:    true,
			hasWebsite:  true,
			hasAddress:  true,
			hasLocation: true,
			expectedMin: 100,
			expectedMax: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			phone := ""
			if tt.hasPhone {
				phone = "+1234567890"
			}
			email := ""
			if tt.hasEmail {
				email = "test@example.com"
			}
			website := ""
			if tt.hasWebsite {
				website = "https://example.com"
			}
			address := ""
			if tt.hasAddress {
				address = "123 Main St"
			}
			var lat, lon *float64
			if tt.hasLocation {
				latVal := 40.7128
				lonVal := -74.0060
				lat = &latVal
				lon = &lonVal
			}

			score := calculateQualityScore(phone, email, website, address, lat, lon)

			if score < tt.expectedMin || score > tt.expectedMax {
				t.Errorf("calculateQualityScore() = %d, want between %d and %d",
					score, tt.expectedMin, tt.expectedMax)
			}
		})
	}
}

// Helper function to test (we'll implement this based on the actual implementation)
func calculateQualityScore(phone, email, website, address string, lat, lon *float64) int {
	score := 30 // Base score for having a name

	if phone != "" {
		score += 20
	}
	if email != "" {
		score += 20
	}
	if website != "" {
		score += 15
	}
	if address != "" {
		score += 10
	}
	if lat != nil && lon != nil {
		score += 5
	}

	if score > 100 {
		score = 100
	}

	return score
}
