package rest

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/magicbox/core/internal/logging"
)

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
