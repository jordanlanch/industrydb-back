package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jordanlanch/industrydb/config"
	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/user"
	"github.com/jordanlanch/industrydb/pkg/api/errors"
	"github.com/jordanlanch/industrydb/pkg/audit"
	"github.com/jordanlanch/industrydb/pkg/auth"
	"github.com/jordanlanch/industrydb/pkg/cache"
	"github.com/jordanlanch/industrydb/pkg/email"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/jordanlanch/industrydb/pkg/oauth"
	"github.com/labstack/echo/v4"
	"github.com/go-playground/validator/v10"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	db           *ent.Client
	config       *config.Config
	blacklist    *auth.TokenBlacklist
	cache        *cache.Client
	auditLogger  *audit.Service
	emailService *email.Service
	validator    *validator.Validate
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(db *ent.Client, cfg *config.Config, blacklist *auth.TokenBlacklist, cache *cache.Client, auditLogger *audit.Service, emailService *email.Service) *AuthHandler {
	return &AuthHandler{
		db:           db,
		config:       cfg,
		blacklist:    blacklist,
		cache:        cache,
		auditLogger:  auditLogger,
		emailService: emailService,
		validator:    validator.New(),
	}
}

// Register godoc
// @Summary Register a new user
// @Description Create a new user account with email and password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.RegisterRequest true "Registration data"
// @Success 200 {object} models.AuthResponse "User registered successfully"
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 409 {object} models.ErrorResponse "User already exists"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /auth/register [post]
func (h *AuthHandler) Register(c echo.Context) error {
	var req models.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		return errors.ValidationError(c, err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	// Check if user already exists
	exists, err := h.db.User.Query().Where(user.EmailEQ(req.Email)).Exist(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "database_error",
		})
	}

	if exists {
		return c.JSON(http.StatusConflict, models.ErrorResponse{
			Error:   "user_exists",
			Message: "User with this email already exists",
		})
	}

	// Hash password
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "password_hashing_error",
		})
	}

	// Generate email verification token
	verificationToken, err := generateVerificationToken()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "token_generation_error",
		})
	}

	// Create user
	newUser, err := h.db.User.Create().
		SetEmail(req.Email).
		SetPasswordHash(hashedPassword).
		SetName(req.Name).
		SetSubscriptionTier(user.SubscriptionTierFree).
		SetUsageCount(0).
		SetUsageLimit(50).
		SetLastResetAt(time.Now()).
		SetAcceptedTermsAt(time.Now()).
		SetEmailVerificationToken(verificationToken).
		SetEmailVerificationTokenExpiresAt(time.Now().Add(24 * time.Hour)).
		Save(ctx)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "user_creation_error",
		})
	}

	// Log registration event
	ipAddress, userAgent := audit.GetRequestContext(c)
	go h.auditLogger.LogUserRegister(context.Background(), newUser.ID, ipAddress, userAgent)

	// Send verification email (async)
	go h.emailService.SendVerificationEmail(newUser.Email, newUser.Name, verificationToken)

	// Generate JWT
	token, err := auth.GenerateJWT(
		newUser.ID,
		newUser.Email,
		string(newUser.SubscriptionTier),
		h.config.JWTSecret,
		h.config.JWTExpirationHours,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "token_generation_error",
		})
	}

	return c.JSON(http.StatusCreated, models.AuthResponse{
		Token: token,
		User: &models.UserInfo{
			ID:                  newUser.ID,
			Email:               newUser.Email,
			Name:                newUser.Name,
			SubscriptionTier:    string(newUser.SubscriptionTier),
			UsageCount:          newUser.UsageCount,
			UsageLimit:          newUser.UsageLimit,
			EmailVerified:       newUser.EmailVerified,
			OnboardingCompleted: newUser.OnboardingCompleted,
			OnboardingStep:      newUser.OnboardingStep,
		},
	})
}

