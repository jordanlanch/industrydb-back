package phone

import (
	"fmt"

	"github.com/nyaruka/phonenumbers"
)

// PhoneFormat represents different phone number format types.
type PhoneFormat int

const (
	// FormatE164 is the E.164 format (+15551234567).
	FormatE164 PhoneFormat = iota
	// FormatInternational is the international format (+1 555-123-4567).
	FormatInternational
	// FormatNational is the national format ((555) 123-4567).
	FormatNational
	// FormatRFC3966 is the RFC3966 format (tel:+1-555-123-4567).
	FormatRFC3966
)

// PhoneType represents the type of phone number.
type PhoneType string

const (
	// TypeFixedLine represents a fixed-line number.
	TypeFixedLine PhoneType = "FIXED_LINE"
	// TypeMobile represents a mobile number.
	TypeMobile PhoneType = "MOBILE"
	// TypeFixedLineOrMobile represents a number that could be either.
	TypeFixedLineOrMobile PhoneType = "FIXED_LINE_OR_MOBILE"
	// TypeTollFree represents a toll-free number.
	TypeTollFree PhoneType = "TOLL_FREE"
	// TypePremiumRate represents a premium rate number.
	TypePremiumRate PhoneType = "PREMIUM_RATE"
	// TypeSharedCost represents a shared cost number.
	TypeSharedCost PhoneType = "SHARED_COST"
	// TypeVoip represents a VoIP number.
	TypeVoip PhoneType = "VOIP"
	// TypePersonalNumber represents a personal number.
	TypePersonalNumber PhoneType = "PERSONAL_NUMBER"
	// TypePager represents a pager number.
	TypePager PhoneType = "PAGER"
	// TypeUAN represents a UAN (Universal Access Number).
	TypeUAN PhoneType = "UAN"
	// TypeVoicemail represents a voicemail number.
	TypeVoicemail PhoneType = "VOICEMAIL"
	// TypeUnknown represents an unknown type.
	TypeUnknown PhoneType = "UNKNOWN"
)

// ValidationResult contains the result of phone number validation.
type ValidationResult struct {
	IsValid            bool      `json:"is_valid"`
	E164Format         string    `json:"e164_format"`
	InternationalFormat string    `json:"international_format"`
	NationalFormat     string    `json:"national_format"`
	CountryCode        string    `json:"country_code"`
	PhoneType          PhoneType `json:"phone_type"`
}

// ValidatePhone validates a phone number and returns detailed information.
func ValidatePhone(phone, countryCode string) (*ValidationResult, error) {
	if phone == "" {
		return nil, fmt.Errorf("phone number cannot be empty")
	}

	if countryCode == "" {
		countryCode = "US" // Default to US
	}

	// Parse the phone number
	parsed, err := phonenumbers.Parse(phone, countryCode)
	if err != nil {
		return nil, fmt.Errorf("failed to parse phone number: %w", err)
	}

	// Check if valid
	isValid := phonenumbers.IsValidNumber(parsed)

	// Get formats
	e164 := phonenumbers.Format(parsed, phonenumbers.E164)
	international := phonenumbers.Format(parsed, phonenumbers.INTERNATIONAL)
	national := phonenumbers.Format(parsed, phonenumbers.NATIONAL)

	// Get country code
	region := phonenumbers.GetRegionCodeForNumber(parsed)

	// Get phone type
	phoneType := getPhoneTypeString(phonenumbers.GetNumberType(parsed))

	return &ValidationResult{
		IsValid:            isValid,
		E164Format:         e164,
		InternationalFormat: international,
		NationalFormat:     national,
		CountryCode:        region,
		PhoneType:          phoneType,
	}, nil
}

