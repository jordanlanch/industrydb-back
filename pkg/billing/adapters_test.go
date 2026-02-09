package billing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmailServiceAdapterImplementsInterface(t *testing.T) {
	// Verify at compile time that EmailServiceAdapter implements EmailSender
	var _ EmailSender = &EmailServiceAdapter{}
}

func TestAuditServiceAdapterImplementsInterface(t *testing.T) {
	// Verify at compile time that AuditServiceAdapter implements AuditLogger
	var _ AuditLogger = &AuditServiceAdapter{}
}

func TestNewEmailServiceAdapter(t *testing.T) {
	adapter := NewEmailServiceAdapter(nil)
	assert.NotNil(t, adapter)
}

func TestNewAuditServiceAdapter(t *testing.T) {
	adapter := NewAuditServiceAdapter(nil)
	assert.NotNil(t, adapter)
}
