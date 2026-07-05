package crypto

import (
	"bytes"
	"crypto/ed25519"
	"testing"
)

func TestPEMRoundTrip_Ed25519PrivateKey(t *testing.T) {
	mnemonic, _ := GenerateMnemonic()
	edPriv, _ := DeriveIdentityKey(mnemonic, 0)

	pemBytes, err := MarshalPrivateKey(edPriv)
	if err != nil {
		t.Fatalf("failed to marshal Ed25519 private key: %v", err)
	}
	parsed, err := UnmarshalEd25519PrivateKey(pemBytes)
	if err != nil {
		t.Fatalf("failed to unmarshal Ed25519 private key: %v", err)
	}
	if !bytes.Equal(edPriv, parsed) {
		t.Error("Ed25519 private keys do not match after PEM roundtrip")
	}
}

func TestPEMRoundTrip_Ed25519PublicKey(t *testing.T) {
	mnemonic, _ := GenerateMnemonic()
	edPriv, _ := DeriveIdentityKey(mnemonic, 0)
	edPub := edPriv.Public().(ed25519.PublicKey)

	pemBytes, err := MarshalPublicKey(edPub)
	if err != nil {
		t.Fatalf("failed to marshal Ed25519 public key: %v", err)
	}
	parsed, err := UnmarshalEd25519PublicKey(pemBytes)
	if err != nil {
		t.Fatalf("failed to unmarshal Ed25519 public key: %v", err)
	}
	if !bytes.Equal(edPub, parsed) {
		t.Error("Ed25519 public keys do not match after PEM roundtrip")
	}
}

func TestPEMRoundTrip_X25519PrivateKey(t *testing.T) {
	mnemonic, _ := GenerateMnemonic()
	xPriv, _ := DeriveEncryptionKey(mnemonic, 0)

	pemBytes, err := MarshalPrivateKey(xPriv)
	if err != nil {
		t.Fatalf("failed to marshal X25519 private key: %v", err)
	}
	parsed, err := UnmarshalX25519PrivateKey(pemBytes)
	if err != nil {
		t.Fatalf("failed to unmarshal X25519 private key: %v", err)
	}
	if !bytes.Equal(xPriv.Bytes(), parsed.Bytes()) {
		t.Error("X25519 private keys do not match after PEM roundtrip")
	}
}

func TestPEMRoundTrip_X25519PublicKey(t *testing.T) {
	mnemonic, _ := GenerateMnemonic()
	xPriv, _ := DeriveEncryptionKey(mnemonic, 0)
	xPub := xPriv.PublicKey()

	pemBytes, err := MarshalPublicKey(xPub)
	if err != nil {
		t.Fatalf("failed to marshal X25519 public key: %v", err)
	}
	parsed, err := UnmarshalX25519PublicKey(pemBytes)
	if err != nil {
		t.Fatalf("failed to unmarshal X25519 public key: %v", err)
	}
	if !bytes.Equal(xPub.Bytes(), parsed.Bytes()) {
		t.Error("X25519 public keys do not match after PEM roundtrip")
	}
}

func TestUnmarshalEd25519PrivateKey_InvalidPEMFails(t *testing.T) {
	_, err := UnmarshalEd25519PrivateKey([]byte("invalid pem blocks"))
	if err == nil {
		t.Error("expected error for invalid PEM data, got nil")
	}
}

func TestUnmarshalX25519PrivateKey_InvalidPEMFails(t *testing.T) {
	_, err := UnmarshalX25519PrivateKey([]byte("invalid pem blocks"))
	if err == nil {
		t.Error("expected error for invalid PEM data, got nil")
	}
}

func TestUnmarshalEd25519PublicKey_InvalidPEMFails(t *testing.T) {
	_, err := UnmarshalEd25519PublicKey([]byte("invalid pem blocks"))
	if err == nil {
		t.Error("expected error for invalid PEM data, got nil")
	}
}

func TestUnmarshalX25519PublicKey_InvalidPEMFails(t *testing.T) {
	_, err := UnmarshalX25519PublicKey([]byte("invalid pem blocks"))
	if err == nil {
		t.Error("expected error for invalid PEM data, got nil")
	}
}

func TestUnmarshalEd25519PrivateKey_WrongKeyTypeFails(t *testing.T) {
	mnemonic, _ := GenerateMnemonic()
	xPriv, _ := DeriveEncryptionKey(mnemonic, 0)

	// Marshal X25519 private key, then try to unmarshal as Ed25519
	pemBytes, err := MarshalPrivateKey(xPriv)
	if err != nil {
		t.Fatalf("failed to marshal X25519 key: %v", err)
	}
	_, err = UnmarshalEd25519PrivateKey(pemBytes)
	if err == nil {
		t.Error("expected error when unmarshaling X25519 PEM as Ed25519, got nil")
	}
}

func TestUnmarshalX25519PrivateKey_WrongKeyTypeFails(t *testing.T) {
	mnemonic, _ := GenerateMnemonic()
	edPriv, _ := DeriveIdentityKey(mnemonic, 0)

	// Marshal Ed25519 private key, then try to unmarshal as X25519
	pemBytes, err := MarshalPrivateKey(edPriv)
	if err != nil {
		t.Fatalf("failed to marshal Ed25519 key: %v", err)
	}
	_, err = UnmarshalX25519PrivateKey(pemBytes)
	if err == nil {
		t.Error("expected error when unmarshaling Ed25519 PEM as X25519, got nil")
	}
}

func TestUnmarshalEd25519PublicKey_WrongKeyTypeFails(t *testing.T) {
	mnemonic, _ := GenerateMnemonic()
	xPriv, _ := DeriveEncryptionKey(mnemonic, 0)

	pemBytes, err := MarshalPublicKey(xPriv.PublicKey())
	if err != nil {
		t.Fatalf("failed to marshal X25519 public key: %v", err)
	}
	_, err = UnmarshalEd25519PublicKey(pemBytes)
	if err == nil {
		t.Error("expected error when unmarshaling X25519 public PEM as Ed25519, got nil")
	}
}

func TestUnmarshalX25519PublicKey_WrongKeyTypeFails(t *testing.T) {
	mnemonic, _ := GenerateMnemonic()
	edPriv, _ := DeriveIdentityKey(mnemonic, 0)

	pemBytes, err := MarshalPublicKey(edPriv.Public())
	if err != nil {
		t.Fatalf("failed to marshal Ed25519 public key: %v", err)
	}
	_, err = UnmarshalX25519PublicKey(pemBytes)
	if err == nil {
		t.Error("expected error when unmarshaling Ed25519 public PEM as X25519, got nil")
	}
}
