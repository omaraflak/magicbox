package keymanager

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/magicbox/core/internal/crypto"
)

func TestLoadOrGenerate_CreatesKeysAndMnemonic(t *testing.T) {
	tempDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tempDir, "core"), 0750)

	paths := NewKeyPaths(tempDir)
	keys, err := LoadOrGenerate(paths)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(keys.MasterPublicKeyPEM) == 0 || len(keys.PrivateKeyPEM) == 0 || len(keys.PublicKeyPEM) == 0 || len(keys.EncryptionKeyPEM) == 0 || len(keys.EncryptionPubPEM) == 0 {
		t.Errorf("expected all keys (including master public key) to be generated and populated")
	}

	if keys.IdentityKeyIndex != 1 || keys.EncryptionKeyIndex != 1 {
		t.Errorf("expected initial indices to be 1, got identity=%d, encryption=%d", keys.IdentityKeyIndex, keys.EncryptionKeyIndex)
	}

	if keys.Mnemonic == "" {
		t.Errorf("expected generated mnemonic to be returned, got empty string")
	}

	// Verify key files exist, but mnemonic file does NOT exist
	coreDir := filepath.Join(tempDir, "core")
	if _, err := os.Stat(filepath.Join(coreDir, "mnemonic")); err == nil || !os.IsNotExist(err) {
		t.Errorf("expected mnemonic file to NOT exist on disk")
	}
	if _, err := os.Stat(paths.MasterIdentityPubPath); os.IsNotExist(err) {
		t.Errorf("expected master identity public key file to exist")
	}
	if _, err := os.Stat(paths.IdentityKeyPath); os.IsNotExist(err) {
		t.Errorf("expected identity private key file to exist")
	}
	if _, err := os.Stat(paths.IdentityPubPath); os.IsNotExist(err) {
		t.Errorf("expected identity public key file to exist")
	}
	if _, err := os.Stat(paths.EncryptionKeyPath); os.IsNotExist(err) {
		t.Errorf("expected encryption private key file to exist")
	}
	if _, err := os.Stat(paths.EncryptionPubPath); os.IsNotExist(err) {
		t.Errorf("expected encryption public key file to exist")
	}
}

func TestLoadOrGenerate_ReadsExistingKeys(t *testing.T) {
	tempDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tempDir, "core"), 0750)

	paths := NewKeyPaths(tempDir)
	keys1, _ := LoadOrGenerate(paths)

	// Call again, should read existing and return empty mnemonic
	keys2, err := LoadOrGenerate(paths)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if keys2.Mnemonic != "" {
		t.Errorf("expected empty mnemonic on reload, got %q", keys2.Mnemonic)
	}
	if keys1.Mnemonic == "" {
		t.Errorf("expected mnemonic1 to not be empty")
	}

	if !bytes.Equal(keys1.MasterPublicKeyPEM, keys2.MasterPublicKeyPEM) {
		t.Errorf("expected master public keys to be identical")
	}
	if !bytes.Equal(keys1.PrivateKeyPEM, keys2.PrivateKeyPEM) {
		t.Errorf("expected identity private keys to be identical")
	}
	if !bytes.Equal(keys1.PublicKeyPEM, keys2.PublicKeyPEM) {
		t.Errorf("expected identity public keys to be identical")
	}
	if !bytes.Equal(keys1.EncryptionKeyPEM, keys2.EncryptionKeyPEM) {
		t.Errorf("expected encryption private keys to be identical")
	}
	if !bytes.Equal(keys1.EncryptionPubPEM, keys2.EncryptionPubPEM) {
		t.Errorf("expected encryption public keys to be identical")
	}
}

