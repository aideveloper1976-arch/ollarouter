package upstream

import (
	"os"
	"path/filepath"
	"testing"
)

func mustNewStore(t *testing.T, path, key string) *EncryptedFileStore {
	t.Helper()
	store, err := NewEncryptedFileStore(path, key)
	if err != nil {
		t.Fatalf("NewEncryptedFileStore() error = %v", err)
	}
	return store
}

func TestEncryptedFileStoreSetAndGet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.enc")
	store := mustNewStore(t, path, "test-encryption-key-32bytes!")

	err := store.Set("test-provider", "bearer-token-123")
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	got, err := store.Get("test-provider")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != "bearer-token-123" {
		t.Errorf("Get() = %q, want bearer-token-123", got)
	}
}

func TestEncryptedFileStoreGetMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.enc")
	store := mustNewStore(t, path, "test-encryption-key-32bytes!")

	_, err := store.Get("nonexistent")
	if err == nil {
		t.Fatal("Expected error for missing credential, got nil")
	}
}

func TestEncryptedFileStoreDelete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.enc")
	store := mustNewStore(t, path, "test-encryption-key-32bytes!")

	_ = store.Set("test-provider", "bearer-token-456")
	err := store.Delete("test-provider")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = store.Get("test-provider")
	if err == nil {
		t.Fatal("Expected error after Delete, got nil")
	}
}

func TestEncryptedFileStoreDeleteNonexistent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.enc")
	store := mustNewStore(t, path, "test-encryption-key-32bytes!")

	err := store.Delete("never-set")
	if err != nil {
		t.Fatalf("Delete() for nonexistent key should not error, got %v", err)
	}
}

func TestEncryptedFileStoreSetOverwrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.enc")
	store := mustNewStore(t, path, "test-encryption-key-32bytes!")

	_ = store.Set("test-provider", "first-token")
	_ = store.Set("test-provider", "second-token")

	got, err := store.Get("test-provider")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != "second-token" {
		t.Errorf("Get() after overwrite = %q, want second-token", got)
	}
}

func TestEncryptedFileStoreFilePersistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.enc")

	// Create store, set value
	s1 := mustNewStore(t, path, "test-encryption-key-32bytes!")
	_ = s1.Set("persist-test", "token-persists")

	// Re-open with same key, should read back
	s2 := mustNewStore(t, path, "test-encryption-key-32bytes!")
	got, err := s2.Get("persist-test")
	if err != nil {
		t.Fatalf("Get() from reopened store error = %v", err)
	}
	if got != "token-persists" {
		t.Errorf("Get() from reopened store = %q, want token-persists", got)
	}

	// Clean up
	os.Remove(path)
}

func TestEncryptedFileStoreDifferentKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.enc")

	store1 := mustNewStore(t, path, "key-one-1234567890123456")
	_ = store1.Set("test", "secret-value")

	store2 := mustNewStore(t, path, "key-two-9876543210987654")
	_, err := store2.Get("test")
	if err == nil {
		t.Error("Expected error when reading with different key, got nil")
	}
}

func TestEncryptedFileStoreEmptyKeyRejected(t *testing.T) {
	_, err := NewEncryptedFileStore("/tmp/test.enc", "")
	if err == nil {
		t.Error("Expected error for empty encryption key, got nil")
	}
}

func TestEncryptedFileStoreShortKeyRejected(t *testing.T) {
	store, err := NewEncryptedFileStore("/tmp/test.enc", "short")
	if err == nil {
		t.Error("NewEncryptedFileStore() with short key should return error, got nil")
	}
	if store != nil {
		t.Error("NewEncryptedFileStore() with short key should return nil store")
	}
}