// Login godoc
// @Summary Login user
// @Description Authenticate user with email and password, returns JWT token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "Login credentials"
// @Success 200 {object} models.AuthResponse "Login successful"
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Invalid credentials"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /auth/login [post]
func (h *AuthHandler) Login(c echo.Context) error {
	var req models.LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		return errors.ValidationError(c, err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	// Find user by email
	u, err := h.db.User.Query().Where(user.EmailEQ(req.Email)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "invalid_credentials",
				Message: "Invalid email or password",
			})
		}
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "database_error",
		})
	}

	// Check password
	if !auth.CheckPassword(u.PasswordHash, req.Password) {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "invalid_credentials",
			Message: "Invalid email or password",
		})
	}

	// Update last login
	_, err = h.db.User.UpdateOneID(u.ID).
		SetLastLoginAt(time.Now()).
		Save(ctx)
	if err != nil {
		// Log error but don't fail the login
	}

	// Log login event
	ipAddress, userAgent := audit.GetRequestContext(c)
	go h.auditLogger.LogUserLogin(context.Background(), u.ID, ipAddress, userAgent)

	// Generate JWT
	token, err := auth.GenerateJWT(
		u.ID,
		u.Email,
		string(u.SubscriptionTier),
		h.config.JWTSecret,
		h.config.JWTExpirationHours,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "token_generation_error",
		})
	}

	return c.JSON(http.StatusOK, models.AuthResponse{
		Token: token,
		User: &models.UserInfo{
			ID:                  u.ID,
			Email:               u.Email,
			Name:                u.Name,
			SubscriptionTier:    string(u.SubscriptionTier),
			UsageCount:          u.UsageCount,
			UsageLimit:          u.UsageLimit,
			EmailVerified:       u.EmailVerified,
			OnboardingCompleted: u.OnboardingCompleted,
			OnboardingStep:      u.OnboardingStep,
		},
	})
}

// Me returns the current user's information
func (h *AuthHandler) Me(c echo.Context) error {
	// Get user ID from context (set by JWT middleware)
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "unauthorized",
		})
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	// Find user
	u, err := h.db.User.Get(ctx, userID)
	if err != nil {
		if ent.IsNotFound(err) {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error: "user_not_found",
			})
		}
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "database_error",
		})
	}

	return c.JSON(http.StatusOK, models.UserInfo{
		ID:                  u.ID,
		Email:               u.Email,
		Name:                u.Name,
		SubscriptionTier:    string(u.SubscriptionTier),
		UsageCount:          u.UsageCount,
		UsageLimit:          u.UsageLimit,
		EmailVerified:       u.EmailVerified,
		OnboardingCompleted: u.OnboardingCompleted,
		OnboardingStep:      u.OnboardingStep,
	})
}

// Logout revokes the current JWT token
func (h *AuthHandler) Logout(c echo.Context) error {
	// Get token from context (set by JWT middleware)
	token, ok := c.Get("token").(string)
	if !ok || token == "" {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "missing_token",
			Message: "No token found in request",
		})
	}

	// Get user ID for audit log
	userID, _ := c.Get("user_id").(int)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	// Add token to blacklist with TTL matching JWT expiration (24 hours)
	expiration := time.Duration(h.config.JWTExpirationHours) * time.Hour
	if err := h.blacklist.Add(ctx, token, expiration); err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "logout_error",
			Message: "Failed to revoke token",
		})
	}

	// Log logout event
	if userID > 0 {
		ipAddress, userAgent := audit.GetRequestContext(c)
		go h.auditLogger.LogUserLogout(context.Background(), userID, ipAddress, userAgent)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Successfully logged out",
	})
}

// VerifyEmail verifies user's email with token
func (h *AuthHandler) VerifyEmail(c echo.Context) error {
	token := c.Param("token")
	if token == "" {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "missing_token",
			Message: "Verification token is required",
		})
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	// Find user by verification token
	u, err := h.db.User.Query().
		Where(user.EmailVerificationTokenEQ(token)).
		Only(ctx)

	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_token",
			Message: "Invalid or expired verification token",
		})
	}

	// Check if token is expired
	if u.EmailVerificationTokenExpiresAt != nil && time.Now().After(*u.EmailVerificationTokenExpiresAt) {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "expired_token",
			Message: "Verification token has expired",
		})
	}

	// Check if already verified
	if u.EmailVerified {
		return c.JSON(http.StatusOK, map[string]string{
			"message": "Email already verified",
		})
	}

	// Update user as verified
	now := time.Now()
	_, err = h.db.User.UpdateOneID(u.ID).
		SetEmailVerified(true).
		SetEmailVerifiedAt(now).
		ClearEmailVerificationToken().
		ClearEmailVerificationTokenExpiresAt().
		Save(ctx)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "verification_failed",
		})
	}

	// Send welcome email (async)
	go h.emailService.SendWelcomeEmail(u.Email, u.Name)

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Email verified successfully",
	})
}

