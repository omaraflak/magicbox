package crypto

import (
	"bytes"
	"crypto/ed25519"
	"testing"
)

func TestGenerateMnemonic_Success(t *testing.T) {
	mnemonic, err := GenerateMnemonic()
	if err != nil {
		t.Fatalf("failed to generate mnemonic: %v", err)
	}
	if len(mnemonic) == 0 {
		t.Error("expected non-empty mnemonic")
	}
}

func TestDeriveKeys_Success(t *testing.T) {
	mnemonic, _ := GenerateMnemonic()

	edPriv1, xPriv1, err := DeriveKeys(mnemonic, 0)
	if err != nil {
		t.Fatalf("failed to derive keys: %v", err)
	}

	edPriv2, xPriv2, err := DeriveKeys(mnemonic, 0)
	if err != nil {
		t.Fatalf("failed to derive keys 2: %v", err)
	}

	// Assert determinism
	if !bytes.Equal(edPriv1, edPriv2) {
		t.Error("deterministic Ed25519 keys do not match")
	}
	if !bytes.Equal(xPriv1.Bytes(), xPriv2.Bytes()) {
		t.Error("deterministic X25519 keys do not match")
	}
}

func TestPEMSerialization_Success(t *testing.T) {
	mnemonic, _ := GenerateMnemonic()
	edPriv, xPriv, _ := DeriveKeys(mnemonic, 0)

	// 1. Ed25519 Private Key PEM
	edPrivPEM, err := MarshalPrivateKey(edPriv)
	if err != nil {
		t.Fatalf("failed to marshal Ed25519 private key: %v", err)
	}
	parsedEdPriv, err := UnmarshalEd25519PrivateKey(edPrivPEM)
	if err != nil {
		t.Fatalf("failed to unmarshal Ed25519 private key: %v", err)
	}
	if !bytes.Equal(edPriv, parsedEdPriv) {
		t.Error("Ed25519 private keys do not match after PEM roundtrip")
	}

	// 2. Ed25519 Public Key PEM
	edPub := edPriv.Public().(ed25519.PublicKey)
	edPubPEM, err := MarshalPublicKey(edPub)
	if err != nil {
		t.Fatalf("failed to marshal Ed25519 public key: %v", err)
	}
	parsedEdPub, err := UnmarshalEd25519PublicKey(edPubPEM)
	if err != nil {
		t.Fatalf("failed to unmarshal Ed25519 public key: %v", err)
	}
	if !bytes.Equal(edPub, parsedEdPub) {
		t.Error("Ed25519 public keys do not match after PEM roundtrip")
	}

	// 3. X25519 Private Key PEM
	xPrivPEM, err := MarshalPrivateKey(xPriv)
	if err != nil {
		t.Fatalf("failed to marshal X25519 private key: %v", err)
	}
	parsedXPriv, err := UnmarshalX25519PrivateKey(xPrivPEM)
	if err != nil {
		t.Fatalf("failed to unmarshal X25519 private key: %v", err)
	}
	if !bytes.Equal(xPriv.Bytes(), parsedXPriv.Bytes()) {
		t.Error("X25519 private keys do not match after PEM roundtrip")
	}

	// 4. X25519 Public Key PEM
	xPub := xPriv.PublicKey()
	xPubPEM, err := MarshalPublicKey(xPub)
	if err != nil {
		t.Fatalf("failed to marshal X25519 public key: %v", err)
	}
	parsedXPub, err := UnmarshalX25519PublicKey(xPubPEM)
	if err != nil {
		t.Fatalf("failed to unmarshal X25519 public key: %v", err)
	}
	if !bytes.Equal(xPub.Bytes(), parsedXPub.Bytes()) {
		t.Error("X25519 public keys do not match after PEM roundtrip")
	}
}

func TestSignAndVerify_Success(t *testing.T) {
	mnemonic, _ := GenerateMnemonic()
	edPriv, _, _ := DeriveKeys(mnemonic, 0)
	edPub := edPriv.Public().(ed25519.PublicKey)
	message := []byte("Hello, Magicbox P2P!")

	sig := Sign(edPriv, message)
	if !Verify(edPub, message, sig) {
		t.Error("failed to verify valid signature")
	}
}

