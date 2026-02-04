package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/jordanlanch/industrydb/config"
	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/user"
)

var (
	// ErrInvalidProvider is returned when an unsupported provider is specified
	ErrInvalidProvider = errors.New("invalid OAuth provider")
	// ErrInvalidCode is returned when the authorization code is invalid
	ErrInvalidCode = errors.New("invalid authorization code")
	// ErrProviderAPIError is returned when the provider API returns an error
	ErrProviderAPIError = errors.New("OAuth provider API error")
)

// Provider represents an OAuth provider
type Provider string

const (
	// ProviderGoogle represents Google OAuth
	ProviderGoogle Provider = "google"
	// ProviderGitHub represents GitHub OAuth
	ProviderGitHub Provider = "github"
	// ProviderMicrosoft represents Microsoft OAuth
	ProviderMicrosoft Provider = "microsoft"
)

// UserInfo holds basic user information from OAuth providers
type UserInfo struct {
	ID            string
	Email         string
	Name          string
	ProfilePicURL string
	Provider      Provider
}

// Service handles OAuth operations
type Service struct {
	db     *ent.Client
	config *config.Config
	client *http.Client
}

// NewService creates a new OAuth service
func NewService(db *ent.Client, cfg *config.Config) *Service {
	return &Service{
		db:     db,
		config: cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetAuthURL returns the OAuth authorization URL for a provider
func (s *Service) GetAuthURL(provider Provider, state string) (string, error) {
	switch provider {
	case ProviderGoogle:
		return s.getGoogleAuthURL(state), nil
	case ProviderGitHub:
		return s.getGitHubAuthURL(state), nil
	case ProviderMicrosoft:
		return s.getMicrosoftAuthURL(state), nil
	default:
		return "", ErrInvalidProvider
	}
}

// HandleCallback processes the OAuth callback and returns user info
func (s *Service) HandleCallback(ctx context.Context, provider Provider, code string) (*UserInfo, error) {
	switch provider {
	case ProviderGoogle:
		return s.handleGoogleCallback(ctx, code)
	case ProviderGitHub:
		return s.handleGitHubCallback(ctx, code)
	case ProviderMicrosoft:
		return s.handleMicrosoftCallback(ctx, code)
	default:
		return nil, ErrInvalidProvider
	}
}

// FindOrCreateUser finds an existing user by OAuth ID or creates a new one
func (s *Service) FindOrCreateUser(ctx context.Context, userInfo *UserInfo) (*ent.User, bool, error) {
	// Try to find existing user by OAuth provider and ID
	existingUser, err := s.db.User.
		Query().
		Where(
			user.OauthProviderEQ(string(userInfo.Provider)),
			user.OauthIDEQ(userInfo.ID),
		).
		Only(ctx)

	if err == nil {
		// User exists with this OAuth account
		return existingUser, false, nil
	}

	if !ent.IsNotFound(err) {
		return nil, false, fmt.Errorf("failed to query user: %w", err)
	}

	// Check if user exists with the same email
	existingUser, err = s.db.User.
		Query().
		Where(user.EmailEQ(userInfo.Email)).
		Only(ctx)

	if err == nil {
		// Link OAuth account to existing user
		existingUser, err = existingUser.Update().
			SetOauthProvider(string(userInfo.Provider)).
			SetOauthID(userInfo.ID).
			Save(ctx)
		if err != nil {
			return nil, false, fmt.Errorf("failed to link OAuth account: %w", err)
		}
		return existingUser, false, nil
	}

	if !ent.IsNotFound(err) {
		return nil, false, fmt.Errorf("failed to query user by email: %w", err)
	}

	// Create new user
	newUser, err := s.db.User.
		Create().
		SetEmail(userInfo.Email).
		SetName(userInfo.Name).
		SetPasswordHash("oauth-user-no-password"). // OAuth users don't have password
		SetOauthProvider(string(userInfo.Provider)).
		SetOauthID(userInfo.ID).
		SetEmailVerified(true). // OAuth emails are pre-verified
		SetEmailVerifiedAt(time.Now()).
		SetAcceptedTermsAt(time.Now()). // Auto-accept terms for OAuth
		Save(ctx)

	if err != nil {
		return nil, false, fmt.Errorf("failed to create user: %w", err)
	}

	return newUser, true, nil
}

// Google OAuth implementation
func (s *Service) getGoogleAuthURL(state string) string {
	baseURL := "https://accounts.google.com/o/oauth2/v2/auth"
	params := url.Values{}
	params.Add("client_id", s.config.GoogleClientID)
	params.Add("redirect_uri", s.config.OAuthCallbackURL+"/google")
	params.Add("response_type", "code")
	params.Add("scope", "openid email profile")
	params.Add("state", state)
	return baseURL + "?" + params.Encode()
}

func (s *Service) handleGoogleCallback(ctx context.Context, code string) (*UserInfo, error) {
	// Exchange code for token
	tokenURL := "https://oauth2.googleapis.com/token"
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", s.config.GoogleClientID)
	data.Set("client_secret", s.config.GoogleClientSecret)
	data.Set("redirect_uri", s.config.OAuthCallbackURL+"/google")
	data.Set("grant_type", "authorization_code")

	resp, err := s.client.PostForm(tokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrInvalidCode
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	// Get user info
	userInfoURL := "https://www.googleapis.com/oauth2/v2/userinfo"
	req, err := http.NewRequestWithContext(ctx, "GET", userInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)

	resp, err = s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrProviderAPIError
	}

	var googleUser struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &UserInfo{
		ID:            googleUser.ID,
		Email:         googleUser.Email,
		Name:          googleUser.Name,
		ProfilePicURL: googleUser.Picture,
		Provider:      ProviderGoogle,
	}, nil
}

// GitHub OAuth implementation
func (s *Service) getGitHubAuthURL(state string) string {
	baseURL := "https://github.com/login/oauth/authorize"
	params := url.Values{}
	params.Add("client_id", s.config.GitHubClientID)
	params.Add("redirect_uri", s.config.OAuthCallbackURL+"/github")
	params.Add("scope", "user:email")
	params.Add("state", state)
	return baseURL + "?" + params.Encode()
}

func (s *Service) handleGitHubCallback(ctx context.Context, code string) (*UserInfo, error) {
	// Exchange code for token
	tokenURL := "https://github.com/login/oauth/access_token"
	data := url.Values{}
	data.Set("client_id", s.config.GitHubClientID)
	data.Set("client_secret", s.config.GitHubClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", s.config.OAuthCallbackURL+"/github")

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.URL.RawQuery = data.Encode()

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrInvalidCode
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	// Get user info
	userInfoURL := "https://api.github.com/user"
	req, err = http.NewRequestWithContext(ctx, "GET", userInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)

	resp, err = s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: status %d: %s", ErrProviderAPIError, resp.StatusCode, string(body))
	}

	var githubUser struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&githubUser); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// If email is not public, fetch from emails endpoint
	email := githubUser.Email
	if email == "" {
		email, err = s.getGitHubPrimaryEmail(ctx, tokenResp.AccessToken)
		if err != nil {
			return nil, err
		}
	}

	name := githubUser.Name
	if name == "" {
		name = githubUser.Login
	}

	return &UserInfo{
		ID:            fmt.Sprintf("%d", githubUser.ID),
		Email:         email,
		Name:          name,
		ProfilePicURL: githubUser.AvatarURL,
		Provider:      ProviderGitHub,
	}, nil
}

func (s *Service) getGitHubPrimaryEmail(ctx context.Context, accessToken string) (string, error) {
	emailsURL := "https://api.github.com/user/emails"
	req, err := http.NewRequestWithContext(ctx, "GET", emailsURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get emails: %w", err)
	}
	defer resp.Body.Close()

	var emails []struct {
		Email   string `json:"email"`
		Primary bool   `json:"primary"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", fmt.Errorf("failed to decode emails: %w", err)
	}

	for _, e := range emails {
		if e.Primary {
			return e.Email, nil
		}
	}

	if len(emails) > 0 {
		return emails[0].Email, nil
	}

	return "", errors.New("no email found for GitHub user")
}

// Microsoft OAuth implementation
func (s *Service) getMicrosoftAuthURL(state string) string {
	baseURL := "https://login.microsoftonline.com/common/oauth2/v2.0/authorize"
	params := url.Values{}
	params.Add("client_id", s.config.MicrosoftClientID)
	params.Add("redirect_uri", s.config.OAuthCallbackURL+"/microsoft")
	params.Add("response_type", "code")
	params.Add("scope", "openid email profile")
	params.Add("state", state)
	return baseURL + "?" + params.Encode()
}

func (s *Service) handleMicrosoftCallback(ctx context.Context, code string) (*UserInfo, error) {
	// Exchange code for token
	tokenURL := "https://login.microsoftonline.com/common/oauth2/v2.0/token"
	data := url.Values{}
	data.Set("client_id", s.config.MicrosoftClientID)
	data.Set("client_secret", s.config.MicrosoftClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", s.config.OAuthCallbackURL+"/microsoft")
	data.Set("grant_type", "authorization_code")

	resp, err := s.client.PostForm(tokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrInvalidCode
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	// Get user info
	userInfoURL := "https://graph.microsoft.com/v1.0/me"
	req, err := http.NewRequestWithContext(ctx, "GET", userInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)

	resp, err = s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrProviderAPIError
	}

	var msUser struct {
		ID                string `json:"id"`
		UserPrincipalName string `json:"userPrincipalName"`
		DisplayName       string `json:"displayName"`
		Mail              string `json:"mail"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&msUser); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	email := msUser.Mail
	if email == "" {
		email = msUser.UserPrincipalName
	}

	return &UserInfo{
		ID:            msUser.ID,
		Email:         email,
		Name:          msUser.DisplayName,
		ProfilePicURL: "", // Microsoft Graph API requires separate call for photo
		Provider:      ProviderMicrosoft,
	}, nil
}