func TestRecoverAll_Success(t *testing.T) {
	tempDir := t.TempDir()
	coreDir := filepath.Join(tempDir, "core")
	_ = os.MkdirAll(coreDir, 0750)

	// Generate a valid mnemonic.
	mnemonic, err := crypto.GenerateMnemonic()
	if err != nil {
		t.Fatalf("failed to generate mnemonic: %v", err)
	}

	paths := NewKeyPaths(tempDir)

	// Call RecoverAll.
	if err := RecoverAll(paths, mnemonic, 1, 1); err != nil {
		t.Fatalf("RecoverAll returned error: %v", err)
	}

	// Read back key files and compare with independent derivation.
	masterPriv, err := crypto.DeriveIdentityKey(mnemonic, 0)
	if err != nil {
		t.Fatalf("failed to derive master identity key: %v", err)
	}
	edPriv, err := crypto.DeriveIdentityKey(mnemonic, 1)
	if err != nil {
		t.Fatalf("failed to derive identity key: %v", err)
	}
	xPriv, err := crypto.DeriveEncryptionKey(mnemonic, 1)
	if err != nil {
		t.Fatalf("failed to derive encryption key: %v", err)
	}

	wantMasterPubPEM, _ := crypto.MarshalPublicKey(masterPriv.Public())
	wantPrivPEM, _ := crypto.MarshalPrivateKey(edPriv)
	wantPubPEM, _ := crypto.MarshalPublicKey(edPriv.Public())
	wantEncKeyPEM, _ := crypto.MarshalPrivateKey(xPriv)
	wantEncPubPEM, _ := crypto.MarshalPublicKey(xPriv.PublicKey())

	gotMasterPubPEM, _ := os.ReadFile(paths.MasterIdentityPubPath)
	gotPrivPEM, _ := os.ReadFile(paths.IdentityKeyPath)
	gotPubPEM, _ := os.ReadFile(paths.IdentityPubPath)
	gotEncKeyPEM, _ := os.ReadFile(paths.EncryptionKeyPath)
	gotEncPubPEM, _ := os.ReadFile(paths.EncryptionPubPath)

	if !bytes.Equal(wantMasterPubPEM, gotMasterPubPEM) {
		t.Errorf("master public key mismatch")
	}
	if !bytes.Equal(wantPrivPEM, gotPrivPEM) {
		t.Errorf("identity private key mismatch")
	}
	if !bytes.Equal(wantPubPEM, gotPubPEM) {
		t.Errorf("identity public key mismatch")
	}
	if !bytes.Equal(wantEncKeyPEM, gotEncKeyPEM) {
		t.Errorf("encryption private key mismatch")
	}
	if !bytes.Equal(wantEncPubPEM, gotEncPubPEM) {
		t.Errorf("encryption public key mismatch")
	}
}

func TestRecoverAll_InvalidMnemonicFails(t *testing.T) {
	tempDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tempDir, "core"), 0750)

	paths := NewKeyPaths(tempDir)
	err := RecoverAll(paths, "invalid mnemonic phrase", 0, 0)
	if err == nil {
		t.Error("expected error for invalid mnemonic, got nil")
	}
}

func TestRecoverAll_CustomIndexSuccess(t *testing.T) {
	tempDir := t.TempDir()
	coreDir := filepath.Join(tempDir, "core")
	_ = os.MkdirAll(coreDir, 0750)

	mnemonic, err := crypto.GenerateMnemonic()
	if err != nil {
		t.Fatalf("failed to generate mnemonic: %v", err)
	}

	paths := NewKeyPaths(tempDir)

	if err := RecoverAll(paths, mnemonic, 3, 3); err != nil {
		t.Fatalf("RecoverAll returned error: %v", err)
	}

	// Read back key files and compare with independent derivation.
	masterPriv, err := crypto.DeriveIdentityKey(mnemonic, 0)
	if err != nil {
		t.Fatalf("failed to derive master identity key: %v", err)
	}
	edPriv, err := crypto.DeriveIdentityKey(mnemonic, 3)
	if err != nil {
		t.Fatalf("failed to derive identity key: %v", err)
	}
	xPriv, err := crypto.DeriveEncryptionKey(mnemonic, 3)
	if err != nil {
		t.Fatalf("failed to derive encryption key: %v", err)
	}

	wantMasterPubPEM, _ := crypto.MarshalPublicKey(masterPriv.Public())
	wantPrivPEM, _ := crypto.MarshalPrivateKey(edPriv)
	wantPubPEM, _ := crypto.MarshalPublicKey(edPriv.Public())
	wantEncKeyPEM, _ := crypto.MarshalPrivateKey(xPriv)
	wantEncPubPEM, _ := crypto.MarshalPublicKey(xPriv.PublicKey())

	gotMasterPubPEM, _ := os.ReadFile(paths.MasterIdentityPubPath)
	gotPrivPEM, _ := os.ReadFile(paths.IdentityKeyPath)
	gotPubPEM, _ := os.ReadFile(paths.IdentityPubPath)
	gotEncKeyPEM, _ := os.ReadFile(paths.EncryptionKeyPath)
	gotEncPubPEM, _ := os.ReadFile(paths.EncryptionPubPath)

	if !bytes.Equal(wantMasterPubPEM, gotMasterPubPEM) {
		t.Errorf("master public key mismatch")
	}
	if !bytes.Equal(wantPrivPEM, gotPrivPEM) {
		t.Errorf("identity private key mismatch")
	}
	if !bytes.Equal(wantPubPEM, gotPubPEM) {
		t.Errorf("identity public key mismatch")
	}
	if !bytes.Equal(wantEncKeyPEM, gotEncKeyPEM) {
		t.Errorf("encryption private key mismatch")
	}
	if !bytes.Equal(wantEncPubPEM, gotEncPubPEM) {
		t.Errorf("encryption public key mismatch")
	}
}

