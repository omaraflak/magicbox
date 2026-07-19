package rest

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/magicbox/core/internal/logging"
)

type UpdateInfo struct {
	Image           string   `json:"image"`
	LocalDigests    []string `json:"local_digests"`
	LatestDigest    string   `json:"latest_digest"`
	UpdateAvailable bool     `json:"update_available"`
	Error           string   `json:"error,omitempty"`
}

type AppUpdateInfo struct {
	ID              string   `json:"id"`
	AppID           string   `json:"app_id"`
	Name            string   `json:"name"`
	Image           string   `json:"image"`
	LocalDigests    []string `json:"local_digests"`
	LatestDigest    string   `json:"latest_digest"`
	UpdateAvailable bool     `json:"update_available"`
	Error           string   `json:"error,omitempty"`
}

type CheckUpdatesResponse struct {
	Core *UpdateInfo     `json:"core,omitempty"`
	Apps []AppUpdateInfo `json:"apps"`
}

func (s *Server) handleCheckUpdates(w http.ResponseWriter, r *http.Request) {
	claims := GetUserFromContext(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var resp CheckUpdatesResponse
	ctx := r.Context()

	// 1. If admin, check Core system update
	if claims.IsAdmin {
		coreInfo := &UpdateInfo{
			Image: "docker.io/omaraflak/magicbox-core:latest",
		}
		if s.docker == nil {
			coreInfo.Error = "docker client not initialized (mock)"
		} else {
			hostname, err := os.Hostname()
			if err != nil {
				coreInfo.Error = "failed to get hostname: " + err.Error()
			} else {
				selfInspect, err := s.docker.InspectRawContainer(ctx, hostname)
				if err != nil {
					coreInfo.Error = "failed to inspect self container: " + err.Error()
				} else {
					imageRef := selfInspect.Config.Image
					coreInfo.Image = imageRef

					local, latest, available, err := s.checkImageUpdate(ctx, imageRef, selfInspect.Image, "")
					coreInfo.LocalDigests = local
					coreInfo.LatestDigest = latest
					coreInfo.UpdateAvailable = available
					if err != nil {
						coreInfo.Error = err.Error()
					}
				}
			}
		}
		resp.Core = coreInfo
	}

	// 2. Check Apps updates
	apps, err := s.db.ListAppsByUserID(claims.UserID)
	if err != nil {
		s.logger.Error("check updates: database error", logging.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp.Apps = make([]AppUpdateInfo, 0, len(apps))
	for _, app := range apps {
		appInfo := AppUpdateInfo{
			ID:    app.ID,
			AppID: app.AppID,
			Name:  app.Name,
			Image: app.Image,
		}

		if s.docker == nil {
			appInfo.Error = "docker client not initialized (mock)"
		} else {
			local, latest, available, err := s.checkImageUpdate(ctx, app.Image, "", app.ImageDigest)
			appInfo.LocalDigests = local
			appInfo.LatestDigest = latest
			appInfo.UpdateAvailable = available
			if err != nil {
				appInfo.Error = err.Error()
			}
		}
		resp.Apps = append(resp.Apps, appInfo)
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) checkImageUpdate(ctx context.Context, imageRef string, localImageID string, fallbackDigest string) (localDigests []string, latestDigest string, updateAvailable bool, err error) {
	if s.docker == nil {
		return nil, "", false, fmt.Errorf("docker client not initialized")
	}

	// Resolve local digests
	if localImageID != "" {
		localDigests, err = s.docker.LocalImageDigests(ctx, localImageID)
	}
	if len(localDigests) == 0 {
		localDigests, _ = s.docker.LocalImageDigests(ctx, imageRef)
	}
	if len(localDigests) == 0 && fallbackDigest != "" {
		localDigests = []string{fallbackDigest}
	}

	// Resolve remote digest
	remoteDigest, err := s.docker.RemoteImageDigest(ctx, imageRef)
	if err != nil {
		return localDigests, "", false, err
	}

	// Check if remoteDigest is present locally
	isMatched := false
	for _, d := range localDigests {
		if strings.Contains(d, remoteDigest) {
			isMatched = true
			break
		}
	}

	return localDigests, remoteDigest, !isMatched, nil
}
