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

func TestLoadOrGenerateKeys_CreatesKeysAndMnemonic(t *testing.T) {
	tempDir := t.TempDir()
	coreDir := filepath.Join(tempDir, "core")
	_ = os.MkdirAll(coreDir, 0750)

	privPEM, pubPEM, encKeyPEM, encPubPEM, err := loadOrGenerateKeys(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(privPEM) == 0 || len(pubPEM) == 0 || len(encKeyPEM) == 0 || len(encPubPEM) == 0 {
		t.Errorf("expected all 4 keys to be generated and populated")
	}

	// Verify files exist
	if _, err := os.Stat(filepath.Join(coreDir, "mnemonic")); os.IsNotExist(err) {
		t.Errorf("expected mnemonic file to exist")
	}
	if _, err := os.Stat(filepath.Join(coreDir, "identity.key")); os.IsNotExist(err) {
		t.Errorf("expected identity private key file to exist")
	}
	if _, err := os.Stat(filepath.Join(coreDir, "identity.pub")); os.IsNotExist(err) {
		t.Errorf("expected identity public key file to exist")
	}
	if _, err := os.Stat(filepath.Join(coreDir, "encryption.key")); os.IsNotExist(err) {
		t.Errorf("expected encryption private key file to exist")
	}
	if _, err := os.Stat(filepath.Join(coreDir, "encryption.pub")); os.IsNotExist(err) {
		t.Errorf("expected encryption public key file to exist")
	}
}

func TestLoadOrGenerateKeys_ReadsExistingKeys(t *testing.T) {
	tempDir := t.TempDir()
	coreDir := filepath.Join(tempDir, "core")
	_ = os.MkdirAll(coreDir, 0750)

	priv1, pub1, encKey1, encPub1, _ := loadOrGenerateKeys(tempDir)

	// Call again, should read existing
	priv2, pub2, encKey2, encPub2, err := loadOrGenerateKeys(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Equal(priv1, priv2) {
		t.Errorf("expected identity private keys to be identical")
	}
	if !bytes.Equal(pub1, pub2) {
		t.Errorf("expected identity public keys to be identical")
	}
	if !bytes.Equal(encKey1, encKey2) {
		t.Errorf("expected encryption private keys to be identical")
	}
	if !bytes.Equal(encPub1, encPub2) {
		t.Errorf("expected encryption public keys to be identical")
	}
}

func TestLoadOrGenerateKeys_DeterministicRecovery(t *testing.T) {
	mnemonic, err := crypto.GenerateMnemonic()
	if err != nil {
		t.Fatalf("failed to generate mnemonic: %v", err)
	}

	os.Setenv("MAGICBOX_RECOVERY_PASSPHRASE", mnemonic)
	defer os.Unsetenv("MAGICBOX_RECOVERY_PASSPHRASE")

	// Generate in TempDir 1
	tempDir1 := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tempDir1, "core"), 0750)
	priv1, pub1, encKey1, encPub1, err := loadOrGenerateKeys(tempDir1)
	if err != nil {
		t.Fatalf("unexpected error on dir 1: %v", err)
	}

	// Generate in TempDir 2
	tempDir2 := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tempDir2, "core"), 0750)
	priv2, pub2, encKey2, encPub2, err := loadOrGenerateKeys(tempDir2)
	if err != nil {
		t.Fatalf("unexpected error on dir 2: %v", err)
	}

	if !bytes.Equal(priv1, priv2) {
		t.Errorf("deterministic identity private keys did not match across directories")
	}
	if !bytes.Equal(pub1, pub2) {
		t.Errorf("deterministic identity public keys did not match across directories")
	}
	if !bytes.Equal(encKey1, encKey2) {
		t.Errorf("deterministic encryption private keys did not match across directories")
	}
	if !bytes.Equal(encPub1, encPub2) {
		t.Errorf("deterministic encryption public keys did not match across directories")
	}
}

func TestLoadOrGenerateKeys_InvalidRecoveryPassphraseFails(t *testing.T) {
	os.Setenv("MAGICBOX_RECOVERY_PASSPHRASE", "invalid passphrase structure")
	defer os.Unsetenv("MAGICBOX_RECOVERY_PASSPHRASE")

	tempDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tempDir, "core"), 0750)
	_, _, _, _, err := loadOrGenerateKeys(tempDir)
	if err == nil {
		t.Error("expected error for invalid recovery passphrase, got nil")
	}
}
