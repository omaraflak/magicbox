package crypto

import (
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
