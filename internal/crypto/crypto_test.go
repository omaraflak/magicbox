package crypto

import (
	"bytes"
	"testing"
)

func TestGenerateKeyPair_Success(t *testing.T) {
	priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}
	if priv == nil {
		t.Fatal("expected private key to be generated, got nil")
	}
	if priv.Validate() != nil {
		t.Errorf("generated private key is invalid")
	}
}

func TestPEMSerialization_Success(t *testing.T) {
	priv, _ := GenerateKeyPair()

	privPEM := EncodePrivateKeyToPEM(priv)
	pubPEM, err := EncodePublicKeyToPEM(&priv.PublicKey)
	if err != nil {
		t.Fatalf("failed to encode public key to PEM: %v", err)
	}

	parsedPriv, err := ParsePrivateKeyFromPEM(privPEM)
	if err != nil {
		t.Fatalf("failed to parse private key from PEM: %v", err)
	}
	if parsedPriv.D.Cmp(priv.D) != 0 {
		t.Errorf("parsed private key D value does not match original D value")
	}

	parsedPub, err := ParsePublicKeyFromPEM(pubPEM)
	if err != nil {
		t.Fatalf("failed to parse public key from PEM: %v", err)
	}
	if parsedPub.N.Cmp(priv.PublicKey.N) != 0 {
		t.Errorf("parsed public key N value does not match original N value")
	}
}

func TestSignAndVerify_Success(t *testing.T) {
	priv, _ := GenerateKeyPair()
	message := []byte("Hello, Magicbox P2P!")

	signature, err := Sign(priv, message)
	if err != nil {
		t.Fatalf("failed to sign message: %v", err)
	}

	err = Verify(&priv.PublicKey, message, signature)
	if err != nil {
		t.Errorf("failed to verify valid signature: %v", err)
	}
}

func TestSignAndVerify_InvalidMessageFails(t *testing.T) {
	priv, _ := GenerateKeyPair()
	message := []byte("Hello, Magicbox P2P!")

	signature, _ := Sign(priv, message)

	err := Verify(&priv.PublicKey, []byte("Wrong message"), signature)
	if err == nil {
		t.Errorf("expected verification to fail for invalid message")
	}
}

func TestEncryptAndDecrypt_Success(t *testing.T) {
	priv, _ := GenerateKeyPair()
	secretPayload := []byte("This is a secret federated message.")

	ciphertext, err := Encrypt(&priv.PublicKey, secretPayload)
	if err != nil {
		t.Fatalf("failed to encrypt payload: %v", err)
	}

	decrypted, err := Decrypt(priv, ciphertext)
	if err != nil {
		t.Fatalf("failed to decrypt payload: %v", err)
	}

	if !bytes.Equal(secretPayload, decrypted) {
		t.Errorf("decrypted payload (%q) does not match original (%q)", decrypted, secretPayload)
	}
}
