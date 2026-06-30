package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/magicbox/core/sdk"
)

// --- HTTP Handler SPA Tests ---

func TestSPAHandler_StaticAsset(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "web-test")
	if err != nil {
		t.Fatalf("Failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldWebRoot := webRoot
	webRoot = tmpDir
	defer func() { webRoot = oldWebRoot }()

	// Write mock asset file
	if err := os.MkdirAll(filepath.Join(tmpDir, "assets"), 0755); err != nil {
		t.Fatalf("Failed to create assets dir: %v", err)
	}
	assetContent := "body { color: red; }"
	if err := os.WriteFile(filepath.Join(tmpDir, "assets", "style.css"), []byte(assetContent), 0644); err != nil {
		t.Fatalf("Failed to write style.css: %v", err)
	}

	req := httptest.NewRequest("GET", "/assets/style.css", nil)
	w := httptest.NewRecorder()
	sdk.NewHTMLHandler(webRoot).ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for styles, got %d", resp.StatusCode)
	}
	if !strings.Contains(w.Body.String(), "body { color: red; }") {
		t.Errorf("Expected mock css content, got: %s", w.Body.String())
	}
}

func TestSPAHandler_SPARoute(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "web-test")
	if err != nil {
		t.Fatalf("Failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldWebRoot := webRoot
	webRoot = tmpDir
	defer func() { webRoot = oldWebRoot }()

	indexContent := `<!doctype html><html><head><title>Mock App</title></head><body><div id="root"></div></body></html>`
	if err := os.WriteFile(filepath.Join(tmpDir, "index.html"), []byte(indexContent), 0644); err != nil {
		t.Fatalf("Failed to write index.html: %v", err)
	}

	req := httptest.NewRequest("GET", "/conversations/123", nil)
	w := httptest.NewRecorder()
	sdk.NewHTMLHandler(webRoot).ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(w.Body.String(), `<base href="/">`) {
		t.Errorf("Expected base tag with /, got: %s", w.Body.String())
	}
}

func TestSPAHandler_XForwardedPrefix(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "web-test")
	if err != nil {
		t.Fatalf("Failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldWebRoot := webRoot
	webRoot = tmpDir
	defer func() { webRoot = oldWebRoot }()

	indexContent := `<!doctype html><html><head><title>Mock App</title></head><body><div id="root"></div></body></html>`
	if err := os.WriteFile(filepath.Join(tmpDir, "index.html"), []byte(indexContent), 0644); err != nil {
		t.Fatalf("Failed to write index.html: %v", err)
	}

	req := httptest.NewRequest("GET", "/conversations/123", nil)
	req.Header.Set("X-Forwarded-Prefix", "/u/omar/ai")
	w := httptest.NewRecorder()
	sdk.NewHTMLHandler(webRoot).ServeHTTP(w, req)

	if !strings.Contains(w.Body.String(), `<base href="/u/omar/ai/">`) {
		t.Errorf("Expected base tag from X-Forwarded-Prefix, got: %s", w.Body.String())
	}
}
