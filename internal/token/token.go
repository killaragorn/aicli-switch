package token

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	ClientID = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
	TokenURL = "https://platform.claude.com/v1/oauth/token"

	// Refresh if token expires within this window
	RefreshWindow = 5 * time.Minute
)

// OAuthData matches the claudeAiOauth structure in ~/.claude/.credentials.json
type OAuthData struct {
	AccessToken      string   `json:"accessToken"`
	RefreshToken     string   `json:"refreshToken"`
	ExpiresAt        int64    `json:"expiresAt"` // milliseconds since epoch
	Scopes           []string `json:"scopes,omitempty"`
	SubscriptionType string   `json:"subscriptionType,omitempty"`
	RateLimitTier    string   `json:"rateLimitTier,omitempty"`
}

// CredentialsFile represents the full ~/.claude/.credentials.json structure
type CredentialsFile struct {
	ClaudeAiOauth json.RawMessage `json:"claudeAiOauth,omitempty"`
	McpOAuth      json.RawMessage `json:"mcpOAuth,omitempty"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// GetExpiryFromData returns the expiry time from OAuthData.
func GetExpiryFromData(data *OAuthData) time.Time {
	if data.ExpiresAt == 0 {
		return time.Time{}
	}
	return time.UnixMilli(data.ExpiresAt)
}

// IsExpiredData checks if OAuth data is expired or will expire within RefreshWindow.
func IsExpiredData(data *OAuthData) bool {
	exp := GetExpiryFromData(data)
	if exp.IsZero() {
		return true
	}
	return time.Until(exp) < RefreshWindow
}

// ParseJWTPayload decodes the payload of a JWT without verification.
func ParseJWTPayload(tokenStr string) (map[string]any, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT: expected 3 parts, got %d", len(parts))
	}

	payload, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}

	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("parse claims: %w", err)
	}
	return claims, nil
}

// GetEmail extracts the email from a JWT access token.
func GetEmail(accessToken string) string {
	claims, err := ParseJWTPayload(accessToken)
	if err != nil {
		return ""
	}
	if email, ok := claims["email"].(string); ok {
		return email
	}
	return ""
}

// RefreshOAuthToken exchanges a refresh token for a new access token.
func RefreshOAuthToken(refreshToken string) (*TokenResponse, error) {
	body := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
		"client_id":     ClientID,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", TokenURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "claude-cli/2.1.81 (external, cli)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh failed (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &tokenResp, nil
}

func base64URLDecode(s string) ([]byte, error) {
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.URLEncoding.DecodeString(s)
}
