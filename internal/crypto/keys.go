package crypto

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"
)

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