// FormatPhone formats a phone number in the specified format.
func FormatPhone(phone, countryCode string, format PhoneFormat) (string, error) {
	if phone == "" {
		return "", fmt.Errorf("phone number cannot be empty")
	}

	if countryCode == "" {
		countryCode = "US"
	}

	parsed, err := phonenumbers.Parse(phone, countryCode)
	if err != nil {
		return "", fmt.Errorf("failed to parse phone number: %w", err)
	}

	var phoneFormat phonenumbers.PhoneNumberFormat
	switch format {
	case FormatE164:
		phoneFormat = phonenumbers.E164
	case FormatInternational:
		phoneFormat = phonenumbers.INTERNATIONAL
	case FormatNational:
		phoneFormat = phonenumbers.NATIONAL
	case FormatRFC3966:
		phoneFormat = phonenumbers.RFC3966
	default:
		phoneFormat = phonenumbers.E164
	}

	return phonenumbers.Format(parsed, phoneFormat), nil
}

// NormalizePhone normalizes a phone number to E.164 format.
func NormalizePhone(phone, countryCode string) (string, error) {
	if phone == "" {
		return "", fmt.Errorf("phone number cannot be empty")
	}

	if countryCode == "" {
		countryCode = "US"
	}

	parsed, err := phonenumbers.Parse(phone, countryCode)
	if err != nil {
		return "", fmt.Errorf("failed to parse phone number: %w", err)
	}

	// Validate the number
	if !phonenumbers.IsValidNumber(parsed) {
		return "", fmt.Errorf("invalid phone number")
	}

	return phonenumbers.Format(parsed, phonenumbers.E164), nil
}

// GetCountryCode extracts the country code from a phone number.
func GetCountryCode(phone string) (string, error) {
	if phone == "" {
		return "", fmt.Errorf("phone number cannot be empty")
	}

	// Parse without region hint (requires international format)
	parsed, err := phonenumbers.Parse(phone, "ZZ")
	if err != nil {
		return "", fmt.Errorf("failed to parse phone number (must include country code): %w", err)
	}

	region := phonenumbers.GetRegionCodeForNumber(parsed)
	if region == "ZZ" || region == "" {
		return "", fmt.Errorf("unable to determine country code")
	}

	return region, nil
}

// GetPhoneType returns the type of the phone number.
func GetPhoneType(phone, countryCode string) (PhoneType, error) {
	if phone == "" {
		return TypeUnknown, fmt.Errorf("phone number cannot be empty")
	}

	if countryCode == "" {
		countryCode = "US"
	}

	parsed, err := phonenumbers.Parse(phone, countryCode)
	if err != nil {
		return TypeUnknown, fmt.Errorf("failed to parse phone number: %w", err)
	}

	numberType := phonenumbers.GetNumberType(parsed)
	return getPhoneTypeString(numberType), nil
}

// getPhoneTypeString converts phonenumbers.PhoneNumberType to PhoneType string.
func getPhoneTypeString(t phonenumbers.PhoneNumberType) PhoneType {
	switch t {
	case phonenumbers.FIXED_LINE:
		return TypeFixedLine
	case phonenumbers.MOBILE:
		return TypeMobile
	case phonenumbers.FIXED_LINE_OR_MOBILE:
		return TypeFixedLineOrMobile
	case phonenumbers.TOLL_FREE:
		return TypeTollFree
	case phonenumbers.PREMIUM_RATE:
		return TypePremiumRate
	case phonenumbers.SHARED_COST:
		return TypeSharedCost
	case phonenumbers.VOIP:
		return TypeVoip
	case phonenumbers.PERSONAL_NUMBER:
		return TypePersonalNumber
	case phonenumbers.PAGER:
		return TypePager
	case phonenumbers.UAN:
		return TypeUAN
	case phonenumbers.VOICEMAIL:
		return TypeVoicemail
	default:
		return TypeUnknown
	}
}

// IsValidForRegion checks if a phone number is valid for a specific region.
func IsValidForRegion(phone, countryCode string) (bool, error) {
	if phone == "" {
		return false, fmt.Errorf("phone number cannot be empty")
	}

	if countryCode == "" {
		return false, fmt.Errorf("country code cannot be empty")
	}

	parsed, err := phonenumbers.Parse(phone, countryCode)
	if err != nil {
		return false, fmt.Errorf("failed to parse phone number: %w", err)
	}

	return phonenumbers.IsValidNumberForRegion(parsed, countryCode), nil
}
