package invite

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestParse_ValidLink(t *testing.T) {
	original := &Payload{
		Multiaddr: "/ip4/1.2.3.4/tcp/4001/p2p/QmPeerID123",
		UserID:    "user-abc",
		EncPubKey: "enc-pub-key-xyz",
	}

	link, err := Build(original)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	parsed, err := Parse(link)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parsed.Multiaddr != original.Multiaddr {
		t.Errorf("Multiaddr mismatch: got %q, want %q", parsed.Multiaddr, original.Multiaddr)
	}
	if parsed.UserID != original.UserID {
		t.Errorf("UserID mismatch: got %q, want %q", parsed.UserID, original.UserID)
	}
	if parsed.EncPubKey != original.EncPubKey {
		t.Errorf("EncPubKey mismatch: got %q, want %q", parsed.EncPubKey, original.EncPubKey)
	}
}

func TestParse_MissingPrefix(t *testing.T) {
	_, err := Parse("https://example.com/invite/abc")
	if err == nil {
		t.Error("expected error for missing prefix, got nil")
	}
	if !strings.Contains(err.Error(), "missing prefix") {
		t.Errorf("expected 'missing prefix' in error, got: %v", err)
	}
}

func TestParse_EmptyPayload(t *testing.T) {
	_, err := Parse(Prefix)
	if err == nil {
		t.Error("expected error for empty payload, got nil")
	}
	if !strings.Contains(err.Error(), "empty payload") {
		t.Errorf("expected 'empty payload' in error, got: %v", err)
	}
}

func TestParse_BadBase64(t *testing.T) {
	_, err := Parse(Prefix + "!!!not-valid-base64!!!")
	if err == nil {
		t.Error("expected error for bad base64, got nil")
	}
	if !strings.Contains(err.Error(), "bad base64") {
		t.Errorf("expected 'bad base64' in error, got: %v", err)
	}
}

func TestParse_BadJSON(t *testing.T) {
	encoded := base64.URLEncoding.EncodeToString([]byte("not valid json"))
	_, err := Parse(Prefix + encoded)
	if err == nil {
		t.Error("expected error for bad JSON, got nil")
	}
	if !strings.Contains(err.Error(), "bad JSON") {
		t.Errorf("expected 'bad JSON' in error, got: %v", err)
	}
}

func TestBuild_RoundTrip(t *testing.T) {
	original := &Payload{
		Multiaddr: "/ip4/10.0.0.1/tcp/9000/p2p/QmRoundTrip",
		UserID:    "user-roundtrip",
		EncPubKey: "roundtrip-key",
	}

	link, err := Build(original)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !strings.HasPrefix(link, Prefix) {
		t.Errorf("expected link to start with %q, got %q", Prefix, link)
	}

	parsed, err := Parse(link)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parsed.Multiaddr != original.Multiaddr || parsed.UserID != original.UserID || parsed.EncPubKey != original.EncPubKey {
		t.Errorf("round-trip mismatch: got %+v, want %+v", parsed, original)
	}
}

func TestExtractPeerID_NormalMultiaddr(t *testing.T) {
	multiaddr := "/ip4/1.2.3.4/tcp/4001/p2p/QmPeerID123"
	got := ExtractPeerID(multiaddr)
	want := "QmPeerID123"
	if got != want {
		t.Errorf("ExtractPeerID(%q) = %q, want %q", multiaddr, got, want)
	}
}

func TestExtractPeerID_CircuitRelayMultiaddr(t *testing.T) {
	multiaddr := "/ip4/1.2.3.4/tcp/4001/p2p/QmRelay/p2p-circuit/p2p/QmTarget"
	got := ExtractPeerID(multiaddr)
	want := "QmTarget"
	if got != want {
		t.Errorf("ExtractPeerID(%q) = %q, want %q", multiaddr, got, want)
	}
}

func TestExtractPeerID_NoP2PSegment(t *testing.T) {
	multiaddr := "/ip4/1.2.3.4/tcp/4001"
	got := ExtractPeerID(multiaddr)
	if got != "" {
		t.Errorf("ExtractPeerID(%q) = %q, want empty string", multiaddr, got)
	}
}

func TestExtractPeerID_EmptyString(t *testing.T) {
	got := ExtractPeerID("")
	if got != "" {
		t.Errorf("ExtractPeerID(\"\") = %q, want empty string", got)
	}
}
