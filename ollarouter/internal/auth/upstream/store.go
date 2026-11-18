package upstream

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
)

// CredentialStore defines the interface for credential storage.
type CredentialStore interface {
	// Get retrieves a credential by provider name.
	Get(provider string) (string, error)
	// Set stores a credential by provider name.
	Set(provider, credential string) error
	// Delete removes a credential by provider name.
	Delete(provider string) error
}

// EncryptedFileStore implements CredentialStore using an encrypted JSON file.
type EncryptedFileStore struct {
	path string
	key  []byte
	mu   sync.RWMutex
}

// NewEncryptedFileStore creates a new EncryptedFileStore.
// encryptionKey must be at least 16 bytes long.
func NewEncryptedFileStore(path string, encryptionKey string) (*EncryptedFileStore, error) {
	if len(encryptionKey) < 16 {
		return nil, fmt.Errorf("encryption key must be at least 16 bytes, got %d", len(encryptionKey))
	}
	return &EncryptedFileStore{
		path: path,
		key:  []byte(encryptionKey),
	}, nil
}

// Get retrieves a credential by provider name from the encrypted file.
func (s *EncryptedFileStore) Get(provider string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.key) == 0 {
		return "", errors.New("encryption key is empty")
	}

	data, err := s.readFile()
	if err != nil {
		return "", fmt.Errorf("reading credential store: %w", err)
	}

	val, ok := data[provider]
	if !ok {
		return "", fmt.Errorf("credential not found for provider %q", provider)
	}
	return val, nil
}

// Set stores a credential by provider name in the encrypted file.
func (s *EncryptedFileStore) Set(provider, credential string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.key) == 0 {
		return errors.New("encryption key is empty")
	}

	data, err := s.readFile()
	if err != nil {
		// If file doesn't exist yet, start fresh
		data = make(map[string]string)
	}

	data[provider] = credential
	return s.writeFile(data)
}

// Delete removes a credential by provider name from the encrypted file.
func (s *EncryptedFileStore) Delete(provider string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.key) == 0 {
		return errors.New("encryption key is empty")
	}

	data, err := s.readFile()
	if err != nil {
		// If file doesn't exist, nothing to delete
		return nil
	}

	delete(data, provider)
	return s.writeFile(data)
}

// readFile reads and decrypts the entire credential file.
func (s *EncryptedFileStore) readFile() (map[string]string, error) {
	ciphertext, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, err
	}

	if len(ciphertext) == 0 {
		return make(map[string]string), nil
	}

	plaintext, err := decrypt(s.key, ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	var data map[string]string
	if err := json.Unmarshal(plaintext, &data); err != nil {
		return nil, fmt.Errorf("decoding credentials: %w", err)
	}
	return data, nil
}

// writeFile encrypts and writes the credential data to file.
func (s *EncryptedFileStore) writeFile(data map[string]string) error {
	plaintext, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("encoding credentials: %w", err)
	}

	ciphertext, err := encrypt(s.key, plaintext)
	if err != nil {
		return fmt.Errorf("encryption failed: %w", err)
	}

	return os.WriteFile(s.path, ciphertext, 0600)
}

// encrypt encrypts plaintext using AES-256-GCM with the provided key.
func encrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(padKey(key))
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// decrypt decrypts ciphertext using AES-256-GCM with the provided key.
func decrypt(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(padKey(key))
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// padKey ensures the key is exactly 32 bytes for AES-256.
func padKey(key []byte) []byte {
	if len(key) >= 32 {
		return key[:32]
	}
	padded := make([]byte, 32)
	copy(padded, key)
	return padded
}
