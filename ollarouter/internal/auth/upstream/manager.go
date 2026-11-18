package upstream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/aideveloper1976-arch/ollarouter/internal/config"
)

// AuthState represents the authentication state for a provider.
type AuthState string

const (
	AuthStateMissing   AuthState = "missing"
	AuthStateConnected AuthState = "connected"
	AuthStateExpired   AuthState = "expired"
)

// pendingLogin tracks an in-flight OAuth login (state -> verifier/provider).
type pendingLogin struct {
	provider string
	verifier string
	created  time.Time
}

// UpstreamAuthManager manages authentication state and credentials for upstream providers.
type UpstreamAuthManager struct {
	store      CredentialStore
	baseURL    string
	mu         sync.RWMutex
	states     map[string]AuthState
	oauth      CodexOAuthConfig
	pending    map[string]pendingLogin
	httpClient *http.Client
}

// NewUpstreamAuthManager creates a new UpstreamAuthManager.
func NewUpstreamAuthManager(store CredentialStore, baseURL string) *UpstreamAuthManager {
	return &UpstreamAuthManager{
		store:      store,
		baseURL:    baseURL,
		states:     make(map[string]AuthState),
		oauth:      DefaultCodexOAuthConfig(),
		pending:    make(map[string]pendingLogin),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// SetOAuthConfig overrides the default Codex OAuth configuration.
func (m *UpstreamAuthManager) SetOAuthConfig(cfg CodexOAuthConfig) {
	m.mu.Lock()
	m.oauth = cfg
	m.mu.Unlock()
}

// OAuthConfig returns the current Codex OAuth configuration.
func (m *UpstreamAuthManager) OAuthConfig() CodexOAuthConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.oauth
}

// GetCredentialStatus returns the current auth status for a provider.
// Returns one of: "missing", "connected", "expired".
func (m *UpstreamAuthManager) GetCredentialStatus(provider string) string {
	m.mu.RLock()
	state, known := m.states[provider]
	m.mu.RUnlock()

	if !known {
		// Check if credential exists in store
		raw, err := m.store.Get(provider)
		if err != nil {
			return string(AuthStateMissing)
		}
		// An OAuth bundle past expiry with no refresh token is "expired".
		if b, ok := tryParseBundle(raw); ok {
			if !b.Expiry.IsZero() && time.Now().After(b.Expiry) && b.RefreshToken == "" {
				return string(AuthStateExpired)
			}
		}
		// Found in store but not yet tracked — mark as connected
		m.mu.Lock()
		m.states[provider] = AuthStateConnected
		m.mu.Unlock()
		return string(AuthStateConnected)
	}
	return string(state)
}

// StoreCredential persists a credential for a provider and updates status.
func (m *UpstreamAuthManager) StoreCredential(provider, credential string) error {
	if provider == "" {
		return errors.New("provider name is required")
	}
	if credential == "" {
		return errors.New("credential is required")
	}

	if err := m.store.Set(provider, credential); err != nil {
		return fmt.Errorf("storing credential: %w", err)
	}

	m.mu.Lock()
	m.states[provider] = AuthStateConnected
	m.mu.Unlock()
	return nil
}

// GetCredential retrieves a credential for a provider. If the stored value is
// an OAuth token bundle, the access token is returned; otherwise the raw value
// (e.g. a legacy bearer token) is returned.
func (m *UpstreamAuthManager) GetCredential(provider string) (string, error) {
	raw, err := m.store.Get(provider)
	if err != nil {
		return "", err
	}
	if b, ok := tryParseBundle(raw); ok {
		return b.AccessToken, nil
	}
	return raw, nil
}

// tryParseBundle attempts to decode a stored credential as a TokenBundle.
func tryParseBundle(raw string) (*TokenBundle, bool) {
	var b TokenBundle
	if json.Unmarshal([]byte(raw), &b) == nil && b.AccessToken != "" {
		return &b, true
	}
	return nil, false
}

// RevokeCredential removes a credential for a provider and updates status.
func (m *UpstreamAuthManager) RevokeCredential(provider string) error {
	if err := m.store.Delete(provider); err != nil {
		return fmt.Errorf("revoking credential: %w", err)
	}

	m.mu.Lock()
	delete(m.states, provider)
	m.mu.Unlock()
	return nil
}

// BaseURL returns the configured base URL for the upstream service.
func (m *UpstreamAuthManager) BaseURL() string {
	return m.baseURL
}

// StartLogin begins an OAuth "Sign in with ChatGPT" flow for a provider.
// It returns the authorize URL the user must open and the state value that ties
// the eventual callback back to this attempt.
func (m *UpstreamAuthManager) StartLogin(provider string) (authURL, state string, err error) {
	if provider == "" {
		return "", "", errors.New("provider name is required")
	}
	pkce, err := GeneratePKCE()
	if err != nil {
		return "", "", fmt.Errorf("generating PKCE: %w", err)
	}
	state, err = randomState()
	if err != nil {
		return "", "", fmt.Errorf("generating state: %w", err)
	}

	m.mu.Lock()
	oauth := m.oauth
	m.pending[state] = pendingLogin{provider: provider, verifier: pkce.Verifier, created: time.Now()}
	// Evict stale pending logins (older than 15 minutes).
	for s, p := range m.pending {
		if time.Since(p.created) > 15*time.Minute {
			delete(m.pending, s)
		}
	}
	m.mu.Unlock()

	return oauth.AuthorizeURL(pkce.Challenge, state), state, nil
}

// CompleteLogin finishes an OAuth flow: it exchanges the authorization code for
// tokens (validating the state) and persists the resulting bundle.
func (m *UpstreamAuthManager) CompleteLogin(ctx context.Context, state, code string) (provider string, err error) {
	if state == "" || code == "" {
		return "", errors.New("state and code are required")
	}
	m.mu.Lock()
	pending, ok := m.pending[state]
	oauth := m.oauth
	httpClient := m.httpClient
	if ok {
		delete(m.pending, state)
	}
	m.mu.Unlock()
	if !ok {
		return "", errors.New("unknown or expired login state; start the login again")
	}

	bundle, err := oauth.Exchange(ctx, httpClient, code, pending.verifier)
	if err != nil {
		return "", err
	}
	if err := m.storeBundle(pending.provider, bundle); err != nil {
		return "", err
	}
	return pending.provider, nil
}

// storeBundle persists a token bundle (JSON-encoded) and marks the provider connected.
func (m *UpstreamAuthManager) storeBundle(provider string, bundle *TokenBundle) error {
	data, err := json.Marshal(bundle)
	if err != nil {
		return fmt.Errorf("encoding token bundle: %w", err)
	}
	if err := m.store.Set(provider, string(data)); err != nil {
		return fmt.Errorf("storing credential: %w", err)
	}
	m.mu.Lock()
	m.states[provider] = AuthStateConnected
	m.mu.Unlock()
	return nil
}

// UpstreamToken returns a currently-valid access token and the ChatGPT account
// id for a provider, refreshing the token bundle if it is near expiry. It
// satisfies the token-resolver interface expected by the ChatGPT adapter.
func (m *UpstreamAuthManager) UpstreamToken(provider string) (accessToken, accountID string, err error) {
	raw, err := m.store.Get(provider)
	if err != nil {
		return "", "", err
	}

	bundle, ok := tryParseBundle(raw)
	if !ok {
		// Legacy plain bearer token — return as-is with no account id.
		return raw, "", nil
	}

	// Refresh if expired (or within 60s of expiry) and we have a refresh token.
	needsRefresh := !bundle.Expiry.IsZero() && time.Now().After(bundle.Expiry.Add(-60*time.Second))
	if needsRefresh && bundle.RefreshToken != "" {
		m.mu.RLock()
		oauth := m.oauth
		httpClient := m.httpClient
		m.mu.RUnlock()

		refreshed, rerr := oauth.Refresh(context.Background(), httpClient, bundle.RefreshToken)
		if rerr != nil {
			m.mu.Lock()
			m.states[provider] = AuthStateExpired
			m.mu.Unlock()
			return "", "", fmt.Errorf("refreshing token: %w", rerr)
		}
		// Carry forward fields the refresh response may omit.
		if refreshed.RefreshToken == "" {
			refreshed.RefreshToken = bundle.RefreshToken
		}
		if refreshed.AccountID == "" {
			refreshed.AccountID = bundle.AccountID
		}
		if refreshed.IDToken == "" {
			refreshed.IDToken = bundle.IDToken
		}
		_ = m.storeBundle(provider, refreshed)
		bundle = refreshed
	}

	return bundle.AccessToken, bundle.AccountID, nil
}

// InitFromProviders creates an UpstreamAuthManager if any provider has auth settings configured.
// Returns nil, nil if no provider requires upstream auth.
// Returns an error if the credential store cannot be created (e.g. short encryption key).
func InitFromProviders(providers []config.ProviderConfig, credentialPath, encryptionKey string) (*UpstreamAuthManager, error) {
	hasAuth := false
	for _, p := range providers {
		if p.Auth != nil {
			hasAuth = true
			break
		}
	}
	if !hasAuth {
		return nil, nil
	}

	store, err := NewEncryptedFileStore(credentialPath, encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("creating upstream auth credential store: %w", err)
	}

	return NewUpstreamAuthManager(store, ""), nil
}
