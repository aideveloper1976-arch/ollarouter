package upstream

import (
	"testing"

	"github.com/aideveloper1976-arch/ollarouter/internal/config"
)

func mustNewTestStore(t *testing.T) *EncryptedFileStore {
	t.Helper()
	store, err := NewEncryptedFileStore("/tmp/test-creds.enc", "test-key-32bytes!!!")
	if err != nil {
		t.Fatalf("NewEncryptedFileStore() error = %v", err)
	}
	return store
}

func TestNewUpstreamAuthManager(t *testing.T) {
	store := mustNewTestStore(t)
	mgr := NewUpstreamAuthManager(store, "https://chatgpt.com")

	if mgr == nil {
		t.Fatal("NewUpstreamAuthManager() returned nil")
	}
}

func TestManagerGetCredentialStatusMissing(t *testing.T) {
	store := mustNewTestStore(t)
	mgr := NewUpstreamAuthManager(store, "https://chatgpt.com")

	status := mgr.GetCredentialStatus("nonexistent-provider")
	if status != "missing" {
		t.Errorf("GetCredentialStatus() = %q, want %q", status, "missing")
	}
}

func TestManagerGetCredentialStatusConnected(t *testing.T) {
	store := mustNewTestStore(t)
	mgr := NewUpstreamAuthManager(store, "https://chatgpt.com")

	_ = mgr.StoreCredential("test-provider", "bearer-token-valid")
	status := mgr.GetCredentialStatus("test-provider")
	if status != "connected" {
		t.Errorf("GetCredentialStatus() = %q, want %q", status, "connected")
	}
}

func TestManagerStoreAndGetCredential(t *testing.T) {
	store := mustNewTestStore(t)
	mgr := NewUpstreamAuthManager(store, "https://chatgpt.com")

	err := mgr.StoreCredential("test-provider", "bearer-token-789")
	if err != nil {
		t.Fatalf("StoreCredential() error = %v", err)
	}

	token, err := mgr.GetCredential("test-provider")
	if err != nil {
		t.Fatalf("GetCredential() error = %v", err)
	}
	if token != "bearer-token-789" {
		t.Errorf("GetCredential() = %q, want bearer-token-789", token)
	}
}

func TestManagerRevokeCredential(t *testing.T) {
	store := mustNewTestStore(t)
	mgr := NewUpstreamAuthManager(store, "https://chatgpt.com")

	_ = mgr.StoreCredential("test-provider", "bearer-token-abc")
	err := mgr.RevokeCredential("test-provider")
	if err != nil {
		t.Fatalf("RevokeCredential() error = %v", err)
	}

	status := mgr.GetCredentialStatus("test-provider")
	if status != "missing" {
		t.Errorf("GetCredentialStatus() after revoke = %q, want %q", status, "missing")
	}
}

func TestManagerHealthStates(t *testing.T) {
	store := mustNewTestStore(t)
	mgr := NewUpstreamAuthManager(store, "https://chatgpt.com")

	// Initially missing
	if s := mgr.GetCredentialStatus("health-test"); s != "missing" {
		t.Errorf("initial status = %q, want missing", s)
	}

	// After storing - connected
	_ = mgr.StoreCredential("health-test", "valid-token")
	if s := mgr.GetCredentialStatus("health-test"); s != "connected" {
		t.Errorf("after store status = %q, want connected", s)
	}

	// After revoke - missing again
	_ = mgr.RevokeCredential("health-test")
	if s := mgr.GetCredentialStatus("health-test"); s != "missing" {
		t.Errorf("after revoke status = %q, want missing", s)
	}
}

func TestManagerGetCredentialNeverStored(t *testing.T) {
	store := mustNewTestStore(t)
	mgr := NewUpstreamAuthManager(store, "https://chatgpt.com")

	_, err := mgr.GetCredential("never-stored")
	if err == nil {
		t.Fatal("Expected error for never-stored credential, got nil")
	}
}

func TestInitFromProvidersWithAuthConfigured(t *testing.T) {
	providers := []config.ProviderConfig{
		{
			Name:    "chatgpt-plus",
			Type:    "chatgpt-plus",
			BaseURL: "https://chatgpt.com",
			Auth: &config.AuthSettings{
				Mode: "session_cookie",
			},
		},
	}

	dir := t.TempDir()
	mgr, err := InitFromProviders(providers, dir+"/creds.enc", "test-encryption-key-32bytes!")
	if err != nil {
		t.Fatalf("InitFromProviders() error = %v", err)
	}
	if mgr == nil {
		t.Fatal("InitFromProviders() returned nil, expected non-nil manager when providers have auth settings")
	}
}

func TestInitFromProvidersWithoutAuthReturnsNil(t *testing.T) {
	providers := []config.ProviderConfig{
		{
			Name:    "openai",
			Type:    "openai",
			BaseURL: "https://api.openai.com/v1",
		},
	}

	mgr, err := InitFromProviders(providers, "/tmp/creds.enc", "test-encryption-key-32bytes!")
	if err != nil {
		t.Fatalf("InitFromProviders() error = %v", err)
	}
	if mgr != nil {
		t.Fatal("InitFromProviders() returned non-nil, expected nil when no providers have auth settings")
	}
}

func TestInitFromProvidersEmptyProvidersReturnsNil(t *testing.T) {
	mgr, err := InitFromProviders([]config.ProviderConfig{}, "/tmp/creds.enc", "test-encryption-key-32bytes!")
	if err != nil {
		t.Fatalf("InitFromProviders() error = %v", err)
	}
	if mgr != nil {
		t.Fatal("InitFromProviders() returned non-nil, expected nil for empty providers list")
	}
}

func TestInitFromProvidersShortKeyReturnsError(t *testing.T) {
	providers := []config.ProviderConfig{
		{
			Name: "chatgpt-plus",
			Type: "chatgpt-plus",
			Auth: &config.AuthSettings{Mode: "session_cookie"},
		},
	}

	mgr, err := InitFromProviders(providers, "/tmp/creds.enc", "short")
	if err == nil {
		t.Fatal("InitFromProviders() with short encryption key should return error")
	}
	if mgr != nil {
		t.Fatal("InitFromProviders() with short encryption key should return nil manager")
	}
}

func TestInitFromProvidersCredentialsPersistAcrossManagers(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/creds.enc"
	providers := []config.ProviderConfig{
		{
			Name: "chatgpt-plus",
			Type: "chatgpt-plus",
			Auth: &config.AuthSettings{Mode: "session_cookie"},
		},
	}

	mgr1, err := InitFromProviders(providers, path, "test-encryption-key-32bytes!")
	if err != nil {
		t.Fatalf("InitFromProviders() 1 error = %v", err)
	}
	if mgr1 == nil {
		t.Fatal("InitFromProviders() 1 returned nil")
	}

	err = mgr1.StoreCredential("chatgpt-plus", "bearer-token-persisted")
	if err != nil {
		t.Fatalf("StoreCredential() error = %v", err)
	}

	mgr2, err := InitFromProviders(providers, path, "test-encryption-key-32bytes!")
	if err != nil {
		t.Fatalf("InitFromProviders() 2 error = %v", err)
	}
	if mgr2 == nil {
		t.Fatal("InitFromProviders() 2 returned nil")
	}

	token, err := mgr2.GetCredential("chatgpt-plus")
	if err != nil {
		t.Fatalf("GetCredential() after re-init error = %v", err)
	}
	if token != "bearer-token-persisted" {
		t.Errorf("GetCredential() = %q, want %q", token, "bearer-token-persisted")
	}
}
