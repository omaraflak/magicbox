package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlePresets_GET_Empty(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	req := httptest.NewRequest("GET", "/api/presets", nil)
	w := httptest.NewRecorder()
	handlePresets(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var list []Preset
	json.NewDecoder(resp.Body).Decode(&list)
	if len(list) != 0 {
		t.Errorf("Expected 0 presets, got %d", len(list))
	}
}

func TestHandlePresets_POST_Create(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	body := strings.NewReader(`{"name":"My Preset","params":"{\"temperature\":0.9}"}`)
	req := httptest.NewRequest("POST", "/api/presets", body)
	w := httptest.NewRecorder()
	handlePresets(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	var p Preset
	json.NewDecoder(resp.Body).Decode(&p)
	if p.Name != "My Preset" || p.Params != `{"temperature":0.9}` {
		t.Errorf("Unexpected preset fields in response: %+v", p)
	}
}

func TestHandlePresets_POST_NoName(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	body := strings.NewReader(`{"name":"","params":"{\"temperature\":0.9}"}`)
	req := httptest.NewRequest("POST", "/api/presets", body)
	w := httptest.NewRecorder()
	handlePresets(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for empty name, got %d", resp.StatusCode)
	}
}

func TestHandlePresets_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest("PUT", "/api/presets", nil)
	w := httptest.NewRecorder()
	handlePresets(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandlePresetDetail_PUT(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	p, _ := createPreset("Preset Original", `{"temp":0.5}`)

	body := strings.NewReader(`{"name":"Preset Updated","params":"{\"temp\":0.8}"}`)
	req := httptest.NewRequest("PUT", "/api/presets/"+p.ID, body)
	req.SetPathValue("id", p.ID)

	w := httptest.NewRecorder()
	handlePresetDetail(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify update in DB
	presets, _ := getPresets()
	if len(presets) != 1 || presets[0].Name != "Preset Updated" {
		t.Errorf("Preset update not reflected in DB: %+v", presets)
	}
}

func TestHandlePresetDetail_DELETE(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	p, _ := createPreset("To Delete", `{"temp":0.5}`)

	req := httptest.NewRequest("DELETE", "/api/presets/"+p.ID, nil)
	req.SetPathValue("id", p.ID)

	w := httptest.NewRecorder()
	handlePresetDetail(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	presets, _ := getPresets()
	if len(presets) != 0 {
		t.Errorf("Preset not deleted from DB: %d remaining", len(presets))
	}
}

func TestHandlePresetDetail_MissingID(t *testing.T) {
	req := httptest.NewRequest("DELETE", "/api/presets/", nil)
	w := httptest.NewRecorder()
	handlePresetDetail(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestHandlePresetDetail_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/presets/123", nil)
	req.SetPathValue("id", "123")
	w := httptest.NewRecorder()
	handlePresetDetail(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}
