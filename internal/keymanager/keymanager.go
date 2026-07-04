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
	MasterIdentityPubPath string
	IdentityKeyPath       string
	IdentityPubPath       string
	EncryptionKeyPath     string
	EncryptionPubPath     string
	IdentityIndexPath     string
	EncryptionIndexPath   string
}

// NewKeyPaths returns the standard key file paths under the given root directory.
func NewKeyPaths(root string) *KeyPaths {
	return &KeyPaths{
		MasterIdentityPubPath: filepath.Join(root, "core", "master_identity.pub"),
		IdentityKeyPath:       filepath.Join(root, "core", "identity.key"),
		IdentityPubPath:       filepath.Join(root, "core", "identity.pub"),
		EncryptionKeyPath:     filepath.Join(root, "core", "encryption.key"),
		EncryptionPubPath:     filepath.Join(root, "core", "encryption.pub"),
		IdentityIndexPath:     filepath.Join(root, "core", "identity.index"),
		EncryptionIndexPath:   filepath.Join(root, "core", "encryption.index"),
	}
}

// KeyState holds the PEM-encoded key bytes, the optional mnemonic used for derivation,
// and the current derivation indices for identity and encryption keys.
type KeyState struct {
	MasterPublicKeyPEM []byte `json:"-"`
	PrivateKeyPEM      []byte `json:"-"`
	PublicKeyPEM       []byte `json:"-"`
	EncryptionKeyPEM   []byte `json:"-"`
	EncryptionPubPEM   []byte `json:"-"`
	Mnemonic           string `json:"-"`
	IdentityKeyIndex   int    `json:"identity_key_index"`
	EncryptionKeyIndex int    `json:"encryption_key_index"`
}

func readIndex(path string) (int, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var index int
	if _, err := fmt.Sscanf(string(bytes), "%d", &index); err != nil {
		return 0, err
	}
	return index, nil
}

func writeIndex(path string, index int) error {
	return os.WriteFile(path, []byte(fmt.Sprintf("%d", index)), 0644)
}

// LoadOrGenerate loads existing keys from disk, or generates new ones from a fresh mnemonic.
// If keys already exist on disk, they are loaded and the returned mnemonic will be empty.
// If keys are generated, the mnemonic is returned in-memory only (never written to disk).
func LoadOrGenerate(paths *KeyPaths) (*KeyState, error) {
	// Try reading all cached key files first
	masterPubPEM, err0 := os.ReadFile(paths.MasterIdentityPubPath)
	privPEM, err1 := os.ReadFile(paths.IdentityKeyPath)
	pubPEM, err2 := os.ReadFile(paths.IdentityPubPath)
	encKeyPEM, err3 := os.ReadFile(paths.EncryptionKeyPath)
	encPubPEM, err4 := os.ReadFile(paths.EncryptionPubPath)

	if err0 == nil && err1 == nil && err2 == nil && err3 == nil && err4 == nil {
		identityIndex, err := readIndex(paths.IdentityIndexPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read identity index: %w", err)
		}
		encryptionIndex, err := readIndex(paths.EncryptionIndexPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read encryption index: %w", err)
		}

		return &KeyState{
			MasterPublicKeyPEM: masterPubPEM,
			PrivateKeyPEM:      privPEM,
			PublicKeyPEM:       pubPEM,
			EncryptionKeyPEM:   encKeyPEM,
			EncryptionPubPEM:   encPubPEM,
			IdentityKeyIndex:   identityIndex,
			EncryptionKeyIndex: encryptionIndex,
		}, nil
	}

	// Generate a new mnemonic for initial key derivation (in-memory only)
	mnemonic, err := crypto.GenerateMnemonic()
	if err != nil {
		return nil, fmt.Errorf("failed to generate mnemonic: %w", err)
	}

	// Derive master identity key from mnemonic (always at index 0 initially)
	masterPriv, err := crypto.DeriveIdentityKey(mnemonic, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to derive master identity key from mnemonic: %w", err)
	}

	// Derive operational identity key from mnemonic (at index 1 initially)
	edPriv, err := crypto.DeriveIdentityKey(mnemonic, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to derive identity key from mnemonic: %w", err)
	}

	// Derive encryption key from mnemonic (at index 1 initially)
	xPriv, err := crypto.DeriveEncryptionKey(mnemonic, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to derive encryption key from mnemonic: %w", err)
	}

	// Marshal keys to PEM
	masterPubPEM, err = crypto.MarshalPublicKey(masterPriv.Public())
	if err != nil {
		return nil, err
	}
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
	if err := os.WriteFile(paths.MasterIdentityPubPath, masterPubPEM, 0644); err != nil {
		return nil, err
	}
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
	if err := writeIndex(paths.IdentityIndexPath, 1); err != nil {
		return nil, fmt.Errorf("failed to write initial identity index: %w", err)
	}
	if err := writeIndex(paths.EncryptionIndexPath, 1); err != nil {
		return nil, fmt.Errorf("failed to write initial encryption index: %w", err)
	}

	return &KeyState{
		MasterPublicKeyPEM: masterPubPEM,
		PrivateKeyPEM:      privPEM,
		PublicKeyPEM:       pubPEM,
		EncryptionKeyPEM:   encKeyPEM,
		EncryptionPubPEM:   encPubPEM,
		Mnemonic:           mnemonic,
		IdentityKeyIndex:   1,
		EncryptionKeyIndex: 1,
	}, nil
}

