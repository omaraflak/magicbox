package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleModels_Success(t *testing.T) {
	oldGetModels := getModels
	defer func() { getModels = oldGetModels }()

	mockModels := []ModelInfo{
		{
			Name:        "models/gemini-3.1-flash-lite",
			DisplayName: "Gemini 3.1 Flash Lite",
			Description: "Lightweight and fast",
		},
	}
	getModels = func(ctx context.Context) ([]ModelInfo, error) {
		return mockModels, nil
	}

	req := httptest.NewRequest("GET", "/api/models", nil)
	w := httptest.NewRecorder()
	handleModels(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var models []ModelInfo
	json.NewDecoder(resp.Body).Decode(&models)
	if len(models) != 1 || models[0].Name != "models/gemini-3.1-flash-lite" {
		t.Errorf("Unexpected models in response: %+v", models)
	}
}

func TestHandleModels_Error(t *testing.T) {
	oldGetModels := getModels
	defer func() { getModels = oldGetModels }()

	getModels = func(ctx context.Context) ([]ModelInfo, error) {
		return nil, errors.New("Gemini client error simulation")
	}

	req := httptest.NewRequest("GET", "/api/models", nil)
	w := httptest.NewRecorder()
	handleModels(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}
}

func TestHandleModels_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/models", nil)
	w := httptest.NewRecorder()
	handleModels(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}
