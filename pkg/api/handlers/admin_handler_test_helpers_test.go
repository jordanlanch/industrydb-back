package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/user"
	custommiddleware "github.com/jordanlanch/industrydb/pkg/middleware"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// createAdminAndRegularUser creates an admin (superadmin) and regular user for testing middleware.
func createAdminAndRegularUser(t *testing.T, client *ent.Client) (*ent.User, *ent.User, func()) {
	t.Helper()
	ctx := context.Background()

	admin, err := client.User.Create().
		SetEmail("admin-mw-test@example.com").
		SetName("Admin User").
		SetPasswordHash("hashed_password").
		SetRole(user.RoleSuperadmin).
		SetSubscriptionTier(user.SubscriptionTierFree).
		SetUsageCount(0).
		SetUsageLimit(50).
		SetEmailVerifiedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	regularUser, err := client.User.Create().
		SetEmail("regular-mw-test@example.com").
		SetName("Regular User").
		SetPasswordHash("hashed_password").
		SetRole(user.RoleUser).
		SetSubscriptionTier(user.SubscriptionTierStarter).
		SetUsageCount(10).
		SetUsageLimit(500).
		Save(ctx)
	require.NoError(t, err)

	return admin, regularUser, func() { client.Close() }
}

// requireAdminMiddleware returns the RequireAdmin middleware for testing.
func requireAdminMiddleware(db *ent.Client) echo.MiddlewareFunc {
	return custommiddleware.RequireAdmin(db)
}
