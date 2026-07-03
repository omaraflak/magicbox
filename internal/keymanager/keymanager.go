// Package keymanager centralizes cryptographic key lifecycle operations:
// generation, derivation, rotation, recovery, and PEM serialization.
package keymanager

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/magicbox/core/internal/crypto"
)

// KeyPaths holds the filesystem paths for all key files.
type KeyPaths struct {
	IdentityKeyPath   string
	IdentityPubPath   string
	EncryptionKeyPath string
	EncryptionPubPath string
}

// NewKeyPaths returns the standard key file paths under the given root directory.
func NewKeyPaths(root string) *KeyPaths {
	return &KeyPaths{
		IdentityKeyPath:   filepath.Join(root, "core", "identity.key"),
		IdentityPubPath:   filepath.Join(root, "core", "identity.pub"),
		EncryptionKeyPath: filepath.Join(root, "core", "encryption.key"),
		EncryptionPubPath: filepath.Join(root, "core", "encryption.pub"),
	}
}

// KeyState holds the PEM-encoded key bytes, the optional mnemonic used for derivation,
// and the current derivation indices for identity and encryption keys.
type KeyState struct {
	PrivateKeyPEM      []byte `json:"-"`
	PublicKeyPEM       []byte `json:"-"`
	EncryptionKeyPEM   []byte `json:"-"`
	EncryptionPubPEM   []byte `json:"-"`
	Mnemonic           string `json:"-"`
	IdentityKeyIndex   int    `json:"identity_key_index"`
	EncryptionKeyIndex int    `json:"encryption_key_index"`
}

// LoadOrGenerate loads existing keys from disk, or generates new ones from a fresh mnemonic.
// If keys already exist on disk, they are loaded and the returned mnemonic will be empty.
// If keys are generated, the mnemonic is returned in-memory only (never written to disk).
func LoadOrGenerate(paths *KeyPaths) (*KeyState, error) {
	// Try reading all cached key files first
	privPEM, err1 := os.ReadFile(paths.IdentityKeyPath)
	pubPEM, err2 := os.ReadFile(paths.IdentityPubPath)
	encKeyPEM, err3 := os.ReadFile(paths.EncryptionKeyPath)
	encPubPEM, err4 := os.ReadFile(paths.EncryptionPubPath)

	if err1 == nil && err2 == nil && err3 == nil && err4 == nil {
		// Key indices default to 0 here; the caller must load the actual
		// indices from the database and set them on the returned KeyState.
		return &KeyState{
			PrivateKeyPEM:    privPEM,
			PublicKeyPEM:     pubPEM,
			EncryptionKeyPEM: encKeyPEM,
			EncryptionPubPEM: encPubPEM,
		}, nil
	}

	// Generate a new mnemonic for initial key derivation (in-memory only)
	mnemonic, err := crypto.GenerateMnemonic()
	if err != nil {
		return nil, fmt.Errorf("failed to generate mnemonic: %w", err)
	}

	// Derive keys from mnemonic (always at index 0 initially)
	edPriv, err := crypto.DeriveIdentityKey(mnemonic, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to derive identity key from mnemonic: %w", err)
	}
	xPriv, err := crypto.DeriveEncryptionKey(mnemonic, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to derive encryption key from mnemonic: %w", err)
	}

	// Marshal keys to PEM
	privPEM, err = crypto.MarshalPrivateKey(edPriv)
	if err != nil {
		return nil, err
	}
	pubPEM, err = crypto.MarshalPublicKey(edPriv.Public())
	if err != nil {
		return nil, err
	}
	encKeyPEM, err = crypto.MarshalPrivateKey(xPriv)
	if err != nil {
		return nil, err
	}
	encPubPEM, err = crypto.MarshalPublicKey(xPriv.PublicKey())
	if err != nil {
		return nil, err
	}

	// Save all key files to disk
	if err := os.WriteFile(paths.IdentityKeyPath, privPEM, 0600); err != nil {
		return nil, err
	}
	if err := os.WriteFile(paths.IdentityPubPath, pubPEM, 0644); err != nil {
		return nil, err
	}
	if err := os.WriteFile(paths.EncryptionKeyPath, encKeyPEM, 0600); err != nil {
		return nil, err
	}
	if err := os.WriteFile(paths.EncryptionPubPath, encPubPEM, 0644); err != nil {
		return nil, err
	}

	return &KeyState{
		PrivateKeyPEM:      privPEM,
		PublicKeyPEM:       pubPEM,
		EncryptionKeyPEM:   encKeyPEM,
		EncryptionPubPEM:   encPubPEM,
		Mnemonic:           mnemonic,
		IdentityKeyIndex:   0,
		EncryptionKeyIndex: 0,
	}, nil
}

