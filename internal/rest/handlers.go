package rest

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/magicbox/core/internal/crypto"
)

const (
	maxBodySize = 1 << 20 // 1 MB
	bcryptCost  = 12
	minPassLen  = 8
	minUserLen  = 3
	maxUserLen  = 32
)

var usernameRegex = regexp.MustCompile(`^[a-z][a-z0-9_]{2,31}$`)

// ---------------------------------------------------------------------------
// JSON Helpers
// ---------------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func readJSON(r *http.Request, v interface{}) error {
	body := http.MaxBytesReader(nil, r.Body, maxBodySize)
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("failed to read body: %w", err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Health
// ---------------------------------------------------------------------------

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ---------------------------------------------------------------------------
// P2P / Key Helpers
// ---------------------------------------------------------------------------

// preferRelayAddr returns the first relay multiaddress (/p2p-circuit) if available,
// otherwise falls back to the first address in the slice, or empty string if slice is empty.
func preferRelayAddr(addrs []string) string {
	if len(addrs) == 0 {
		return ""
	}
	for _, addr := range addrs {
		if strings.Contains(addr, "/p2p-circuit") {
			return addr
		}
	}
	return addrs[0]
}

// localEncPubKeyHex retrieves and hex-encodes the local encryption public key.
func (s *Server) localEncPubKeyHex() (string, error) {
	pubKey, err := crypto.UnmarshalX25519PublicKey(s.config.Keys.EncryptionPubPEM)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(pubKey.Bytes()), nil
}

// localMasterPubKeyHex retrieves and hex-encodes the local master public key.
func (s *Server) localMasterPubKeyHex() (string, error) {
	masterPub, err := crypto.UnmarshalEd25519PublicKey(s.config.Keys.MasterPublicKeyPEM)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(masterPub), nil
}

