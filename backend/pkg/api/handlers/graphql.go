package handlers

import (
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/graph"
	"github.com/jordanlanch/industrydb/pkg/leads"
	"github.com/labstack/echo/v4"
)

// GraphQLHandler creates GraphQL server handler
type GraphQLHandler struct {
	resolver *graph.Resolver
}

// NewGraphQLHandler creates a new GraphQL handler
func NewGraphQLHandler(db *ent.Client, leadService *leads.Service, jwtSecret string, jwtExpirationHours int) *GraphQLHandler {
	resolver := &graph.Resolver{
		DB:                 db,
		LeadService:        leadService,
		JWTSecret:          jwtSecret,
		JWTExpirationHours: jwtExpirationHours,
	}

	return &GraphQLHandler{
		resolver: resolver,
	}
}

// GraphQLEndpoint handles GraphQL queries
func (h *GraphQLHandler) GraphQLEndpoint(c echo.Context) error {
	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: h.resolver}))

	// Wrap the GraphQL handler
	srv.ServeHTTP(c.Response(), c.Request())
	return nil
}

// Playground serves the GraphQL Playground interface
func (h *GraphQLHandler) Playground(c echo.Context) error {
	pg := playground.Handler("GraphQL Playground", "/graphql")
	pg.ServeHTTP(c.Response(), c.Request())
	return nil
}
