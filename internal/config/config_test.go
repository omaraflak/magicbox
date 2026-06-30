package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrGenerateJWTSecret_CreatesSecret(t *testing.T) {
	tempDir := t.TempDir()
	secretPath := filepath.Join(tempDir, "jwt_secret")

	// Verify creation
	secret1, err := loadOrGenerateJWTSecret(secretPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(secret1) != 32 {
		t.Errorf("expected 32-byte secret, got %d bytes", len(secret1))
	}

	// Verify the file was written
	if _, err := os.Stat(secretPath); os.IsNotExist(err) {
		t.Fatalf("expected secret file to exist")
	}
}

func TestLoadOrGenerateJWTSecret_ReadsExistingSecret(t *testing.T) {
	tempDir := t.TempDir()
	secretPath := filepath.Join(tempDir, "jwt_secret")

	secret1, _ := loadOrGenerateJWTSecret(secretPath)

	// Call again, should read existing
	secret2, err := loadOrGenerateJWTSecret(secretPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Equal(secret1, secret2) {
		t.Errorf("expected secrets to be identical, got %v and %v", secret1, secret2)
	}
}

func TestLoadOrGenerateIdentityKeys_CreatesKeys(t *testing.T) {
	tempDir := t.TempDir()
	coreDir := filepath.Join(tempDir, "core")
	_ = os.MkdirAll(coreDir, 0750)

	privPEM, pubPEM, err := loadOrGenerateIdentityKeys(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(privPEM) == 0 || len(pubPEM) == 0 {
		t.Errorf("expected private and public keys to be populated")
	}

	// Verify files exist
	if _, err := os.Stat(filepath.Join(coreDir, "identity.key")); os.IsNotExist(err) {
		t.Errorf("expected private key file to exist")
	}
	if _, err := os.Stat(filepath.Join(coreDir, "identity.pub")); os.IsNotExist(err) {
		t.Errorf("expected public key file to exist")
	}
}

func TestLoadOrGenerateIdentityKeys_ReadsExistingKeys(t *testing.T) {
	tempDir := t.TempDir()
	coreDir := filepath.Join(tempDir, "core")
	_ = os.MkdirAll(coreDir, 0750)

	priv1, pub1, _ := loadOrGenerateIdentityKeys(tempDir)

	// Call again, should read existing
	priv2, pub2, err := loadOrGenerateIdentityKeys(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Equal(priv1, priv2) {
		t.Errorf("expected private keys to be identical")
	}
	if !bytes.Equal(pub1, pub2) {
		t.Errorf("expected public keys to be identical")
	}
}
