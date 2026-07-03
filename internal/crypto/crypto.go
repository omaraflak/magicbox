package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"

	"github.com/tyler-smith/go-bip39"
)

// GenerateMnemonic generates a new 12-word BIP-39 mnemonic phrase.
func GenerateMnemonic() (string, error) {
	entropy, err := bip39.NewEntropy(128)
	if err != nil {
		return "", fmt.Errorf("failed to generate entropy: %w", err)
	}
	return bip39.NewMnemonic(entropy)
}

// mnemonicToSeed validates a BIP-39 mnemonic and returns the derived seed.
func mnemonicToSeed(mnemonic string) ([]byte, error) {
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, fmt.Errorf("invalid mnemonic phrase")
	}

	seed := bip39.NewSeed(mnemonic, "")
	if len(seed) < 64 {
		return nil, fmt.Errorf("invalid seed derived from mnemonic")
	}

	return seed, nil
}

// DeriveIdentityKey derives a deterministic Ed25519 private key from a mnemonic at a specific index.
// The key is derived via SHA256(MasterSeed || "/ed25519/{index}").
func DeriveIdentityKey(mnemonic string, index int) (ed25519.PrivateKey, error) {
	seed, err := mnemonicToSeed(mnemonic)
	if err != nil {
		return nil, err
	}

	h := sha256.New()
	h.Write(seed)
	h.Write([]byte(fmt.Sprintf("/ed25519/%d", index)))
	edSeed := h.Sum(nil)
	return ed25519.NewKeyFromSeed(edSeed), nil
}

// DeriveEncryptionKey derives a deterministic X25519 private key from a mnemonic at a specific index.
// The key is derived via SHA256(MasterSeed || "/x25519/{index}").
func DeriveEncryptionKey(mnemonic string, index int) (*ecdh.PrivateKey, error) {
	seed, err := mnemonicToSeed(mnemonic)
	if err != nil {
		return nil, err
	}

	h := sha256.New()
	h.Write(seed)
	h.Write([]byte(fmt.Sprintf("/x25519/%d", index)))
	xSeed := h.Sum(nil)
	curve := ecdh.X25519()
	xPriv, err := curve.NewPrivateKey(xSeed)
	if err != nil {
		return nil, fmt.Errorf("failed to generate X25519 key: %w", err)
	}

	return xPriv, nil
}

// MarshalPrivateKey PEM-encodes an Ed25519 or X25519 private key using PKCS#8.
func MarshalPrivateKey(priv interface{}) ([]byte, error) {
	derBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, err
	}
	block := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: derBytes,
	}
	return pem.EncodeToMemory(block), nil
}

// UnmarshalEd25519PrivateKey decodes an Ed25519 private key from PEM bytes.
func UnmarshalEd25519PrivateKey(pemBytes []byte) (ed25519.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("invalid private key PEM")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	edPriv, ok := parsed.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an Ed25519 private key")
	}
	return edPriv, nil
}

// UnmarshalX25519PrivateKey decodes an X25519 private key from PEM bytes.
func UnmarshalX25519PrivateKey(pemBytes []byte) (*ecdh.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("invalid private key PEM")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	xPriv, ok := parsed.(*ecdh.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an X25519 private key")
	}
	return xPriv, nil
}

// MarshalPublicKey PEM-encodes an Ed25519 or X25519 public key using PKIX.
func MarshalPublicKey(pub interface{}) ([]byte, error) {
	derBytes, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, err
	}
	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: derBytes,
	}
	return pem.EncodeToMemory(block), nil
}

// UnmarshalEd25519PublicKey decodes an Ed25519 public key from PEM bytes.
func UnmarshalEd25519PublicKey(pemBytes []byte) (ed25519.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("invalid public key PEM")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	edPub, ok := parsed.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an Ed25519 public key")
	}
	return edPub, nil
}

// UnmarshalX25519PublicKey decodes an X25519 public key from PEM bytes.
func UnmarshalX25519PublicKey(pemBytes []byte) (*ecdh.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("invalid public key PEM")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	xPub, ok := parsed.(*ecdh.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an X25519 public key")
	}
	return xPub, nil
}

// Sign signs arbitrary data using the Ed25519 private key.
func Sign(priv ed25519.PrivateKey, data []byte) []byte {
	return ed25519.Sign(priv, data)
}

// Verify checks if the signature is valid for the given data and Ed25519 public key.
func Verify(pub ed25519.PublicKey, data, sig []byte) bool {
	return ed25519.Verify(pub, data, sig)
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

	aesKey := sha256.Sum256(sharedSecret)

	iv = make([]byte, 12)
	if _, err = rand.Read(iv); err != nil {
		return nil, nil, nil, err
	}

	block, err := aes.NewCipher(aesKey[:])
	if err != nil {
		return nil, nil, nil, err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
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

	aesKey := sha256.Sum256(sharedSecret)

	block, err := aes.NewCipher(aesKey[:])
	if err != nil {
		return nil, err
	}
	aesGCM, err := cipher.NewGCM(block)
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

// EncryptAndSign wraps the data payload in a signed, hybrid-encrypted envelope.
// The signature covers ephemeralPub + iv + ciphertext to prevent any field from
// being swapped without detection.
func EncryptAndSign(senderPriv ed25519.PrivateKey, recipientPub *ecdh.PublicKey, data []byte) ([]byte, error) {
	ephemeralPubBytes, iv, ciphertext, err := EncryptECDH(recipientPub, data)
	if err != nil {
		return nil, err
	}

	// Sign over ALL cryptographic material, not just ciphertext
	signed := make([]byte, 0, len(ephemeralPubBytes)+len(iv)+len(ciphertext))
	signed = append(signed, ephemeralPubBytes...)
	signed = append(signed, iv...)
	signed = append(signed, ciphertext...)
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
	signed := make([]byte, 0, len(envelope.EphemeralPub)+len(envelope.IV)+len(envelope.Ciphertext))
	signed = append(signed, envelope.EphemeralPub...)
	signed = append(signed, envelope.IV...)
	signed = append(signed, envelope.Ciphertext...)
	if !Verify(senderPub, signed, envelope.Signature) {
		return nil, fmt.Errorf("invalid signature")
	}

	return DecryptECDH(recipientPriv, envelope.EphemeralPub, envelope.IV, envelope.Ciphertext)
}
