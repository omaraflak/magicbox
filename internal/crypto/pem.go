package crypto

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

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