// RecoverAll derives both identity and encryption keys from a mnemonic at the given indices
// and writes all key files to disk.
func RecoverAll(paths *KeyPaths, mnemonic string, identityIndex, encryptionIndex int) error {
	// Derive master identity at index 0
	masterPriv, err := crypto.DeriveIdentityKey(mnemonic, 0)
	if err != nil {
		return fmt.Errorf("failed to derive master identity key from mnemonic: %w", err)
	}

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
	masterPubPEM, err := crypto.MarshalPublicKey(masterPriv.Public())
	if err != nil {
		return fmt.Errorf("failed to marshal master public key: %w", err)
	}
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
	if err := os.WriteFile(paths.MasterIdentityPubPath, masterPubPEM, 0644); err != nil {
		return fmt.Errorf("failed to write master public key: %w", err)
	}
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

	if err := writeIndex(paths.IdentityIndexPath, identityIndex); err != nil {
		return fmt.Errorf("failed to write identity index: %w", err)
	}
	if err := writeIndex(paths.EncryptionIndexPath, encryptionIndex); err != nil {
		return fmt.Errorf("failed to write encryption index: %w", err)
	}

	return nil
}

// RotateEncryption derives a new X25519 encryption key after incrementing the index on disk,
// and writes only the encryption key files to disk. Identity keys are not affected.
func RotateEncryption(paths *KeyPaths, mnemonic string) error {
	index, err := readIndex(paths.EncryptionIndexPath)
	if err != nil {
		return fmt.Errorf("failed to read encryption index: %w", err)
	}
	index++

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

	if err := writeIndex(paths.EncryptionIndexPath, index); err != nil {
		return fmt.Errorf("failed to write encryption index: %w", err)
	}

	return nil
}

// RotateIdentity derives a new Ed25519 identity key after incrementing the index on disk,
// and writes only the identity key files to disk. Encryption keys are not affected.
func RotateIdentity(paths *KeyPaths, mnemonic string) error {
	index, err := readIndex(paths.IdentityIndexPath)
	if err != nil {
		return fmt.Errorf("failed to read identity index: %w", err)
	}
	index++

	edPriv, err := crypto.DeriveIdentityKey(mnemonic, index)
	if err != nil {
		return fmt.Errorf("failed to derive identity key from mnemonic: %w", err)
	}

	privPEM, err := crypto.MarshalPrivateKey(edPriv)
	if err != nil {
		return fmt.Errorf("failed to marshal identity private key: %w", err)
	}
	pubPEM, err := crypto.MarshalPublicKey(edPriv.Public())
	if err != nil {
		return fmt.Errorf("failed to marshal identity public key: %w", err)
	}

	if err := os.WriteFile(paths.IdentityKeyPath, privPEM, 0600); err != nil {
		return fmt.Errorf("failed to write identity private key: %w", err)
	}
	if err := os.WriteFile(paths.IdentityPubPath, pubPEM, 0644); err != nil {
		return fmt.Errorf("failed to write identity public key: %w", err)
	}

	if err := writeIndex(paths.IdentityIndexPath, index); err != nil {
		return fmt.Errorf("failed to write identity index: %w", err)
	}

	return nil
}

