package rest

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	libp2pCrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/tyler-smith/go-bip39"

	"github.com/magicbox/core/internal/crypto"
	"github.com/magicbox/core/internal/db"
	"github.com/magicbox/core/internal/keymanager"
	"github.com/magicbox/core/internal/logging"
	"github.com/magicbox/core/internal/protocol"
)

func (s *Server) handleAdminGetMnemonic(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"mnemonic":             s.config.Keys.Mnemonic,
		"acknowledged":         s.config.Keys.Mnemonic == "",
		"identity_key_index":   s.config.Keys.IdentityKeyIndex,
		"encryption_key_index": s.config.Keys.EncryptionKeyIndex,
	})
}

func (s *Server) handleAdminAcknowledgeMnemonic(w http.ResponseWriter, r *http.Request) {
	// Security: Wipe the in-memory plaintext mnemonic phrase.
	s.config.Keys.Mnemonic = ""

	writeJSON(w, http.StatusOK, map[string]string{"message": "mnemonic acknowledged and cleared from memory"})
}

func (s *Server) handleAdminRotateKeys(w http.ResponseWriter, r *http.Request) {
	mnemonic := s.config.MnemonicStore.Get()
	if mnemonic == "" {
		writeError(w, http.StatusPreconditionFailed, "system is locked")
		return
	}

	var req struct {
		RotateEncryption bool `json:"rotate_encryption"`
		RotateIdentity   bool `json:"rotate_identity"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !req.RotateEncryption && !req.RotateIdentity {
		writeError(w, http.StatusBadRequest, "at least one key type must be selected for rotation")
		return
	}

	user := GetUserFromContext(r)
	if user == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	contacts, err := s.db.GetContacts(user.UserID)
	if err != nil {
		s.logger.Error("admin rotate keys: database error getting contacts", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	paths := keymanager.NewKeyPaths(s.config.Root)

	oldIdentityIndex := s.config.Keys.IdentityKeyIndex
	newIdentityIndex := oldIdentityIndex
	if req.RotateIdentity {
		newIdentityIndex++
	}

	newEncryptionIndex := s.config.Keys.EncryptionKeyIndex
	if req.RotateEncryption {
		newEncryptionIndex++
	}

	// 1. Rotate on disk
	if req.RotateIdentity {
		if err := keymanager.RotateIdentity(paths, mnemonic); err != nil {
			s.logger.Error("admin rotate keys: failed to rotate identity key on disk", logging.F("error", err.Error()))
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
	}
	if req.RotateEncryption {
		if err := keymanager.RotateEncryption(paths, mnemonic); err != nil {
			s.logger.Error("admin rotate keys: failed to rotate encryption key on disk", logging.F("error", err.Error()))
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
	}

	// 2. Derive encryption key (new one if rotated, current one if not)
	xPriv, err := crypto.DeriveEncryptionKey(mnemonic, newEncryptionIndex)
	if err != nil {
		s.logger.Error("admin rotate keys: failed to derive encryption key", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	encPubKeyHex := hex.EncodeToString(xPriv.PublicKey().Bytes())

	// 3. Enqueue updates for contacts
	if req.RotateIdentity {
		if err := s.buildAndEnqueueSuccession(mnemonic, oldIdentityIndex, newIdentityIndex, contacts, encPubKeyHex); err != nil {
			s.logger.Error("admin rotate keys: failed to build and enqueue succession", logging.F("error", err.Error()))
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
	} else {
		// Only encryption keys rotated, send regular key update message
		if err := protocol.EnqueueForContacts(s.db, contacts, protocol.AppIDKeyUpdate, []byte(encPubKeyHex)); err != nil {
			s.logger.Error("admin rotate keys: failed to enqueue key updates", logging.F("error", err.Error()))
		}
	}

	// 4. Update in-memory config indices
	s.config.Keys.IdentityKeyIndex = newIdentityIndex
	s.config.Keys.EncryptionKeyIndex = newEncryptionIndex

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Selected keys rotated successfully. Restart required.",
	})

	go func() {
		time.Sleep(1 * time.Second)
		s.onRestart()
	}()
}

func (s *Server) handleAdminResetIdentityKeys(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)
	if user == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req struct {
		Mnemonic string `json:"mnemonic"`
	}
	bodyBytes, err := io.ReadAll(http.MaxBytesReader(nil, r.Body, maxBodySize))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read body")
		return
	}
	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}

	mnemonic := req.Mnemonic
	isNew := false
	if mnemonic == "" {
		isNew = true
		var err error
		mnemonic, err = crypto.GenerateMnemonic()
		if err != nil {
			s.logger.Error("admin reset identity keys: failed to generate mnemonic", logging.F("error", err.Error()))
			writeError(w, http.StatusInternalServerError, "failed to generate mnemonic")
			return
		}
	} else {
		if !bip39.IsMnemonicValid(mnemonic) {
			writeError(w, http.StatusBadRequest, "invalid mnemonic phrase")
			return
		}
	}

	masterPriv, err := crypto.DeriveIdentityKey(mnemonic, 0)
	if err != nil {
		s.logger.Error("admin reset identity keys: failed to derive master identity key", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	masterPubBytes, err := crypto.MarshalPublicKey(masterPriv.Public())
	if err != nil {
		s.logger.Error("admin reset identity keys: failed to marshal master public key", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	matchesMaster := bytes.Equal(masterPubBytes, s.config.Keys.MasterPublicKeyPEM)
	paths := keymanager.NewKeyPaths(s.config.Root)

	if matchesMaster && !isNew {
		// Path A: Operational key compromise recovery (Succession)
		newIndex := s.config.Keys.IdentityKeyIndex + 1
		newEncIndex := s.config.Keys.EncryptionKeyIndex + 1

		newXPriv, err := crypto.DeriveEncryptionKey(mnemonic, newEncIndex)
		if err != nil {
			s.logger.Error("admin reset identity keys: failed to derive new encryption key", logging.F("error", err.Error()))
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		newEncPubKeyHex := hex.EncodeToString(newXPriv.PublicKey().Bytes())

		contacts, err := s.db.GetContacts(user.UserID)
		if err != nil {
			s.logger.Error("admin reset identity keys: database error getting contacts", logging.F("error", err.Error()))
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		if err := s.buildAndEnqueueSuccession(mnemonic, s.config.Keys.IdentityKeyIndex, newIndex, contacts, newEncPubKeyHex); err != nil {
			s.logger.Error("admin reset identity keys: failed to build and enqueue succession", logging.F("error", err.Error()))
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		if err := keymanager.RotateIdentity(paths, mnemonic); err != nil {
			s.logger.Error("admin reset identity keys: failed to rotate identity on disk", logging.F("error", err.Error()))
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if err := keymanager.RotateEncryption(paths, mnemonic); err != nil {
			s.logger.Error("admin reset identity keys: failed to rotate encryption on disk", logging.F("error", err.Error()))
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		s.config.Keys.IdentityKeyIndex = newIndex
		s.config.Keys.EncryptionKeyIndex = newEncIndex

		writeJSON(w, http.StatusOK, map[string]string{
			"message": "Identity keys rotated and succession certificate queued for all contacts. Contacts preserved.",
		})

		go func() {
			time.Sleep(1 * time.Second)
			s.onRestart()
		}()
	} else {
		// Path B: Mnemonic compromise recovery (Nuclear Reset)
		oldMnemonic := s.config.MnemonicStore.Get()
		if oldMnemonic == "" {
			writeError(w, http.StatusPreconditionFailed, "system is locked")
			return
		}

		oldMasterPriv, err := crypto.DeriveIdentityKey(oldMnemonic, 0)
		if err != nil {
			s.logger.Error("admin reset identity keys: failed to derive old master key", logging.F("error", err.Error()))
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		// 1. Get all contacts of the user
		contacts, err := s.db.GetContacts(user.UserID)
		if err != nil {
			s.logger.Error("admin reset identity keys: failed to get contacts", logging.F("error", err.Error()))
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		// 2. Generate new keys
		newMasterPriv, err := crypto.DeriveIdentityKey(mnemonic, 0)
		if err != nil {
			s.logger.Error("admin reset identity keys: failed to derive new master key", logging.F("error", err.Error()))
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		newMasterPubBytesRaw := newMasterPriv.Public().(ed25519.PublicKey)
		newMasterPubHex := hex.EncodeToString(newMasterPubBytesRaw)

		newEdPriv, err := crypto.DeriveIdentityKey(mnemonic, 1)
		if err != nil {
			s.logger.Error("admin reset identity keys: failed to derive new identity key", logging.F("error", err.Error()))
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		libp2pNewPriv, _, err := libp2pCrypto.KeyPairFromStdKey(&newEdPriv)
		if err != nil {
			s.logger.Error("admin reset identity keys: failed to convert new standard key", logging.F("error", err.Error()))
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		newPeerID, err := peer.IDFromPrivateKey(libp2pNewPriv)
		if err != nil {
			s.logger.Error("admin reset identity keys: failed to get new peer ID", logging.F("error", err.Error()))
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		newXPriv, err := crypto.DeriveEncryptionKey(mnemonic, 1)
		if err != nil {
			s.logger.Error("admin reset identity keys: failed to derive new encryption key", logging.F("error", err.Error()))
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		newEncPubHex := hex.EncodeToString(newXPriv.PublicKey().Bytes())

		// Compute new multiaddr
		oldPeerID := s.p2pService.HostID()
		addrs := s.p2pService.Multiaddrs()
		if len(addrs) == 0 {
			s.logger.Error("admin reset identity keys: no multiaddresses found")
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		oldMultiaddr := addrs[0]
		newMultiaddr := strings.ReplaceAll(oldMultiaddr, oldPeerID, newPeerID.String())

		// Generate the reconnect payload using new keys
		requestPayloadBytes, err := json.Marshal(protocol.ContactRequestPayload{
			DisplayName:  user.Username,
			Multiaddr:    newMultiaddr,
			EncPubKey:    newEncPubHex,
			UserID:       user.UserID,
			MasterPubKey: newMasterPubHex,
		})
		if err != nil {
			s.logger.Error("admin reset identity keys: failed to marshal contact request payload", logging.F("error", err.Error()))
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		timestamp := time.Now().UTC().Format(time.RFC3339)

		// 3. For each contact, enqueue revocation and reconnection request
		for _, c := range contacts {
			// Sign revocation with old master key
			msgToSign := []byte("REVOKE_MASTER_KEY:" + user.UserID + ":" + timestamp)
			sigBytes := ed25519.Sign(oldMasterPriv, msgToSign)
			sigHex := hex.EncodeToString(sigBytes)

			payloadBytes, err := json.Marshal(protocol.MasterRevocationPayload{
				UserID:    user.UserID,
				Timestamp: timestamp,
				Signature: sigHex,
			})
			if err != nil {
				s.logger.Error("admin reset identity keys: failed to marshal revocation payload", logging.F("error", err.Error()))
				continue
			}

			// Enqueue revocation payload
			if err := protocol.EnqueueForContacts(s.db, []db.Contact{c}, protocol.AppIDMasterRevocation, payloadBytes); err != nil {
				s.logger.Error("admin reset identity keys: failed to enqueue revocation", logging.F("contact_id", c.ID), logging.F("error", err.Error()))
			}

			// Enqueue reconnect payload
			if err := protocol.EnqueueForContacts(s.db, []db.Contact{c}, protocol.AppIDContactRequest, requestPayloadBytes); err != nil {
				s.logger.Error("admin reset identity keys: failed to enqueue reconnect request", logging.F("contact_id", c.ID), logging.F("error", err.Error()))
			}
		}

		// 4. Recover keys on disk to the new mnemonic
		if err := keymanager.RecoverAll(paths, mnemonic, 1, 1); err != nil {
			s.logger.Error("admin reset identity keys: failed to recover keys", logging.F("error", err.Error()))
			writeError(w, http.StatusBadRequest, "failed to recover keys: "+err.Error())
			return
		}

		s.config.Keys.IdentityKeyIndex = 1
		s.config.Keys.EncryptionKeyIndex = 1
		s.config.Keys.Mnemonic = mnemonic
		s.config.MnemonicStore.Set(mnemonic)

		writeJSON(w, http.StatusOK, map[string]string{
			"mnemonic": mnemonic,
			"message":  "P2P identity reset successfully. Revocation and reconnect requests sent to all contacts. Restart required.",
		})

		go func() {
			time.Sleep(1 * time.Second)
			s.onRestart()
		}()
	}
}

func (s *Server) handleAdminUnlock(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Mnemonic string `json:"mnemonic"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Mnemonic == "" || !bip39.IsMnemonicValid(req.Mnemonic) {
		writeError(w, http.StatusBadRequest, "invalid mnemonic phrase")
		return
	}

	masterPriv, err := crypto.DeriveIdentityKey(req.Mnemonic, 0)
	if err != nil {
		s.logger.Error("admin unlock: failed to derive master key", logging.F("error", err.Error()))
		writeError(w, http.StatusBadRequest, "invalid mnemonic")
		return
	}

	masterPubPEM, err := crypto.MarshalPublicKey(masterPriv.Public())
	if err != nil {
		s.logger.Error("admin unlock: failed to marshal master public key", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if !bytes.Equal(masterPubPEM, s.config.Keys.MasterPublicKeyPEM) {
		s.logger.Warn("admin unlock: mnemonic does not match master public key")
		writeError(w, http.StatusBadRequest, "mnemonic mismatch")
		return
	}

	s.config.MnemonicStore.Set(req.Mnemonic)
	s.logger.Info("admin unlock: system unlocked successfully")
	writeJSON(w, http.StatusOK, map[string]string{"message": "system unlocked successfully"})
}

func (s *Server) handleAdminStatus(w http.ResponseWriter, r *http.Request) {
	unlocked := s.config.MnemonicStore.Get() != ""
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"unlocked":         unlocked,
		"identity_index":   s.config.Keys.IdentityKeyIndex,
		"encryption_index": s.config.Keys.EncryptionKeyIndex,
	})
}

// buildAndEnqueueSuccession derives standard keys, converts to libp2p keys, generates
// a succession certificate, and enqueues it for delivery to contact peers.
func (s *Server) buildAndEnqueueSuccession(mnemonic string, oldIndex, newIndex int, contacts []db.Contact, encPubKeyHex string) error {
	masterPriv, err := crypto.DeriveIdentityKey(mnemonic, 0)
	if err != nil {
		return fmt.Errorf("failed to derive master identity key: %w", err)
	}
	masterPubBytes := masterPriv.Public().(ed25519.PublicKey)
	masterPubKeyHex := hex.EncodeToString(masterPubBytes)

	oldPriv, err := crypto.DeriveIdentityKey(mnemonic, oldIndex)
	if err != nil {
		return fmt.Errorf("failed to derive old identity key: %w", err)
	}

	newPriv, err := crypto.DeriveIdentityKey(mnemonic, newIndex)
	if err != nil {
		return fmt.Errorf("failed to derive new identity key: %w", err)
	}

	libp2pNewPriv, _, err := libp2pCrypto.KeyPairFromStdKey(&newPriv)
	if err != nil {
		return fmt.Errorf("failed to convert new std key to libp2p key: %w", err)
	}
	newPeerID, err := peer.IDFromPrivateKey(libp2pNewPriv)
	if err != nil {
		return fmt.Errorf("failed to get peer ID from private key: %w", err)
	}

	libp2pOldPriv, _, err := libp2pCrypto.KeyPairFromStdKey(&oldPriv)
	if err != nil {
		return fmt.Errorf("failed to convert old std key to libp2p key: %w", err)
	}
	oldPeerID, err := peer.IDFromPrivateKey(libp2pOldPriv)
	if err != nil {
		return fmt.Errorf("failed to get old peer ID: %w", err)
	}

	addrs := s.p2pService.Multiaddrs()
	if len(addrs) == 0 {
		return fmt.Errorf("no multiaddresses found")
	}
	oldMultiaddr := addrs[0]
	newMultiaddr := strings.ReplaceAll(oldMultiaddr, oldPeerID.String(), newPeerID.String())

	cert, err := protocol.SignSuccessionCertificate(masterPriv, masterPubKeyHex, oldPeerID.String(), newPeerID.String(), newMultiaddr, encPubKeyHex)
	if err != nil {
		return fmt.Errorf("failed to sign succession certificate: %w", err)
	}
	certBytes, err := json.Marshal(cert)
	if err != nil {
		return fmt.Errorf("failed to marshal succession certificate: %w", err)
	}

	return protocol.EnqueueForContacts(s.db, contacts, protocol.AppIDKeySuccession, certBytes)
}
