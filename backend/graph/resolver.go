package graph

import (
	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/pkg/leads"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

type Resolver struct {
	DB                 *ent.Client
	LeadService        *leads.Service
	JWTSecret          string
	JWTExpirationHours int
}