// RecoverAll derives both identity and encryption keys from a mnemonic at the given indices
// and writes all key files to disk.
func RecoverAll(paths *KeyPaths, mnemonic string, identityIndex, encryptionIndex int) error {
	// Derive identity key from mnemonic (validates mnemonic internally).
	edPriv, err := crypto.DeriveIdentityKey(mnemonic, identityIndex)
	if err != nil {
		return fmt.Errorf("failed to derive identity key from mnemonic: %w", err)
	}
	xPriv, err := crypto.DeriveEncryptionKey(mnemonic, encryptionIndex)
	if err != nil {
		return fmt.Errorf("failed to derive encryption key from mnemonic: %w", err)
	}

	// Marshal keys to PEM.
	privPEM, err := crypto.MarshalPrivateKey(edPriv)
	if err != nil {
		return fmt.Errorf("failed to marshal identity private key: %w", err)
	}
	pubPEM, err := crypto.MarshalPublicKey(edPriv.Public())
	if err != nil {
		return fmt.Errorf("failed to marshal identity public key: %w", err)
	}
	encKeyPEM, err := crypto.MarshalPrivateKey(xPriv)
	if err != nil {
		return fmt.Errorf("failed to marshal encryption private key: %w", err)
	}
	encPubPEM, err := crypto.MarshalPublicKey(xPriv.PublicKey())
	if err != nil {
		return fmt.Errorf("failed to marshal encryption public key: %w", err)
	}

	// Write all key files.
	if err := os.WriteFile(paths.IdentityKeyPath, privPEM, 0600); err != nil {
		return fmt.Errorf("failed to write identity private key: %w", err)
	}
	if err := os.WriteFile(paths.IdentityPubPath, pubPEM, 0644); err != nil {
		return fmt.Errorf("failed to write identity public key: %w", err)
	}
	if err := os.WriteFile(paths.EncryptionKeyPath, encKeyPEM, 0600); err != nil {
		return fmt.Errorf("failed to write encryption private key: %w", err)
	}
	if err := os.WriteFile(paths.EncryptionPubPath, encPubPEM, 0644); err != nil {
		return fmt.Errorf("failed to write encryption public key: %w", err)
	}

	return nil
}

// RotateEncryption derives a new X25519 encryption key at the given index
// and writes only the encryption key files to disk. Identity keys are not affected.
func RotateEncryption(paths *KeyPaths, mnemonic string, index int) error {
	// Derive encryption key from mnemonic at the given index.
	xPriv, err := crypto.DeriveEncryptionKey(mnemonic, index)
	if err != nil {
		return fmt.Errorf("failed to derive encryption key from mnemonic: %w", err)
	}

	encKeyPEM, err := crypto.MarshalPrivateKey(xPriv)
	if err != nil {
		return fmt.Errorf("failed to marshal encryption private key: %w", err)
	}
	encPubPEM, err := crypto.MarshalPublicKey(xPriv.PublicKey())
	if err != nil {
		return fmt.Errorf("failed to marshal encryption public key: %w", err)
	}

	if err := os.WriteFile(paths.EncryptionKeyPath, encKeyPEM, 0600); err != nil {
		return fmt.Errorf("failed to write encryption private key: %w", err)
	}
	if err := os.WriteFile(paths.EncryptionPubPath, encPubPEM, 0644); err != nil {
		return fmt.Errorf("failed to write encryption public key: %w", err)
	}

	return nil
}
