package rest

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/magicbox/core/internal/logging"
)

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

func (s *Server) handleAdminRestart(w http.ResponseWriter, r *http.Request) {
	s.logger.Info("admin: container restart requested, shutting down process")

	writeJSON(w, http.StatusOK, map[string]string{"message": "restarting"})

	go func() {
		time.Sleep(1 * time.Second)
		s.onRestart()
	}()
}
