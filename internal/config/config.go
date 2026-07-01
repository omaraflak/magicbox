// Package config handles loading and managing Magicbox configuration.
package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/magicbox/core/internal/crypto"
)

// Config holds the runtime configuration for Magicbox.
type Config struct {
	Root             string
	HostRoot         string
	Port             string
	JWTSecret        []byte
	DBPath           string
	PrivateKeyPEM    []byte
	PublicKeyPEM     []byte
	EncryptionKeyPEM []byte
	EncryptionPubPEM []byte
}

// Load reads configuration from environment variables and initializes
// all required directories and secrets.
func Load() (*Config, error) {
	root := "/opt/magicbox"
	if err := os.MkdirAll(root, 0750); err != nil {
		root = "./data"
		if err := os.MkdirAll(root, 0750); err != nil {
			return nil, fmt.Errorf("failed to create root directory: %w", err)
		}
	}

	port := os.Getenv("MAGICBOX_PORT")
	if port == "" {
		port = "80"
	}

	// Create required subdirectories.
	subdirs := []string{
		"core",
		"core/logs",
		"core/web",
		"backups",
		"transit",
		"users",
	}
	for _, sub := range subdirs {
		dir := filepath.Join(root, sub)
		if err := os.MkdirAll(dir, 0750); err != nil {
			return nil, fmt.Errorf("failed to create directory %q: %w", dir, err)
		}
	}

	// Sync web assets from staging area to core/web
	if err := syncWebAssets(root); err != nil {
		log.Printf("Warning: failed to sync web assets: %v", err)
	}

	// Load or generate JWT secret.
	jwtSecret, err := loadOrGenerateJWTSecret(filepath.Join(root, "core", "jwt_secret"))
	if err != nil {
		return nil, fmt.Errorf("failed to load JWT secret: %w", err)
	}

	// Load or generate deterministic keys.
	privPEM, pubPEM, encKeyPEM, encPubPEM, err := loadOrGenerateKeys(root)
	if err != nil {
		return nil, fmt.Errorf("failed to load/generate keys: %w", err)
	}

	hostRoot := os.Getenv("MAGICBOX_HOST_ROOT")
	if hostRoot == "" {
		hostRoot = root
	}

	return &Config{
		Root:             root,
		HostRoot:         hostRoot,
		Port:             port,
		JWTSecret:        jwtSecret,
		DBPath:           filepath.Join(root, "core", "magicbox.db"),
		PrivateKeyPEM:    privPEM,
		PublicKeyPEM:     pubPEM,
		EncryptionKeyPEM: encKeyPEM,
		EncryptionPubPEM: encPubPEM,
	}, nil
}

// loadOrGenerateJWTSecret reads an existing hex-encoded secret from file,
// or generates a new 32-byte secret if the file does not exist.
func loadOrGenerateJWTSecret(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		// File exists — hex-decode and return.
		secret, decErr := hex.DecodeString(string(data))
		if decErr != nil {
			return nil, fmt.Errorf("failed to hex-decode JWT secret: %w", decErr)
		}
		return secret, nil
	}

	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read JWT secret file: %w", err)
	}

	// Generate new secret.
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("failed to generate JWT secret: %w", err)
	}

	encoded := hex.EncodeToString(secret)
	if err := os.WriteFile(path, []byte(encoded), 0600); err != nil {
		return nil, fmt.Errorf("failed to write JWT secret file: %w", err)
	}

	log.Println("WARNING: new JWT secret generated — all existing sessions are invalidated")

	return secret, nil
}

func syncWebAssets(root string) error {
	src := "/app/web"
	if _, err := os.Stat(src); err != nil {
		src = "web"
		if _, err := os.Stat(src); err != nil {
			return fmt.Errorf("no web source directory found at /app/web or ./web")
		}
	}

	dest := filepath.Join(root, "core", "web")

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)

		if info.IsDir() {
			return os.MkdirAll(target, 0750)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(target, data, info.Mode())
	})
}

// loadOrGenerateKeys loads existing Ed25519/X25519 keys from disk, or generates/derives them from a mnemonic.
func loadOrGenerateKeys(root string) (privPEM, pubPEM, encKeyPEM, encPubPEM []byte, err error) {
	mnemonicPath := filepath.Join(root, "core", "mnemonic")
	privPath := filepath.Join(root, "core", "identity.key")
	pubPath := filepath.Join(root, "core", "identity.pub")
	encKeyPath := filepath.Join(root, "core", "encryption.key")
	encPubPath := filepath.Join(root, "core", "encryption.pub")

	// Try reading all cached key files first
	privPEM, err1 := os.ReadFile(privPath)
	pubPEM, err2 := os.ReadFile(pubPath)
	encKeyPEM, err3 := os.ReadFile(encKeyPath)
	encPubPEM, err4 := os.ReadFile(encPubPath)

	if err1 == nil && err2 == nil && err3 == nil && err4 == nil {
		return privPEM, pubPEM, encKeyPEM, encPubPEM, nil
	}

	// Otherwise, load or generate mnemonic
	var mnemonic string
	passphrase := os.Getenv("MAGICBOX_RECOVERY_PASSPHRASE")
	if passphrase != "" {
		mnemonic = passphrase
		log.Println("Initializing Magicbox keys from MAGICBOX_RECOVERY_PASSPHRASE")
	} else {
		mBytes, err := os.ReadFile(mnemonicPath)
		if err == nil {
			mnemonic = string(mBytes)
		} else {
			mnemonic, err = crypto.GenerateMnemonic()
			if err != nil {
				return nil, nil, nil, nil, fmt.Errorf("failed to generate mnemonic: %w", err)
			}
			if err := os.WriteFile(mnemonicPath, []byte(mnemonic), 0600); err != nil {
				return nil, nil, nil, nil, fmt.Errorf("failed to write mnemonic: %w", err)
			}
			log.Println("New BIP-39 mnemonic generated and saved to core/mnemonic")
		}
	}

	// Derive keys from mnemonic
	edPriv, xPriv, err := crypto.DeriveKeys(mnemonic)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to derive keys from mnemonic: %w", err)
	}

	// Marshal keys to PEM
	privPEM, err = crypto.MarshalPrivateKey(edPriv)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	pubPEM, err = crypto.MarshalPublicKey(edPriv.Public())
	if err != nil {
		return nil, nil, nil, nil, err
	}
	encKeyPEM, err = crypto.MarshalPrivateKey(xPriv)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	encPubPEM, err = crypto.MarshalPublicKey(xPriv.PublicKey())
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// Save all key files to disk
	if err := os.WriteFile(privPath, privPEM, 0600); err != nil {
		return nil, nil, nil, nil, err
	}
	if err := os.WriteFile(pubPath, pubPEM, 0644); err != nil {
		return nil, nil, nil, nil, err
	}
	if err := os.WriteFile(encKeyPath, encKeyPEM, 0600); err != nil {
		return nil, nil, nil, nil, err
	}
	if err := os.WriteFile(encPubPath, encPubPEM, 0644); err != nil {
		return nil, nil, nil, nil, err
	}

	return privPEM, pubPEM, encKeyPEM, encPubPEM, nil
}
