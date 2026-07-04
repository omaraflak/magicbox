package rest

import (
	"bufio"
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	libp2pCrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/bcrypt"

	"github.com/magicbox/core/internal/crypto"
	"github.com/magicbox/core/internal/keymanager"
	"github.com/magicbox/core/internal/logging"
	"github.com/magicbox/core/internal/protocol"
)

func (s *Server) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.db.ListUsers()
	if err != nil {
		s.logger.Error("admin list users: database error", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	var result []map[string]interface{}
	for _, u := range users {
		result = append(result, map[string]interface{}{
			"id":         u.ID,
			"username":   u.Username,
			"is_admin":   u.IsAdmin,
			"created_at": u.CreatedAt,
		})
	}
	writeJSON(w, http.StatusOK, result)
}

type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	IsAdmin  bool   `json:"is_admin"`
}

func (s *Server) handleAdminCreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !usernameRegex.MatchString(req.Username) {
		writeError(w, http.StatusBadRequest, "username must be 3-32 lowercase alphanumeric or underscore characters")
		return
	}
	if len(req.Password) < minPassLen {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	existing, err := s.db.GetUserByUsername(req.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if existing != nil {
		writeError(w, http.StatusConflict, "username already exists")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	userID := uuid.NewString()
	if err := s.db.CreateUser(userID, req.Username, string(hash), req.IsAdmin); err != nil {
		s.logger.Error("admin create user: database error", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Create user directories.
	userDir := filepath.Join(s.config.Root, "users", req.Username)
	for _, sub := range []string{"apps", "shared"} {
		if err := os.MkdirAll(filepath.Join(userDir, sub), 0750); err != nil {
			s.logger.Error("admin create user: failed to create directory", logging.F("error", err.Error()))
		}
	}

	s.logger.Info("admin: user created", logging.F("username", req.Username))
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":       userID,
		"username": req.Username,
		"is_admin": req.IsAdmin,
	})
}

func (s *Server) handleAdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "missing user id")
		return
	}

	user, err := s.db.GetUserByID(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if user == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	// Cascade delete via orchestrator.
	if err := s.orchestrator.CascadeDeleteUser(r.Context(), userID); err != nil {
		s.logger.Error("admin delete user: orchestrator error", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.logger.Info("admin: user deleted", logging.F("username", user.Username))
	writeJSON(w, http.StatusOK, map[string]string{"message": "user deleted"})
}

func (s *Server) handleAdminListRegistries(w http.ResponseWriter, r *http.Request) {
	registries, err := s.db.ListRegistries()
	if err != nil {
		s.logger.Error("admin list registries: database error", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	var result []map[string]interface{}
	for _, reg := range registries {
		result = append(result, map[string]interface{}{
			"id":         reg.ID,
			"prefix":     reg.Prefix,
			"created_at": reg.CreatedAt,
		})
	}
	writeJSON(w, http.StatusOK, result)
}

type createRegistryRequest struct {
	Prefix string `json:"prefix"`
}

func (s *Server) handleAdminCreateRegistry(w http.ResponseWriter, r *http.Request) {
	var req createRegistryRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Prefix == "" {
		writeError(w, http.StatusBadRequest, "prefix is required")
		return
	}

	id := uuid.NewString()
	if err := s.db.InsertRegistry(id, req.Prefix); err != nil {
		s.logger.Error("admin create registry: database error", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	s.logger.Info("admin: registry added", logging.F("prefix", req.Prefix))
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":     id,
		"prefix": req.Prefix,
	})
}

func (s *Server) handleAdminDeleteRegistry(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing registry id")
		return
	}

	if err := s.db.DeleteRegistry(id); err != nil {
		s.logger.Error("admin delete registry: database error", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "registry deleted"})
}

func (s *Server) handleAdminListLogs(w http.ResponseWriter, r *http.Request) {
	logDir := filepath.Join(s.config.Root, "core/logs")
	files, err := os.ReadDir(logDir)
	if err != nil {
		s.logger.Error("admin list logs: failed to read logs dir", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to read logs directory")
		return
	}

	var logFiles []string
	for _, f := range files {
		if !f.IsDir() && strings.HasPrefix(f.Name(), "magicbox-") && strings.HasSuffix(f.Name(), ".log") {
			logFiles = append(logFiles, f.Name())
		}
	}

	if len(logFiles) == 0 {
		writeJSON(w, http.StatusOK, []string{})
		return
	}

	// Alphabetical sorting sorts YYYY-MM-DD format chronologically.
	sort.Strings(logFiles)
	latestLogFile := filepath.Join(logDir, logFiles[len(logFiles)-1])

	file, err := os.Open(latestLogFile)
	if err != nil {
		s.logger.Error("admin list logs: failed to open log file", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to open log file")
		return
	}
	defer file.Close()

	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			limit = val
			if limit > 1000 {
				limit = 1000
			}
		}
	}

	levelFilter := strings.ToUpper(r.URL.Query().Get("level"))

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		if levelFilter != "" {
			var entry map[string]interface{}
			if err := json.Unmarshal([]byte(line), &entry); err == nil {
				if lvl, ok := entry["level"].(string); ok && lvl == levelFilter {
					lines = append(lines, line)
				}
			}
		} else {
			lines = append(lines, line)
		}
	}

	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}

	writeJSON(w, http.StatusOK, lines)
}

type upgradeRequest struct {
	Image string `json:"image"`
}

func (s *Server) handleAdminUpgrade(w http.ResponseWriter, r *http.Request) {
	var req upgradeRequest
	_ = readJSON(r, &req)

	targetImage := req.Image
	if targetImage == "" {
		targetImage = "docker.io/omaraflak/magicbox-core:latest"
	}

	if s.docker == nil {
		s.logger.Info("admin upgrade: docker client is nil (test mode), skipping actual upgrade")
		writeJSON(w, http.StatusOK, map[string]string{
			"message": "upgrade initiated successfully (mock)",
			"new_id":  "mock-new-id-12345",
		})
		return
	}

	// 1. Resolve our own container hostname (Container ID)
	hostname, err := os.Hostname()
	if err != nil {
		s.logger.Error("admin upgrade: failed to resolve own hostname", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to resolve container hostname")
		return
	}

	// 2. Inspect ourselves to read current config metadata (ports, binds, networks, env, etc.)
	selfInspect, err := s.docker.InspectRawContainer(r.Context(), hostname)
	if err != nil {
		s.logger.Error("admin upgrade: failed to inspect self container", logging.F("hostname", hostname), logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to inspect self container: "+err.Error())
		return
	}

	s.logger.Info("admin upgrade: self-upgrading core container",
		logging.F("target_image", targetImage),
		logging.F("current_id", selfInspect.ID),
		logging.F("current_name", selfInspect.Name),
	)

	// 3. Pull the target image first
	s.logger.Info("admin upgrade: pulling new core image", logging.F("image", targetImage))
	_, err = s.docker.PullImage(r.Context(), targetImage, true)
	if err != nil {
		s.logger.Error("admin upgrade: failed to pull image", logging.F("image", targetImage), logging.F("error", err.Error()))
		writeError(w, http.StatusBadRequest, "failed to pull image: "+err.Error())
		return
	}

	// 4. Rename the current container to current_name + "_old"
	oldName := strings.TrimPrefix(selfInspect.Name, "/") + "_old"
	s.logger.Info("admin upgrade: renaming current container", logging.F("old_name", oldName))
	err = s.docker.RenameContainer(r.Context(), selfInspect.ID, oldName)
	if err != nil {
		s.logger.Error("admin upgrade: failed to rename container", logging.F("id", selfInspect.ID), logging.F("new_name", oldName), logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to rename container: "+err.Error())
		return
	}

	// 5. Recreate the new container using the original config and name
	s.logger.Info("admin upgrade: creating new container version")
	newID, err := s.docker.CreateCoreContainer(r.Context(), targetImage, &selfInspect)
	if err != nil {
		s.logger.Error("admin upgrade: failed to recreate core container", logging.F("error", err.Error()))
		
		// Attempt rollback: rename back to original name
		rollbackName := strings.TrimPrefix(selfInspect.Name, "/")
		_ = s.docker.RenameContainer(r.Context(), selfInspect.ID, rollbackName)

		writeError(w, http.StatusInternalServerError, "failed to recreate container: "+err.Error())
		return
	}

	// 6. Spawn the updater container to stop the old one and start the new one
	originalName := strings.TrimPrefix(selfInspect.Name, "/")
	s.logger.Info("admin upgrade: starting updater helper container", logging.F("old_name", oldName), logging.F("new_name", originalName))
	err = s.docker.StartUpdaterContainer(r.Context(), oldName, originalName)
	if err != nil {
		s.logger.Error("admin upgrade: failed to start updater container", logging.F("error", err.Error()))
		
		// Attempt rollback: delete recreated container and rename old one back
		_ = s.docker.RemoveContainer(r.Context(), newID)
		_ = s.docker.RenameContainer(r.Context(), selfInspect.ID, originalName)

		writeError(w, http.StatusInternalServerError, "failed to start updater container: "+err.Error())
		return
	}

	s.logger.Info("admin upgrade: updater container spawned successfully, core container will restart shortly", logging.F("new_id", newID))

	// Return a success JSON response to frontend client so the HTTP connection can close nicely
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "upgrade initiated successfully, container restarting",
		"new_id":  newID,
	})
}

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

func (s *Server) handleAdminRotateEncryptionKeys(w http.ResponseWriter, r *http.Request) {
	mnemonic := s.config.MnemonicStore.Get()
	if mnemonic == "" {
		writeError(w, http.StatusPreconditionFailed, "system is locked")
		return
	}

	newIndex := s.config.Keys.EncryptionKeyIndex + 1

	paths := keymanager.NewKeyPaths(s.config.Root)
	if err := keymanager.RotateEncryption(paths, mnemonic); err != nil {
		s.logger.Error("admin rotate encryption keys: failed to rotate key", logging.F("error", err.Error()))
		writeError(w, http.StatusBadRequest, "failed to rotate encryption keys: "+err.Error())
		return
	}

	s.config.Keys.EncryptionKeyIndex = newIndex

	xPriv, err := crypto.DeriveEncryptionKey(mnemonic, newIndex)
	if err != nil {
		s.logger.Error("admin rotate encryption keys: failed to derive key", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	pubHex := hex.EncodeToString(xPriv.PublicKey().Bytes())

	user := GetUserFromContext(r)
	if user == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	contacts, err := s.db.GetContacts(user.UserID)
	if err != nil {
		s.logger.Error("admin rotate encryption keys: database error getting contacts", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := protocol.EnqueueForContacts(s.db, contacts, protocol.AppIDKeyUpdate, []byte(pubHex)); err != nil {
		s.logger.Error("admin rotate encryption keys: failed to enqueue key updates", logging.F("error", err.Error()))
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Encryption keys rotated successfully. Restart required.",
	})
}

func (s *Server) handleAdminResetIdentityKeys(w http.ResponseWriter, r *http.Request) {
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
	if mnemonic == "" {
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

	paths := keymanager.NewKeyPaths(s.config.Root)
	if err := keymanager.RecoverAll(paths, mnemonic, 1, 1); err != nil {
		s.logger.Error("admin reset identity keys: failed to recover keys", logging.F("error", err.Error()))
		writeError(w, http.StatusBadRequest, "failed to recover keys: "+err.Error())
		return
	}

	if err := s.db.WipeAllContactsAndRequests(); err != nil {
		s.logger.Error("admin reset identity keys: failed to wipe contacts and requests", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	s.config.Keys.IdentityKeyIndex = 1
	s.config.Keys.EncryptionKeyIndex = 1
	s.config.Keys.Mnemonic = mnemonic

	writeJSON(w, http.StatusOK, map[string]string{
		"mnemonic": mnemonic,
		"message":  "Identity keys rotated successfully. System reset. Restart required.",
	})
}

func (s *Server) handleAdminRotateIdentityKeys(w http.ResponseWriter, r *http.Request) {
	mnemonic := s.config.MnemonicStore.Get()
	if mnemonic == "" {
		writeError(w, http.StatusPreconditionFailed, "system is locked")
		return
	}

	newIndex := s.config.Keys.IdentityKeyIndex + 1

	masterPriv, err := crypto.DeriveIdentityKey(mnemonic, 0)
	if err != nil {
		s.logger.Error("admin rotate identity keys: failed to derive master identity key", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	masterPubBytes := masterPriv.Public().(ed25519.PublicKey)
	masterPubKeyHex := hex.EncodeToString(masterPubBytes)

	oldPriv, err := crypto.DeriveIdentityKey(mnemonic, s.config.Keys.IdentityKeyIndex)
	if err != nil {
		s.logger.Error("admin rotate identity keys: failed to derive old identity key", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	newPriv, err := crypto.DeriveIdentityKey(mnemonic, newIndex)
	if err != nil {
		s.logger.Error("admin rotate identity keys: failed to derive new identity key", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	libp2pNewPriv, _, err := libp2pCrypto.KeyPairFromStdKey(&newPriv)
	if err != nil {
		s.logger.Error("admin rotate identity keys: failed to convert new std key to libp2p key", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	newPeerID, err := peer.IDFromPrivateKey(libp2pNewPriv)
	if err != nil {
		s.logger.Error("admin rotate identity keys: failed to get peer ID from private key", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	libp2pOldPriv, _, err := libp2pCrypto.KeyPairFromStdKey(&oldPriv)
	if err != nil {
		s.logger.Error("admin rotate identity keys: failed to convert old std key to libp2p key", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	oldPeerID, err := peer.IDFromPrivateKey(libp2pOldPriv)
	if err != nil {
		s.logger.Error("admin rotate identity keys: failed to get old peer ID", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	addrs := s.p2pService.Multiaddrs()
	if len(addrs) == 0 {
		s.logger.Error("admin rotate identity keys: no multiaddresses found")
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	oldMultiaddr := addrs[0]
	newMultiaddr := strings.ReplaceAll(oldMultiaddr, oldPeerID.String(), newPeerID.String())

	xPriv, err := crypto.DeriveEncryptionKey(mnemonic, s.config.Keys.EncryptionKeyIndex)
	if err != nil {
		s.logger.Error("admin rotate identity keys: failed to derive encryption key", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	pubHex := hex.EncodeToString(xPriv.PublicKey().Bytes())

	cert, err := protocol.SignSuccessionCertificate(masterPriv, masterPubKeyHex, oldPeerID.String(), newPeerID.String(), newMultiaddr, pubHex)
	if err != nil {
		s.logger.Error("admin rotate identity keys: failed to sign succession certificate", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	certBytes, err := json.Marshal(cert)
	if err != nil {
		s.logger.Error("admin rotate identity keys: failed to marshal succession certificate", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	paths := keymanager.NewKeyPaths(s.config.Root)
	if err := keymanager.RotateIdentity(paths, mnemonic); err != nil {
		s.logger.Error("admin rotate identity keys: failed to rotate identity key on disk", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	s.config.Keys.IdentityKeyIndex = newIndex

	user := GetUserFromContext(r)
	if user == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	contacts, err := s.db.GetContacts(user.UserID)
	if err != nil {
		s.logger.Error("admin rotate identity keys: database error getting contacts", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := protocol.EnqueueForContacts(s.db, contacts, protocol.AppIDKeySuccession, certBytes); err != nil {
		s.logger.Error("admin rotate identity keys: failed to enqueue succession certificate", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Identity keys rotated successfully. Succession certificates queued. Restarting...",
	})

	go func() {
		time.Sleep(1 * time.Second)
		s.onRestart()
	}()
}

func (s *Server) handleAdminRestart(w http.ResponseWriter, r *http.Request) {
	s.logger.Info("admin: container restart requested, shutting down process")

	writeJSON(w, http.StatusOK, map[string]string{"message": "restarting"})

	go func() {
		time.Sleep(1 * time.Second)
		s.onRestart()
	}()
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



