package cron

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	libp2pCrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/magicbox/core/internal/config"
	"github.com/magicbox/core/internal/crypto"
	"github.com/magicbox/core/internal/db"
	"github.com/magicbox/core/internal/keymanager"
	"github.com/magicbox/core/internal/logging"
	"github.com/magicbox/core/internal/p2p"
	"github.com/magicbox/core/internal/protocol"
)

const (
	identityRotationInterval   = 15 * 24 * time.Hour
	encryptionRotationInterval = 3 * 24 * time.Hour
)

// StartKeyRotationJobs starts key rotation checks on a ticker (every 1 hour).
// It runs once immediately on start. Returns a stop function.
func StartKeyRotationJobs(
	database *db.DB,
	p2pService p2p.Service,
	config *config.Config,
	logger *logging.Logger,
) func() {
	ticker := time.NewTicker(1 * time.Hour)
	done := make(chan struct{})

	go func() {
		// Run immediate check.
		checkAndRotate(database, p2pService, config, logger)

		for {
			select {
			case <-ticker.C:
				checkAndRotate(database, p2pService, config, logger)
			case <-done:
				return
			}
		}
	}()

	return func() {
		ticker.Stop()
		close(done)
	}
}

func checkAndRotate(
	database *db.DB,
	p2pService p2p.Service,
	config *config.Config,
	logger *logging.Logger,
) {
	paths := keymanager.NewKeyPaths(config.Root)

	// 1. Identity Key rotation check
	if info, err := os.Stat(paths.IdentityKeyPath); err == nil {
		modTime := info.ModTime()
		if time.Since(modTime) >= identityRotationInterval {
			mnemonic := config.MnemonicStore.Get()
			if mnemonic == "" {
				logger.Warn("identity key rotation due but system is locked")
			} else {
				logger.Info("running automatic identity key rotation")
				if err := rotateIdentity(database, p2pService, config, paths, mnemonic, logger); err != nil {
					logger.Error("automatic identity rotation failed", logging.F("error", err.Error()))
				}
			}
		}
	} else if !os.IsNotExist(err) {
		logger.Error("failed to stat identity key file", logging.F("error", err.Error()))
	}

	// 2. Encryption Key rotation check
	if info, err := os.Stat(paths.EncryptionKeyPath); err == nil {
		modTime := info.ModTime()
		if time.Since(modTime) >= encryptionRotationInterval {
			mnemonic := config.MnemonicStore.Get()
			if mnemonic == "" {
				logger.Warn("encryption key rotation due but system is locked")
			} else {
				logger.Info("running automatic encryption key rotation")
				if err := rotateEncryption(database, config, paths, mnemonic, logger); err != nil {
					logger.Error("automatic encryption rotation failed", logging.F("error", err.Error()))
				}
			}
		}
	} else if !os.IsNotExist(err) {
		logger.Error("failed to stat encryption key file", logging.F("error", err.Error()))
	}
}

func rotateIdentity(
	database *db.DB,
	p2pService p2p.Service,
	config *config.Config,
	paths *keymanager.KeyPaths,
	mnemonic string,
	logger *logging.Logger,
) error {
	newIndex := config.Keys.IdentityKeyIndex + 1

	masterPriv, err := crypto.DeriveIdentityKey(mnemonic, 0)
	if err != nil {
		return err
	}
	masterPubBytes := masterPriv.Public().(ed25519.PublicKey)
	masterPubKeyHex := hex.EncodeToString(masterPubBytes)

	oldPriv, err := crypto.DeriveIdentityKey(mnemonic, config.Keys.IdentityKeyIndex)
	if err != nil {
		return err
	}

	newPriv, err := crypto.DeriveIdentityKey(mnemonic, newIndex)
	if err != nil {
		return err
	}

	libp2pNewPriv, _, err := libp2pCrypto.KeyPairFromStdKey(&newPriv)
	if err != nil {
		return err
	}
	newPeerID, err := peer.IDFromPrivateKey(libp2pNewPriv)
	if err != nil {
		return err
	}

	libp2pOldPriv, _, err := libp2pCrypto.KeyPairFromStdKey(&oldPriv)
	if err != nil {
		return err
	}
	oldPeerID, err := peer.IDFromPrivateKey(libp2pOldPriv)
	if err != nil {
		return err
	}

	addrs := p2pService.Multiaddrs()
	if len(addrs) == 0 {
		return fmt.Errorf("no multiaddresses found on p2p service")
	}
	oldMultiaddr := addrs[0]
	newMultiaddr := strings.ReplaceAll(oldMultiaddr, oldPeerID.String(), newPeerID.String())

	xPriv, err := crypto.DeriveEncryptionKey(mnemonic, config.Keys.EncryptionKeyIndex)
	if err != nil {
		return err
	}
	pubHex := hex.EncodeToString(xPriv.PublicKey().Bytes())

	cert, err := protocol.SignSuccessionCertificate(masterPriv, masterPubKeyHex, oldPeerID.String(), newPeerID.String(), newMultiaddr, pubHex)
	if err != nil {
		return err
	}
	certBytes, err := json.Marshal(cert)
	if err != nil {
		return err
	}

	if err := keymanager.RotateIdentity(paths, mnemonic); err != nil {
		return err
	}

	config.Keys.IdentityKeyIndex = newIndex

	contacts, err := database.GetAllContacts()
	if err != nil {
		return err
	}
	if err := protocol.EnqueueForContacts(database, contacts, protocol.AppIDKeySuccession, certBytes); err != nil {
		logger.Error("failed to enqueue succession certificates to contacts during automatic rotation", logging.F("error", err.Error()))
	}

	logger.Info("automatic identity key rotation succeeded")
	return nil
}

func rotateEncryption(
	database *db.DB,
	config *config.Config,
	paths *keymanager.KeyPaths,
	mnemonic string,
	logger *logging.Logger,
) error {
	newIndex := config.Keys.EncryptionKeyIndex + 1

	if err := keymanager.RotateEncryption(paths, mnemonic); err != nil {
		return err
	}

	config.Keys.EncryptionKeyIndex = newIndex

	xPriv, err := crypto.DeriveEncryptionKey(mnemonic, newIndex)
	if err != nil {
		return err
	}
	pubHex := hex.EncodeToString(xPriv.PublicKey().Bytes())

	contacts, err := database.GetAllContacts()
	if err != nil {
		return err
	}

	if err := protocol.EnqueueForContacts(database, contacts, protocol.AppIDKeyUpdate, []byte(pubHex)); err != nil {
		logger.Error("failed to enqueue key updates to contacts during automatic rotation", logging.F("error", err.Error()))
	}

	logger.Info("automatic encryption key rotation succeeded")
	return nil
}