func TestSignAndVerify_InvalidMessageFails(t *testing.T) {
	mnemonic, _ := GenerateMnemonic()
	edPriv, _, _ := DeriveKeys(mnemonic, 0)
	edPub := edPriv.Public().(ed25519.PublicKey)
	message := []byte("Hello, Magicbox P2P!")

	sig := Sign(edPriv, message)
	if Verify(edPub, []byte("Wrong message"), sig) {
		t.Error("verification should have failed for wrong message")
	}
}

func TestEncryptAndDecryptECDH_Success(t *testing.T) {
	mnemonic, _ := GenerateMnemonic()
	_, xPriv, _ := DeriveKeys(mnemonic, 0)
	xPub := xPriv.PublicKey()
	message := []byte("Secret E2E message payload.")

	ephemeralPubBytes, iv, ciphertext, err := EncryptECDH(xPub, message)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	decrypted, err := DecryptECDH(xPriv, ephemeralPubBytes, iv, ciphertext)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if !bytes.Equal(message, decrypted) {
		t.Errorf("decrypted payload (%q) does not match original (%q)", decrypted, message)
	}
}

func TestEncryptAndSign_Success(t *testing.T) {
	mnemonicSender, _ := GenerateMnemonic()
	mnemonicRecipient, _ := GenerateMnemonic()

	edPrivSender, _, _ := DeriveKeys(mnemonicSender, 0)
	edPubSender := edPrivSender.Public().(ed25519.PublicKey)

	_, xPrivRecipient, _ := DeriveKeys(mnemonicRecipient, 0)
	xPubRecipient := xPrivRecipient.PublicKey()

	secretPayload := []byte("ECIES secured e2e message payload.")

	envelopeBytes, err := EncryptAndSign(edPrivSender, xPubRecipient, secretPayload)
	if err != nil {
		t.Fatalf("failed to encrypt and sign: %v", err)
	}

	decrypted, err := DecryptAndVerify(xPrivRecipient, edPubSender, envelopeBytes)
	if err != nil {
		t.Fatalf("failed to decrypt and verify: %v", err)
	}

	if !bytes.Equal(secretPayload, decrypted) {
		t.Errorf("decrypted payload (%q) does not match original (%q)", decrypted, secretPayload)
	}
}

func TestEncryptAndSign_InvalidSignatureFails(t *testing.T) {
	mnemonicSender, _ := GenerateMnemonic()
	mnemonicRecipient, _ := GenerateMnemonic()
	mnemonicWrongSender, _ := GenerateMnemonic()

	edPrivSender, _, _ := DeriveKeys(mnemonicSender, 0)
	_, xPrivRecipient, _ := DeriveKeys(mnemonicRecipient, 0)
	xPubRecipient := xPrivRecipient.PublicKey()

	edPrivWrongSender, _, _ := DeriveKeys(mnemonicWrongSender, 0)
	edPubWrongSender := edPrivWrongSender.Public().(ed25519.PublicKey)

	secretPayload := []byte("ECIES secured e2e message payload.")

	envelopeBytes, _ := EncryptAndSign(edPrivSender, xPubRecipient, secretPayload)

	_, err := DecryptAndVerify(xPrivRecipient, edPubWrongSender, envelopeBytes)
	if err == nil {
		t.Error("expected error due to invalid signature, got nil")
	}
}

func TestDeriveKeys_InvalidMnemonicFails(t *testing.T) {
	_, _, err := DeriveKeys("invalid mnemonic sentence that fails checksum completely", 0)
	if err == nil {
		t.Error("expected error for invalid mnemonic phrase, got nil")
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

func TestDeriveKeys_DifferentIndicesProduceDifferentKeys(t *testing.T) {
	mnemonic, _ := GenerateMnemonic()

	edPriv1, xPriv1, err := DeriveKeys(mnemonic, 0)
	if err != nil {
		t.Fatalf("failed to derive keys at index 0: %v", err)
	}

	edPriv2, xPriv2, err := DeriveKeys(mnemonic, 1)
	if err != nil {
		t.Fatalf("failed to derive keys at index 1: %v", err)
	}

	if bytes.Equal(edPriv1, edPriv2) {
		t.Error("expected different Ed25519 keys for different indices, but they are the same")
	}
	if bytes.Equal(xPriv1.Bytes(), xPriv2.Bytes()) {
		t.Error("expected different X25519 keys for different indices, but they are the same")
	}
}