func TestRotateEncryption_Success(t *testing.T) {
	tempDir := t.TempDir()
	coreDir := filepath.Join(tempDir, "core")
	_ = os.MkdirAll(coreDir, 0750)

	mnemonic, err := crypto.GenerateMnemonic()
	if err != nil {
		t.Fatalf("failed to generate mnemonic: %v", err)
	}

	paths := NewKeyPaths(tempDir)

	if err := writeIndex(paths.EncryptionIndexPath, 4); err != nil {
		t.Fatalf("failed to write initial encryption index: %v", err)
	}

	// Rotate encryption key (4 -> 5)
	if err := RotateEncryption(paths, mnemonic); err != nil {
		t.Fatalf("RotateEncryption failed: %v", err)
	}

	// Verify encryption key files exist, but identity key files DO NOT exist (since they shouldn't be overwritten/created)
	if _, err := os.Stat(paths.IdentityKeyPath); !os.IsNotExist(err) {
		t.Errorf("expected identity.key to NOT be created by RotateEncryption")
	}

	gotEncKeyPEM, err := os.ReadFile(paths.EncryptionKeyPath)
	if err != nil {
		t.Fatalf("failed to read encryption.key: %v", err)
	}

	// Deriving independently to verify correctness
	xPriv, err := crypto.DeriveEncryptionKey(mnemonic, 5)
	if err != nil {
		t.Fatalf("failed to derive encryption key: %v", err)
	}
	wantEncKeyPEM, _ := crypto.MarshalPrivateKey(xPriv)

	if !bytes.Equal(wantEncKeyPEM, gotEncKeyPEM) {
		t.Errorf("expected encryption key to match index 5 key")
	}

	// Verify index on disk is now 5
	index, err := readIndex(paths.EncryptionIndexPath)
	if err != nil {
		t.Fatalf("failed to read index: %v", err)
	}
	if index != 5 {
		t.Errorf("expected encryption index on disk to be 5, got %d", index)
	}
}

func TestRotateIdentity_Success(t *testing.T) {
	tempDir := t.TempDir()
	coreDir := filepath.Join(tempDir, "core")
	_ = os.MkdirAll(coreDir, 0750)

	mnemonic, err := crypto.GenerateMnemonic()
	if err != nil {
		t.Fatalf("failed to generate mnemonic: %v", err)
	}

	paths := NewKeyPaths(tempDir)

	if err := writeIndex(paths.IdentityIndexPath, 4); err != nil {
		t.Fatalf("failed to write initial identity index: %v", err)
	}

	// Rotate identity key (4 -> 5)
	if err := RotateIdentity(paths, mnemonic); err != nil {
		t.Fatalf("RotateIdentity failed: %v", err)
	}

	// Verify identity key files exist, but encryption key files DO NOT exist
	if _, err := os.Stat(paths.EncryptionKeyPath); !os.IsNotExist(err) {
		t.Errorf("expected encryption.key to NOT be created by RotateIdentity")
	}

	gotPrivPEM, err := os.ReadFile(paths.IdentityKeyPath)
	if err != nil {
		t.Fatalf("failed to read identity.key: %v", err)
	}

	// Deriving independently to verify correctness
	edPriv, err := crypto.DeriveIdentityKey(mnemonic, 5)
	if err != nil {
		t.Fatalf("failed to derive identity key: %v", err)
	}
	wantPrivPEM, _ := crypto.MarshalPrivateKey(edPriv)

	if !bytes.Equal(wantPrivPEM, gotPrivPEM) {
		t.Errorf("expected identity key to match index 5 key")
	}

	// Verify index on disk is now 5
	index, err := readIndex(paths.IdentityIndexPath)
	if err != nil {
		t.Fatalf("failed to read index: %v", err)
	}
	if index != 5 {
		t.Errorf("expected identity index on disk to be 5, got %d", index)
	}
}

func TestMnemonicStore(t *testing.T) {
	store := NewMnemonicStore()
	if store.Get() != "" {
		t.Errorf("expected initial store to be empty, got %q", store.Get())
	}

	mnemonic := "apple banana cherry dog elephant fox grape horse ink juice king lemon"
	store.Set(mnemonic)
	if store.Get() != mnemonic {
		t.Errorf("expected store to return %q, got %q", mnemonic, store.Get())
	}
}

func TestKeyState_HexPubKeys(t *testing.T) {
	tempDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tempDir, "core"), 0750)

	paths := NewKeyPaths(tempDir)
	keys, err := LoadOrGenerate(paths)
	if err != nil {
		t.Fatalf("LoadOrGenerate failed: %v", err)
	}

	encPubHex, err := keys.EncPubKeyHex()
	if err != nil {
		t.Fatalf("EncPubKeyHex failed: %v", err)
	}
	if len(encPubHex) == 0 {
		t.Error("expected non-empty hex-encoded encryption public key")
	}

	masterPubHex, err := keys.MasterPubKeyHex()
	if err != nil {
		t.Fatalf("MasterPubKeyHex failed: %v", err)
	}
	if len(masterPubHex) == 0 {
		t.Error("expected non-empty hex-encoded master identity public key")
	}
}


