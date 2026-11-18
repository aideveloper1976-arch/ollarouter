package upstream

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratePKCE(t *testing.T) {
	p, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE() error = %v", err)
	}
	if p.Verifier == "" || p.Challenge == "" {
		t.Fatal("expected non-empty verifier and challenge")
	}
	if p.Verifier == p.Challenge {
		t.Fatal("challenge must differ from verifier (S256)")
	}
}

func TestAuthorizeURL(t *testing.T) {
	cfg := DefaultCodexOAuthConfig()
	u := cfg.AuthorizeURL("CHAL", "STATE")
	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatalf("authorize URL not parseable: %v", err)
	}
	q := parsed.Query()
	if q.Get("code_challenge") != "CHAL" || q.Get("code_challenge_method") != "S256" {
		t.Errorf("missing PKCE challenge params: %v", q)
	}
	if q.Get("state") != "STATE" {
		t.Errorf("state = %q", q.Get("state"))
	}
	if q.Get("client_id") != cfg.ClientID {
		t.Errorf("client_id = %q", q.Get("client_id"))
	}
}

func makeJWT(t *testing.T, claims map[string]interface{}) string {
	t.Helper()
	payload, _ := json.Marshal(claims)
	enc := base64.RawURLEncoding.EncodeToString
	return enc([]byte(`{"alg":"none"}`)) + "." + enc(payload) + "." + "sig"
}

func TestAccountIDFromJWT(t *testing.T) {
	tok := makeJWT(t, map[string]interface{}{
		"https://api.openai.com/auth": map[string]interface{}{"chatgpt_account_id": "acc-123"},
	})
	if got := accountIDFromJWT(tok); got != "acc-123" {
		t.Errorf("accountIDFromJWT = %q, want acc-123", got)
	}
	if got := accountIDFromJWT("not-a-jwt"); got != "" {
		t.Errorf("expected empty for non-jwt, got %q", got)
	}
}

func TestStartAndCompleteLogin(t *testing.T) {
	// Mock OAuth token endpoint.
	idToken := makeJWT(t, map[string]interface{}{
		"https://api.openai.com/auth": map[string]interface{}{"chatgpt_account_id": "acc-xyz"},
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.FormValue("grant_type") != "authorization_code" || r.FormValue("code") != "the-code" {
			w.WriteHeader(400)
			w.Write([]byte(`{"error":"invalid_grant"}`))
			return
		}
		if r.FormValue("code_verifier") == "" {
			w.WriteHeader(400)
			w.Write([]byte(`{"error":"missing_verifier"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"at-1","refresh_token":"rt-1","id_token":"` + idToken + `","expires_in":3600}`))
	}))
	defer srv.Close()

	store, err := NewEncryptedFileStore(filepath.Join(t.TempDir(), "cred.enc"), "0123456789abcdef0123")
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	m := NewUpstreamAuthManager(store, "")
	oc := DefaultCodexOAuthConfig()
	oc.TokenEndpoint = srv.URL
	m.SetOAuthConfig(oc)

	authURL, state, err := m.StartLogin("chatgpt")
	if err != nil {
		t.Fatalf("StartLogin error = %v", err)
	}
	if !strings.Contains(authURL, "state="+state) {
		t.Errorf("authURL %q missing state %q", authURL, state)
	}

	provider, err := m.CompleteLogin(context.Background(), state, "the-code")
	if err != nil {
		t.Fatalf("CompleteLogin error = %v", err)
	}
	if provider != "chatgpt" {
		t.Errorf("provider = %q, want chatgpt", provider)
	}

	// Status should be connected, and the resolved token + account id correct.
	if s := m.GetCredentialStatus("chatgpt"); s != "connected" {
		t.Errorf("status = %q, want connected", s)
	}
	tok, acc, err := m.UpstreamToken("chatgpt")
	if err != nil {
		t.Fatalf("UpstreamToken error = %v", err)
	}
	if tok != "at-1" || acc != "acc-xyz" {
		t.Errorf("UpstreamToken = (%q,%q), want (at-1, acc-xyz)", tok, acc)
	}

	// GetCredential should surface the access token (not the raw bundle JSON).
	if c, _ := m.GetCredential("chatgpt"); c != "at-1" {
		t.Errorf("GetCredential = %q, want at-1", c)
	}

	// Unknown state must be rejected.
	if _, err := m.CompleteLogin(context.Background(), "bogus", "the-code"); err == nil {
		t.Error("expected error for unknown state")
	}
}
