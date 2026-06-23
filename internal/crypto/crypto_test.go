package crypto

import (
	"bytes"
	"testing"
)

func TestCryptoFlow(t *testing.T) {
	// 1. Generate key pair
	priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	// 2. PEM Serialization/Deserialization
	privPEM := EncodePrivateKeyToPEM(priv)
	pubPEM, err := EncodePublicKeyToPEM(&priv.PublicKey)
	if err != nil {
		t.Fatalf("failed to encode public key to PEM: %v", err)
	}

	parsedPriv, err := ParsePrivateKeyFromPEM(privPEM)
	if err != nil {
		t.Fatalf("failed to parse private key from PEM: %v", err)
	}

	parsedPub, err := ParsePublicKeyFromPEM(pubPEM)
	if err != nil {
		t.Fatalf("failed to parse public key from PEM: %v", err)
	}

	// 3. Test Signing & Verification
	message := []byte("Hello, Magicbox P2P!")
	signature, err := Sign(parsedPriv, message)
	if err != nil {
		t.Fatalf("failed to sign message: %v", err)
	}

	err = Verify(parsedPub, message, signature)
	if err != nil {
		t.Errorf("failed to verify valid signature: %v", err)
	}

	// Test invalid message verification
	err = Verify(parsedPub, []byte("Wrong message"), signature)
	if err == nil {
		t.Errorf("expected verification to fail for invalid message")
	}

	// 4. Test Encryption & Decryption
	secretPayload := []byte("This is a secret federated message.")
	ciphertext, err := Encrypt(parsedPub, secretPayload)
	if err != nil {
		t.Fatalf("failed to encrypt payload: %v", err)
	}

	decrypted, err := Decrypt(parsedPriv, ciphertext)
	if err != nil {
		t.Fatalf("failed to decrypt payload: %v", err)
	}

	if !bytes.Equal(secretPayload, decrypted) {
		t.Errorf("decrypted payload (%q) does not match original (%q)", decrypted, secretPayload)
	}
}
