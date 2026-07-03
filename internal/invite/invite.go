package invite

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

const Prefix = "magicbox://invite/"

// Payload represents the fields encoded in a magicbox invite link.
type Payload struct {
	Multiaddr string `json:"multiaddr"`
	UserID    string `json:"user_id"`
	EncPubKey string `json:"enc_pub_key"`
}

// Parse decodes a magicbox://invite/<base64> link into its payload components.
func Parse(link string) (*Payload, error) {
	if !strings.HasPrefix(link, Prefix) {
		return nil, fmt.Errorf("invalid invite link: missing prefix %q", Prefix)
	}

	encoded := strings.TrimPrefix(link, Prefix)
	if encoded == "" {
		return nil, fmt.Errorf("invalid invite link: empty payload")
	}

	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("invalid invite link: bad base64: %w", err)
	}

	var payload Payload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("invalid invite link: bad JSON: %w", err)
	}

	return &payload, nil
}

// Build encodes a Payload into a magicbox://invite/<base64> link.
func Build(payload *Payload) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal invite payload: %w", err)
	}

	encoded := base64.URLEncoding.EncodeToString(data)
	return Prefix + encoded, nil
}

// ExtractPeerID extracts the peer ID from a multiaddr string (the segment after the last "/p2p/").
func ExtractPeerID(multiaddr string) string {
	const sep = "/p2p/"
	idx := strings.LastIndex(multiaddr, sep)
	if idx == -1 {
		return ""
	}
	return multiaddr[idx+len(sep):]
}
