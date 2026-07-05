package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

// Sign signs arbitrary data using the Ed25519 private key.
func Sign(priv ed25519.PrivateKey, data []byte) []byte {
	return ed25519.Sign(priv, data)
}

// Verify checks if the signature is valid for the given data and Ed25519 public key.
func Verify(pub ed25519.PublicKey, data, sig []byte) bool {
	return ed25519.Verify(pub, data, sig)
}

// newAESGCM derives an AES-256-GCM cipher from an ECDH shared secret.
func newAESGCM(sharedSecret []byte) (cipher.AEAD, error) {
	aesKey := sha256.Sum256(sharedSecret)
	block, err := aes.NewCipher(aesKey[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// EncryptECDH encrypts data using recipient's X25519 public key and AES-256-GCM.
func EncryptECDH(recipientPub *ecdh.PublicKey, data []byte) (ephemeralPubBytes, iv, ciphertext []byte, err error) {
	curve := ecdh.X25519()
	ephemeralPriv, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, nil, err
	}
	ephemeralPubBytes = ephemeralPriv.PublicKey().Bytes()

	sharedSecret, err := ephemeralPriv.ECDH(recipientPub)
	if err != nil {
		return nil, nil, nil, err
	}

	aesGCM, err := newAESGCM(sharedSecret)
	if err != nil {
		return nil, nil, nil, err
	}

	iv = make([]byte, 12)
	if _, err = rand.Read(iv); err != nil {
		return nil, nil, nil, err
	}

	ciphertext = aesGCM.Seal(nil, iv, data, nil)
	return ephemeralPubBytes, iv, ciphertext, nil
}

// DecryptECDH decrypts data using recipient's X25519 private key and sender's ephemeral public key.
func DecryptECDH(recipientPriv *ecdh.PrivateKey, ephemeralPubBytes, iv, ciphertext []byte) ([]byte, error) {
	curve := ecdh.X25519()
	ephemeralPub, err := curve.NewPublicKey(ephemeralPubBytes)
	if err != nil {
		return nil, err
	}

	sharedSecret, err := recipientPriv.ECDH(ephemeralPub)
	if err != nil {
		return nil, err
	}

	aesGCM, err := newAESGCM(sharedSecret)
	if err != nil {
		return nil, err
	}

	return aesGCM.Open(nil, iv, ciphertext, nil)
}

// EncryptedEnvelope represents a signed, ECDH-encrypted package.
type EncryptedEnvelope struct {
	Version      string `json:"version"`
	EphemeralPub []byte `json:"ephemeral_pub"`
	IV           []byte `json:"iv"`
	Ciphertext   []byte `json:"ciphertext"`
	Signature    []byte `json:"signature"`
}

// envelopeSignedData concatenates the cryptographic material that is covered by
// the envelope signature: ephemeralPub || iv || ciphertext.
func envelopeSignedData(ephemeralPub, iv, ciphertext []byte) []byte {
	data := make([]byte, 0, len(ephemeralPub)+len(iv)+len(ciphertext))
	data = append(data, ephemeralPub...)
	data = append(data, iv...)
	data = append(data, ciphertext...)
	return data
}

// EncryptAndSign wraps the data payload in a signed, hybrid-encrypted envelope.
// The signature covers ephemeralPub + iv + ciphertext to prevent any field from
// being swapped without detection.
func EncryptAndSign(senderPriv ed25519.PrivateKey, recipientPub *ecdh.PublicKey, data []byte) ([]byte, error) {
	ephemeralPubBytes, iv, ciphertext, err := EncryptECDH(recipientPub, data)
	if err != nil {
		return nil, err
	}

	// Sign over ALL cryptographic material, not just ciphertext
	signed := envelopeSignedData(ephemeralPubBytes, iv, ciphertext)
	signature := Sign(senderPriv, signed)

	envelope := EncryptedEnvelope{
		Version:      "1.0",
		EphemeralPub: ephemeralPubBytes,
		IV:           iv,
		Ciphertext:   ciphertext,
		Signature:    signature,
	}

	return json.Marshal(envelope)
}

// DecryptAndVerify decrypts and verifies the signed, hybrid-encrypted envelope.
func DecryptAndVerify(recipientPriv *ecdh.PrivateKey, senderPub ed25519.PublicKey, envelopeBytes []byte) ([]byte, error) {
	var envelope EncryptedEnvelope
	if err := json.Unmarshal(envelopeBytes, &envelope); err != nil {
		return nil, fmt.Errorf("failed to unmarshal envelope: %w", err)
	}

	// Verify signature over ALL cryptographic material
	signed := envelopeSignedData(envelope.EphemeralPub, envelope.IV, envelope.Ciphertext)
	if !Verify(senderPub, signed, envelope.Signature) {
		return nil, fmt.Errorf("invalid signature")
	}

	return DecryptECDH(recipientPriv, envelope.EphemeralPub, envelope.IV, envelope.Ciphertext)
}
