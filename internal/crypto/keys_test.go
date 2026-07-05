package crypto

import (
	"bytes"
	"testing"
)

func TestDeriveIdentityKey_Success(t *testing.T) {
	mnemonic, _ := GenerateMnemonic()

	edPriv1, err := DeriveIdentityKey(mnemonic, 0)
	if err != nil {
		t.Fatalf("failed to derive identity key: %v", err)
	}

	edPriv2, err := DeriveIdentityKey(mnemonic, 0)
	if err != nil {
		t.Fatalf("failed to derive identity key 2: %v", err)
	}

	// Assert determinism
	if !bytes.Equal(edPriv1, edPriv2) {
		t.Error("deterministic Ed25519 keys do not match")
	}
}

func TestDeriveEncryptionKey_Success(t *testing.T) {
	mnemonic, _ := GenerateMnemonic()

	xPriv1, err := DeriveEncryptionKey(mnemonic, 0)
	if err != nil {
		t.Fatalf("failed to derive encryption key: %v", err)
	}

	xPriv2, err := DeriveEncryptionKey(mnemonic, 0)
	if err != nil {
		t.Fatalf("failed to derive encryption key 2: %v", err)
	}

	// Assert determinism
	if !bytes.Equal(xPriv1.Bytes(), xPriv2.Bytes()) {
		t.Error("deterministic X25519 keys do not match")
	}
}

func TestDeriveIdentityKey_InvalidMnemonicFails(t *testing.T) {
	_, err := DeriveIdentityKey("invalid mnemonic sentence that fails checksum completely", 0)
	if err == nil {
		t.Error("expected error for invalid mnemonic phrase, got nil")
	}
}

func TestDeriveEncryptionKey_InvalidMnemonicFails(t *testing.T) {
	_, err := DeriveEncryptionKey("invalid mnemonic sentence that fails checksum completely", 0)
	if err == nil {
		t.Error("expected error for invalid mnemonic phrase, got nil")
	}
}

func TestDeriveIdentityKey_DifferentIndicesProduceDifferentKeys(t *testing.T) {
	mnemonic, _ := GenerateMnemonic()

	edPriv1, err := DeriveIdentityKey(mnemonic, 0)
	if err != nil {
		t.Fatalf("failed to derive identity key at index 0: %v", err)
	}

	edPriv2, err := DeriveIdentityKey(mnemonic, 1)
	if err != nil {
		t.Fatalf("failed to derive identity key at index 1: %v", err)
	}

	if bytes.Equal(edPriv1, edPriv2) {
		t.Error("expected different Ed25519 keys for different indices, but they are the same")
	}
}

func TestDeriveEncryptionKey_DifferentIndicesProduceDifferentKeys(t *testing.T) {
	mnemonic, _ := GenerateMnemonic()

	xPriv1, err := DeriveEncryptionKey(mnemonic, 0)
	if err != nil {
		t.Fatalf("failed to derive encryption key at index 0: %v", err)
	}

	xPriv2, err := DeriveEncryptionKey(mnemonic, 1)
	if err != nil {
		t.Fatalf("failed to derive encryption key at index 1: %v", err)
	}

	if bytes.Equal(xPriv1.Bytes(), xPriv2.Bytes()) {
		t.Error("expected different X25519 keys for different indices, but they are the same")
	}
}

func TestDeriveIdentityKey_DifferentMnemonicsProduceDifferentKeys(t *testing.T) {
	m1, _ := GenerateMnemonic()
	m2, _ := GenerateMnemonic()

	k1, _ := DeriveIdentityKey(m1, 0)
	k2, _ := DeriveIdentityKey(m2, 0)

	if bytes.Equal(k1, k2) {
		t.Error("expected different keys for different mnemonics")
	}
}

func TestDeriveEncryptionKey_DifferentMnemonicsProduceDifferentKeys(t *testing.T) {
	m1, _ := GenerateMnemonic()
	m2, _ := GenerateMnemonic()

	k1, _ := DeriveEncryptionKey(m1, 0)
	k2, _ := DeriveEncryptionKey(m2, 0)

	if bytes.Equal(k1.Bytes(), k2.Bytes()) {
		t.Error("expected different keys for different mnemonics")
	}
}