// ResendVerificationEmail resends verification email
func (h *AuthHandler) ResendVerificationEmail(c echo.Context) error {
	// Get user ID from context (must be authenticated)
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "unauthorized",
		})
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	// Get user
	u, err := h.db.User.Get(ctx, userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "user_not_found",
		})
	}

	// Check if already verified
	if u.EmailVerified {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "already_verified",
			Message: "Email is already verified",
		})
	}

	// Generate new verification token
	verificationToken, err := generateVerificationToken()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "token_generation_error",
		})
	}

	// Update user with new token
	_, err = h.db.User.UpdateOneID(userID).
		SetEmailVerificationToken(verificationToken).
		SetEmailVerificationTokenExpiresAt(time.Now().Add(24 * time.Hour)).
		Save(ctx)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "update_failed",
		})
	}

	// Send verification email (async)
	go h.emailService.SendVerificationEmail(u.Email, u.Name, verificationToken)

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Verification email sent",
	})
}

// ForgotPassword generates a password reset token and sends reset email
func (h *AuthHandler) ForgotPassword(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	var req struct {
		Email string `json:"email" validate:"required,email"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request format",
		})
	}

	if err := c.Validate(req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid email address",
		})
	}

	// Find user by email
	u, err := h.db.User.Query().
		Where(user.EmailEQ(req.Email)).
		Only(ctx)

	if err != nil {
		// For security, don't reveal if email exists or not
		// Return success even if user not found
		return c.JSON(http.StatusOK, map[string]string{
			"message": "If an account exists with this email, you will receive a password reset link",
		})
	}

	// Generate reset token
	resetToken, err := generatePasswordResetToken()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "token_generation_error",
			Message: "Failed to generate reset token",
		})
	}

	// Store token hash in Redis with 1-hour expiration
	tokenHash := sha256.Sum256([]byte(resetToken))
	tokenKey := fmt.Sprintf("password_reset:%s", hex.EncodeToString(tokenHash[:]))

	err = h.cache.Set(ctx, tokenKey, fmt.Sprintf("%d", u.ID), time.Hour)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "cache_error",
			Message: "Failed to store reset token",
		})
	}

	// Send password reset email (async)
	go h.emailService.SendPasswordResetEmail(u.Email, u.Name, resetToken)

	// Log password reset request
	ipAddress, userAgent := audit.GetRequestContext(c)
	go h.auditLogger.LogUserPasswordChange(context.Background(), u.ID, ipAddress, userAgent)

	return c.JSON(http.StatusOK, map[string]string{
		"message": "If an account exists with this email, you will receive a password reset link",
	})
}

// ResetPassword validates the reset token and updates the user's password
func (h *AuthHandler) ResetPassword(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	var req struct {
		Token       string `json:"token" validate:"required"`
		NewPassword string `json:"new_password" validate:"required,min=8"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request format",
		})
	}

	if err := c.Validate(req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "validation_error",
			Message: "Password must be at least 8 characters",
		})
	}

	// Hash token to look up in Redis
	tokenHash := sha256.Sum256([]byte(req.Token))
	tokenKey := fmt.Sprintf("password_reset:%s", hex.EncodeToString(tokenHash[:]))

	// Get user ID from Redis
	userIDStr, err := h.cache.Get(ctx, tokenKey)
	if err != nil || userIDStr == "" {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_token",
			Message: "Invalid or expired reset token",
		})
	}

	// Convert user ID to int
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "invalid_user_id",
			Message: "Invalid user ID in token",
		})
	}

	// Hash new password
	hashedPassword, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "hashing_error",
			Message: "Failed to hash password",
		})
	}

	// Update user password
	_, err = h.db.User.UpdateOneID(userID).
		SetPasswordHash(hashedPassword).
		Save(ctx)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "update_error",
			Message: "Failed to update password",
		})
	}

	// Delete token from Redis (one-time use)
	h.cache.Delete(ctx, tokenKey)

	// Log password reset success
	ipAddress, userAgent := audit.GetRequestContext(c)
	go h.auditLogger.LogUserPasswordChange(context.Background(), userID, ipAddress, userAgent)

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Password reset successfully",
	})
}

