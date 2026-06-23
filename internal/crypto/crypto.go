package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

// GenerateKeyPair generates a new RSA 4096-bit private and public key pair.
func GenerateKeyPair() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, 4096)
}

// EncodePrivateKeyToPEM serializes an RSA private key to PEM format.
func EncodePrivateKeyToPEM(priv *rsa.PrivateKey) []byte {
	privBytes := x509.MarshalPKCS1PrivateKey(priv)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	}
	return pem.EncodeToMemory(block)
}

// EncodePublicKeyToPEM serializes an RSA public key to PEM format.
func EncodePublicKeyToPEM(pub *rsa.PublicKey) ([]byte, error) {
	pubBytes, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, err
	}
	block := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pubBytes,
	}
	return pem.EncodeToMemory(block), nil
}

// ParsePrivateKeyFromPEM deserializes an RSA private key from PEM bytes.
func ParsePrivateKeyFromPEM(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, fmt.Errorf("invalid private key PEM")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

// ParsePublicKeyFromPEM deserializes an RSA public key from PEM bytes.
func ParsePublicKeyFromPEM(pemBytes []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "RSA PUBLIC KEY" {
		return nil, fmt.Errorf("invalid public key PEM")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}
	return rsaPub, nil
}

// Sign signs arbitrary data using the private key.
func Sign(priv *rsa.PrivateKey, data []byte) ([]byte, error) {
	hashed := sha256.Sum256(data)
	return rsa.SignPSS(rand.Reader, priv, crypto.SHA256, hashed[:], nil)
}

// Verify checks if the signature is valid for the given data and public key.
func Verify(pub *rsa.PublicKey, data, sig []byte) error {
	hashed := sha256.Sum256(data)
	return rsa.VerifyPSS(pub, crypto.SHA256, hashed[:], sig, nil)
}

// Encrypt encrypts arbitrary data using the public key.
func Encrypt(pub *rsa.PublicKey, data []byte) ([]byte, error) {
	return rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, data, nil)
}

// Decrypt decrypts encrypted data using the private key.
func Decrypt(priv *rsa.PrivateKey, ciphertext []byte) ([]byte, error) {
	return rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, ciphertext, nil)
}
