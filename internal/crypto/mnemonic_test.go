package crypto

import (
	"strings"
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

func TestGenerateMnemonic_Returns12Words(t *testing.T) {
	mnemonic, err := GenerateMnemonic()
	if err != nil {
		t.Fatalf("failed to generate mnemonic: %v", err)
	}
	words := strings.Fields(mnemonic)
	if len(words) != 12 {
		t.Errorf("expected 12 words, got %d", len(words))
	}
}

func TestGenerateMnemonic_UniqueMnemonics(t *testing.T) {
	m1, _ := GenerateMnemonic()
	m2, _ := GenerateMnemonic()
	if m1 == m2 {
		t.Error("expected two generated mnemonics to differ")
	}
}
