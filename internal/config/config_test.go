package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/magicbox/core/internal/crypto"
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

func TestRecoverKeys_Wrapper(t *testing.T) {
	tempDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tempDir, "core"), 0750)
	mnemonic, _ := crypto.GenerateMnemonic()
	if err := RecoverKeys(tempDir, mnemonic, 0, 0); err != nil {
		t.Fatalf("RecoverKeys wrapper failed: %v", err)
	}
}

func TestRotateEncryptionKey_Wrapper(t *testing.T) {
	tempDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tempDir, "core"), 0750)
	mnemonic, _ := crypto.GenerateMnemonic()
	if err := RotateEncryptionKey(tempDir, mnemonic, 5); err != nil {
		t.Fatalf("RotateEncryptionKey wrapper failed: %v", err)
	}
}
