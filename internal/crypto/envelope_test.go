package crypto

import (
	"bytes"
	"crypto/ed25519"
	"testing"
)

func TestSignAndVerify_Success(t *testing.T) {
	mnemonic, _ := GenerateMnemonic()
	edPriv, _ := DeriveIdentityKey(mnemonic, 0)
	edPub := edPriv.Public().(ed25519.PublicKey)
	message := []byte("Hello, Magicbox P2P!")

	sig := Sign(edPriv, message)
	if !Verify(edPub, message, sig) {
		t.Error("failed to verify valid signature")
	}
}

func TestSignAndVerify_InvalidMessageFails(t *testing.T) {
	mnemonic, _ := GenerateMnemonic()
	edPriv, _ := DeriveIdentityKey(mnemonic, 0)
	edPub := edPriv.Public().(ed25519.PublicKey)
	message := []byte("Hello, Magicbox P2P!")

	sig := Sign(edPriv, message)
	if Verify(edPub, []byte("Wrong message"), sig) {
		t.Error("verification should have failed for wrong message")
	}
}

func TestSignAndVerify_EmptyMessage(t *testing.T) {
	mnemonic, _ := GenerateMnemonic()
	edPriv, _ := DeriveIdentityKey(mnemonic, 0)
	edPub := edPriv.Public().(ed25519.PublicKey)

	sig := Sign(edPriv, []byte{})
	if !Verify(edPub, []byte{}, sig) {
		t.Error("failed to verify signature on empty message")
	}
}

func TestEncryptAndDecryptECDH_Success(t *testing.T) {
	mnemonic, _ := GenerateMnemonic()
	xPriv, _ := DeriveEncryptionKey(mnemonic, 0)
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

func TestEncryptAndDecryptECDH_EmptyData(t *testing.T) {
	mnemonic, _ := GenerateMnemonic()
	xPriv, _ := DeriveEncryptionKey(mnemonic, 0)
	xPub := xPriv.PublicKey()

	ephemeralPubBytes, iv, ciphertext, err := EncryptECDH(xPub, []byte{})
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	decrypted, err := DecryptECDH(xPriv, ephemeralPubBytes, iv, ciphertext)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if len(decrypted) != 0 {
		t.Errorf("expected empty decrypted payload, got %q", decrypted)
	}
}

func TestDecryptECDH_CorruptedCiphertextFails(t *testing.T) {
	mnemonic, _ := GenerateMnemonic()
	xPriv, _ := DeriveEncryptionKey(mnemonic, 0)
	xPub := xPriv.PublicKey()

	ephemeralPubBytes, iv, ciphertext, err := EncryptECDH(xPub, []byte("secret"))
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	// Corrupt ciphertext
	ciphertext[0] ^= 0xFF

	_, err = DecryptECDH(xPriv, ephemeralPubBytes, iv, ciphertext)
	if err == nil {
		t.Error("expected error for corrupted ciphertext, got nil")
	}
}

func TestDecryptECDH_WrongKeyFails(t *testing.T) {
	mnemonic1, _ := GenerateMnemonic()
	mnemonic2, _ := GenerateMnemonic()
	xPriv1, _ := DeriveEncryptionKey(mnemonic1, 0)
	xPriv2, _ := DeriveEncryptionKey(mnemonic2, 0)

	ephemeralPub, iv, ciphertext, _ := EncryptECDH(xPriv1.PublicKey(), []byte("secret"))

	_, err := DecryptECDH(xPriv2, ephemeralPub, iv, ciphertext)
	if err == nil {
		t.Error("expected error when decrypting with wrong key, got nil")
	}
}

func TestEncryptAndSign_Success(t *testing.T) {
	mnemonicSender, _ := GenerateMnemonic()
	mnemonicRecipient, _ := GenerateMnemonic()

	edPrivSender, _ := DeriveIdentityKey(mnemonicSender, 0)
	edPubSender := edPrivSender.Public().(ed25519.PublicKey)

	xPrivRecipient, _ := DeriveEncryptionKey(mnemonicRecipient, 0)
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

	edPrivSender, _ := DeriveIdentityKey(mnemonicSender, 0)
	xPrivRecipient, _ := DeriveEncryptionKey(mnemonicRecipient, 0)
	xPubRecipient := xPrivRecipient.PublicKey()

	edPrivWrongSender, _ := DeriveIdentityKey(mnemonicWrongSender, 0)
	edPubWrongSender := edPrivWrongSender.Public().(ed25519.PublicKey)

	secretPayload := []byte("ECIES secured e2e message payload.")

	envelopeBytes, _ := EncryptAndSign(edPrivSender, xPubRecipient, secretPayload)

	_, err := DecryptAndVerify(xPrivRecipient, edPubWrongSender, envelopeBytes)
	if err == nil {
		t.Error("expected error due to invalid signature, got nil")
	}
}

func TestDecryptAndVerify_CorruptedEnvelopeFails(t *testing.T) {
	_, err := DecryptAndVerify(nil, nil, []byte("not valid json"))
	if err == nil {
		t.Error("expected error for corrupted envelope JSON, got nil")
	}
}

func TestEnvelopeSignedData_Concatenation(t *testing.T) {
	eph := []byte{1, 2, 3}
	iv := []byte{4, 5}
	ct := []byte{6, 7, 8, 9}

	result := envelopeSignedData(eph, iv, ct)
	expected := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}
	if !bytes.Equal(result, expected) {
		t.Errorf("envelopeSignedData = %v, want %v", result, expected)
	}
}
