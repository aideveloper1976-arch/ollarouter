package upstream

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// CodexOAuthConfig holds the parameters for the "Sign in with ChatGPT" OAuth
// (PKCE) flow and the Codex backend that a ChatGPT Plus/Pro subscription is
// billed through. The defaults match OpenAI's public Codex CLI client; they can
// be overridden per-provider via AuthSettings if OpenAI changes them.
type CodexOAuthConfig struct {
	ClientID      string
	AuthEndpoint  string // OAuth authorize URL
	TokenEndpoint string // OAuth token URL
	RedirectURI   string // must be a redirect URI registered for ClientID
	Scopes        string
	ResponsesURL  string // Codex Responses API endpoint
}

// DefaultCodexOAuthConfig returns the public Codex CLI OAuth configuration.
func DefaultCodexOAuthConfig() CodexOAuthConfig {
	return CodexOAuthConfig{
		ClientID:      "app_EMoamEEZ73f0CkXaXp7hrann",
		AuthEndpoint:  "https://auth.openai.com/oauth/authorize",
		TokenEndpoint: "https://auth.openai.com/oauth/token",
		RedirectURI:   "http://localhost:1455/auth/callback",
		Scopes:        "openid profile email offline_access",
		ResponsesURL:  "https://chatgpt.com/backend-api/codex/responses",
	}
}

// withOverrides applies non-empty AuthSettings-derived overrides.
func (c CodexOAuthConfig) withOverrides(clientID, authEndpoint, tokenEndpoint, redirectURI, scopes string) CodexOAuthConfig {
	if clientID != "" {
		c.ClientID = clientID
	}
	if authEndpoint != "" {
		c.AuthEndpoint = authEndpoint
	}
	if tokenEndpoint != "" {
		c.TokenEndpoint = tokenEndpoint
	}
	if redirectURI != "" {
		c.RedirectURI = redirectURI
	}
	if scopes != "" {
		c.Scopes = scopes
	}
	return c
}

// PKCE holds a generated PKCE verifier/challenge pair.
type PKCE struct {
	Verifier  string
	Challenge string
}

// GeneratePKCE creates a new S256 PKCE pair.
func GeneratePKCE() (PKCE, error) {
	b := make([]byte, 64)
	if _, err := rand.Read(b); err != nil {
		return PKCE{}, err
	}
	verifier := base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(verifier))
	return PKCE{
		Verifier:  verifier,
		Challenge: base64.RawURLEncoding.EncodeToString(sum[:]),
	}, nil
}

// randomState returns a URL-safe random state string.
func randomState() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// AuthorizeURL builds the OAuth authorize URL for the given challenge and state.
func (c CodexOAuthConfig) AuthorizeURL(challenge, state string) string {
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", c.ClientID)
	q.Set("redirect_uri", c.RedirectURI)
	q.Set("scope", c.Scopes)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	q.Set("id_token_add_organizations", "true")
	q.Set("codex_cli_simplified_flow", "true")
	q.Set("state", state)
	return c.AuthEndpoint + "?" + q.Encode()
}

// TokenBundle is the persisted set of credentials for a subscription provider.
type TokenBundle struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	IDToken      string    `json:"id_token"`
	AccountID    string    `json:"account_id"`
	Expiry       time.Time `json:"expiry"`
}

// tokenResponse is the OAuth token endpoint response.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

// Exchange swaps an authorization code for a token bundle.
func (c CodexOAuthConfig) Exchange(ctx context.Context, httpClient *http.Client, code, verifier string) (*TokenBundle, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", c.ClientID)
	form.Set("code", code)
	form.Set("redirect_uri", c.RedirectURI)
	form.Set("code_verifier", verifier)
	return c.postToken(ctx, httpClient, form)
}

// Refresh obtains a fresh token bundle from a refresh token.
func (c CodexOAuthConfig) Refresh(ctx context.Context, httpClient *http.Client, refreshToken string) (*TokenBundle, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", c.ClientID)
	form.Set("refresh_token", refreshToken)
	form.Set("scope", c.Scopes)
	return c.postToken(ctx, httpClient, form)
}

func (c CodexOAuthConfig) postToken(ctx context.Context, httpClient *http.Client, form url.Values) (*TokenBundle, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("decoding token response (HTTP %d): %w", resp.StatusCode, err)
	}
	if resp.StatusCode != 200 || tr.Error != "" {
		msg := tr.ErrorDesc
		if msg == "" {
			msg = tr.Error
		}
		if msg == "" {
			msg = string(body)
		}
		return nil, fmt.Errorf("oauth token error (HTTP %d): %s", resp.StatusCode, msg)
	}

	bundle := &TokenBundle{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		IDToken:      tr.IDToken,
	}
	if tr.ExpiresIn > 0 {
		bundle.Expiry = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	}
	// The ChatGPT account id is needed for the Codex backend "chatgpt-account-id"
	// header. It is carried in the id_token (preferred) or the access_token claims.
	bundle.AccountID = accountIDFromJWT(tr.IDToken)
	if bundle.AccountID == "" {
		bundle.AccountID = accountIDFromJWT(tr.AccessToken)
	}
	return bundle, nil
}

// accountIDFromJWT extracts the ChatGPT account id from a JWT's claims.
// It looks for the OpenAI auth claim "https://api.openai.com/auth".chatgpt_account_id.
func accountIDFromJWT(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		// Some encoders include padding.
		payload, err = base64.URLEncoding.DecodeString(parts[1])
		if err != nil {
			return ""
		}
	}
	var claims map[string]interface{}
	if json.Unmarshal(payload, &claims) != nil {
		return ""
	}
	if authClaim, ok := claims["https://api.openai.com/auth"].(map[string]interface{}); ok {
		if id, ok := authClaim["chatgpt_account_id"].(string); ok {
			return id
		}
		if id, ok := authClaim["account_id"].(string); ok {
			return id
		}
	}
	if id, ok := claims["chatgpt_account_id"].(string); ok {
		return id
	}
	return ""
}
