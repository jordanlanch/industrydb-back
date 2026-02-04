package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateReturnURL(t *testing.T) {
	const defaultURL = "https://industrydb.io/dashboard/settings/billing"

	tests := []struct {
		name     string
		input    string
		expected string
		reason   string
	}{
		{
			name:     "empty URL returns default",
			input:    "",
			expected: defaultURL,
			reason:   "Empty URL should return default",
		},
		{
			name:     "localhost development URL is allowed (root setup)",
			input:    "http://localhost:5678/dashboard/settings/billing",
			expected: "http://localhost:5678/dashboard/settings/billing",
			reason:   "Localhost development URL (root setup) should be allowed",
		},
		{
			name:     "localhost development URL is allowed (modular setup)",
			input:    "http://localhost:5566/dashboard/settings/billing",
			expected: "http://localhost:5566/dashboard/settings/billing",
			reason:   "Localhost development URL (modular setup) should be allowed",
		},
		{
			name:     "production URL is allowed",
			input:    "https://industrydb.io/dashboard/settings/billing?success=true",
			expected: "https://industrydb.io/dashboard/settings/billing?success=true",
			reason:   "Production URL with query params should be allowed",
		},
		{
			name:     "www production URL is allowed",
			input:    "https://www.industrydb.io/dashboard/settings/billing",
			expected: "https://www.industrydb.io/dashboard/settings/billing",
			reason:   "WWW production URL should be allowed",
		},
		{
			name:     "malicious external URL is blocked",
			input:    "https://evil.com/phishing",
			expected: defaultURL,
			reason:   "External malicious URL should be blocked",
		},
		{
			name:     "open redirect attempt is blocked",
			input:    "https://attacker.com/steal-credentials",
			expected: defaultURL,
			reason:   "Open redirect attempt should be blocked",
		},
		{
			name:     "subdomain attack is blocked",
			input:    "https://industrydb.io.evil.com/fake",
			expected: defaultURL,
			reason:   "Subdomain attack should be blocked",
		},
		{
			name:     "invalid URL format returns default",
			input:    "not-a-valid-url",
			expected: defaultURL,
			reason:   "Invalid URL format should return default",
		},
		{
			name:     "javascript protocol is blocked",
			input:    "javascript:alert('xss')",
			expected: defaultURL,
			reason:   "JavaScript protocol should be blocked",
		},
		{
			name:     "data protocol is blocked",
			input:    "data:text/html,<script>alert('xss')</script>",
			expected: defaultURL,
			reason:   "Data protocol should be blocked",
		},
		{
			name:     "localhost without port is blocked",
			input:    "http://localhost/dashboard",
			expected: defaultURL,
			reason:   "Localhost without specific port should be blocked for security",
		},
		{
			name:     "localhost with wrong port is blocked",
			input:    "http://localhost:8080/dashboard",
			expected: defaultURL,
			reason:   "Localhost with wrong port should be blocked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateReturnURL(tt.input)
			assert.Equal(t, tt.expected, result, tt.reason)
		})
	}
}

func TestValidateReturnURL_Whitelist(t *testing.T) {
	// Test that ONLY whitelisted hosts are allowed
	allowedURLs := []string{
		"http://localhost:5678/any/path",
		"http://localhost:5566/any/path",
		"https://localhost:5678/any/path",
		"http://industrydb.io/any/path",
		"https://industrydb.io/any/path",
		"http://www.industrydb.io/any/path",
		"https://www.industrydb.io/any/path",
	}

	for _, url := range allowedURLs {
		result := validateReturnURL(url)
		assert.Equal(t, url, result, "Allowed URL should pass through: "+url)
	}
}

func TestValidateReturnURL_Security(t *testing.T) {
	// Test various attack vectors
	const defaultURL = "https://industrydb.io/dashboard/settings/billing"

	attackVectors := []string{
		"https://evil.com",
		"https://industrydb.io.attacker.com",
		"https://attacker.com@industrydb.io",
		"https://industrydb.io:8080@attacker.com",
		"//evil.com/phishing",
		"///evil.com/phishing",
		"http://evil.com",
		"ftp://industrydb.io",
		"file:///etc/passwd",
	}

	for _, attack := range attackVectors {
		result := validateReturnURL(attack)
		assert.Equal(t, defaultURL, result, "Attack vector should be blocked: "+attack)
	}
}
