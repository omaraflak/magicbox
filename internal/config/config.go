// Package config handles loading and managing Magicbox configuration.
package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// Config holds the runtime configuration for Magicbox.
type Config struct {
	Root      string
	Port      string
	JWTSecret []byte
	DBPath    string
}

// Load reads configuration from environment variables and initializes
// all required directories and secrets.
func Load() (*Config, error) {
	root := os.Getenv("MAGICBOX_ROOT")
	if root == "" {
		root = "/opt/magicbox"
	}

	port := os.Getenv("MAGICBOX_PORT")
	if port == "" {
		port = "80"
	}

	// Assert root directory exists.
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("root directory %q does not exist: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("root path %q is not a directory", root)
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

	return &Config{
		Root:      root,
		Port:      port,
		JWTSecret: jwtSecret,
		DBPath:    filepath.Join(root, "core", "magicbox.db"),
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
