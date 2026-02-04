package graph

import (
	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/pkg/analytics"
	"github.com/jordanlanch/industrydb/pkg/domain"
	"github.com/jordanlanch/industrydb/pkg/export"
	"github.com/jordanlanch/industrydb/pkg/leads"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

type Resolver struct {
	DB                 *ent.Client
	LeadService        *leads.Service
	ExportService      *export.Service
	AnalyticsService   *analytics.Service
	TokenBlacklist     domain.TokenBlacklist
	JWTSecret          string
	JWTExpirationHours int
}