// generateVerificationToken generates a random token for email verification
func generateVerificationToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// generatePasswordResetToken generates a random token for password reset
func generatePasswordResetToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// OAuthLogin godoc
// @Summary Initiate OAuth login
// @Description Redirects to OAuth provider for authentication
// @Tags auth
// @Param provider path string true "OAuth Provider" Enums(google, github, microsoft)
// @Success 302 "Redirect to OAuth provider"
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /auth/oauth/{provider} [get]
func (h *AuthHandler) OAuthLogin(c echo.Context) error {
	if !h.config.FeatureSocialLogin {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "feature_disabled",
			Message: "Social login is disabled",
		})
	}

	provider := c.Param("provider")
	if provider == "" {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "missing_provider",
			Message: "Provider is required",
		})
	}

	// Generate random state token for CSRF protection
	state, err := generateStateToken()
	if err != nil {
		return errors.InternalError(c, err)
	}

	// Store state in session/cache with expiration (5 minutes)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = h.cache.Set(ctx, "oauth_state:"+state, provider, 5*time.Minute)
	if err != nil {
		return errors.InternalError(c, err)
	}

	// Get OAuth service
	oauthService := h.getOAuthService()

	// Get authorization URL
	authURL, err := oauthService.GetAuthURL(oauth.Provider(provider), state)
	if err != nil {
		if err == oauth.ErrInvalidProvider {
			return c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_provider",
				Message: "Invalid OAuth provider",
			})
		}
		return errors.InternalError(c, err)
	}

	return c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// OAuthCallback godoc
// @Summary Handle OAuth callback
// @Description Processes OAuth callback and creates/logs in user
// @Tags auth
// @Param provider path string true "OAuth Provider" Enums(google, github, microsoft)
// @Param code query string true "Authorization code"
// @Param state query string true "State token"
// @Success 302 "Redirect to frontend with JWT token"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /auth/oauth/callback/{provider} [get]
func (h *AuthHandler) OAuthCallback(c echo.Context) error {
	if !h.config.FeatureSocialLogin {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "feature_disabled",
			Message: "Social login is disabled",
		})
	}

	provider := c.Param("provider")
	code := c.QueryParam("code")
	state := c.QueryParam("state")

	if code == "" || state == "" {
		return c.Redirect(http.StatusTemporaryRedirect, h.config.FrontendURL+"/login?error=oauth_failed")
	}

	// Verify state token (CSRF protection)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	storedProvider, err := h.cache.Get(ctx, "oauth_state:"+state)
	if err != nil || storedProvider != provider {
		return c.Redirect(http.StatusTemporaryRedirect, h.config.FrontendURL+"/login?error=invalid_state")
	}

	// Delete state token (one-time use)
	h.cache.Delete(ctx, "oauth_state:"+state)

	// Get OAuth service
	oauthService := h.getOAuthService()

	// Exchange code for user info
	userInfo, err := oauthService.HandleCallback(ctx, oauth.Provider(provider), code)
	if err != nil {
		if err == oauth.ErrInvalidCode {
			return c.Redirect(http.StatusTemporaryRedirect, h.config.FrontendURL+"/login?error=invalid_code")
		}
		return c.Redirect(http.StatusTemporaryRedirect, h.config.FrontendURL+"/login?error=oauth_failed")
	}

	// Find or create user
	user, isNew, err := oauthService.FindOrCreateUser(ctx, userInfo)
	if err != nil {
		return c.Redirect(http.StatusTemporaryRedirect, h.config.FrontendURL+"/login?error=user_creation_failed")
	}

	// Generate JWT token
	token, err := auth.GenerateJWT(user.ID, user.Email, user.SubscriptionTier.String(), h.config.JWTSecret, h.config.JWTExpirationHours)
	if err != nil {
		return c.Redirect(http.StatusTemporaryRedirect, h.config.FrontendURL+"/login?error=token_generation_failed")
	}

	// Log successful OAuth login
	go func() {
		logCtx, logCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer logCancel()

		h.auditLogger.LogUserLogin(logCtx, user.ID, c.RealIP(), c.Request().UserAgent())
	}()

	// Redirect to frontend with token
	redirectURL := fmt.Sprintf("%s/auth/callback?token=%s&is_new=%t", h.config.FrontendURL, token, isNew)
	return c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

// Helper method to get OAuth service
func (h *AuthHandler) getOAuthService() *oauth.Service {
	return oauth.NewService(h.db, h.config)
}

// generateStateToken generates a random state token for OAuth CSRF protection
func generateStateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
