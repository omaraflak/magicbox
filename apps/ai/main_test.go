package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) {
	var err error
	dbConn, err = sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open memory db: %v", err)
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS conversations (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			params TEXT NOT NULL,
			parent_id TEXT REFERENCES conversations(id)
		)`,
		`CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			conversation_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(conversation_id) REFERENCES conversations(id)
		)`,
		`CREATE TABLE IF NOT EXISTS presets (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			params TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, q := range queries {
		if _, err := dbConn.Exec(q); err != nil {
			t.Fatalf("Failed to create tables: %v", err)
		}
	}
}

func TestSettings(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	err := setSetting("api_key", "test-key-123")
	if err != nil {
		t.Fatalf("setSetting failed: %v", err)
	}

	val := getSetting("api_key")
	if val != "test-key-123" {
		t.Errorf("Expected test-key-123, got %s", val)
	}

	val = getSetting("non_existent")
	if val != "" {
		t.Errorf("Expected empty string for non_existent, got %s", val)
	}
}

func TestConversations(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	c, err := createConversation("Test Chat", `{"temp":0.7}`)
	if err != nil {
		t.Fatalf("createConversation failed: %v", err)
	}

	if c.Title != "Test Chat" {
		t.Errorf("Expected Title 'Test Chat', got %s", c.Title)
	}

	err = updateConversationParams(c.ID, `{"temp":0.9}`)
	if err != nil {
		t.Fatalf("updateConversationParams failed: %v", err)
	}

	c2, err := getConversation(c.ID)
	if err != nil {
		t.Fatalf("getConversation failed: %v", err)
	}
	if c2.Params != `{"temp":0.9}` {
		t.Errorf("Expected params updated, got %s", c2.Params)
	}

	err = updateConversationTitle(c.ID, "New Chat Title")
	if err != nil {
		t.Fatalf("updateConversationTitle failed: %v", err)
	}

	c3, err := getConversation(c.ID)
	if err != nil {
		t.Fatalf("getConversation failed: %v", err)
	}
	if c3.Title != "New Chat Title" {
		t.Errorf("Expected title updated, got %s", c3.Title)
	}

	list, err := getConversationsPaginated(100, 0)
	if err != nil {
		t.Fatalf("getConversationsPaginated failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("Expected list of size 1, got %d", len(list))
	}

	m, err := addMessage(c.ID, "user", "Hello AI")
	if err != nil {
		t.Fatalf("addMessage failed: %v", err)
	}
	if m.Role != "user" || m.Content != "Hello AI" {
		t.Errorf("Unexpected message fields: %+v", m)
	}

	msgs, err := getMessages(c.ID)
	if err != nil {
		t.Fatalf("getMessages failed: %v", err)
	}
	if len(msgs) != 1 || msgs[0].Content != "Hello AI" {
		t.Errorf("Expected 1 message, got %+v", msgs)
	}

	err = deleteConversation(c.ID)
	if err != nil {
		t.Fatalf("deleteConversation failed: %v", err)
	}

	c4, err := getConversation(c.ID)
	if err == nil && c4 != nil {
		t.Errorf("Expected error or nil when fetching deleted conversation")
	}

	msgs2, err := getMessages(c.ID)
	if err != nil {
		t.Fatalf("getMessages failed: %v", err)
	}
	if len(msgs2) != 0 {
		t.Errorf("Expected 0 messages after delete, got %d", len(msgs2))
	}
}

func TestPresets(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	p, err := createPreset("Preset 1", `{"temp":0.5}`)
	if err != nil {
		t.Fatalf("createPreset failed: %v", err)
	}

	if p.Name != "Preset 1" {
		t.Errorf("Expected Preset 1, got %s", p.Name)
	}

	presets, err := getPresets()
	if err != nil {
		t.Fatalf("getPresets failed: %v", err)
	}
	if len(presets) != 1 || presets[0].Name != "Preset 1" {
		t.Errorf("Unexpected presets list: %+v", presets)
	}

	err = deletePreset(p.ID)
	if err != nil {
		t.Fatalf("deletePreset failed: %v", err)
	}

	presets2, err := getPresets()
	if err != nil {
		t.Fatalf("getPresets failed: %v", err)
	}
	if len(presets2) != 0 {
		t.Errorf("Expected empty presets after delete, got %d", len(presets2))
	}
}

func TestHandleSettings(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	// GET request
	req := httptest.NewRequest("GET", "/api/settings", nil)
	w := httptest.NewRecorder()
	handleSettings(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var data map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&data)
	if data["has_api_key"].(bool) != false {
		t.Errorf("Expected has_api_key false, got true")
	}

	// POST request
	body := strings.NewReader(`{"api_key":"my-api-key"}`)
	req = httptest.NewRequest("POST", "/api/settings", body)
	w = httptest.NewRecorder()
	handleSettings(w, req)

	resp = w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// GET again to check
	req = httptest.NewRequest("GET", "/api/settings", nil)
	w = httptest.NewRecorder()
	handleSettings(w, req)

	resp = w.Result()
	json.NewDecoder(resp.Body).Decode(&data)
	if data["has_api_key"].(bool) != true {
		t.Errorf("Expected has_api_key true after POST")
	}
	if data["api_key"].(string) != "my-api-key" {
		t.Errorf("Expected api_key 'my-api-key', got '%v'", data["api_key"])
	}
}

func TestFork(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	c, err := createConversation("Parent Chat", `{"temp":0.5}`)
	if err != nil {
		t.Fatalf("createConversation failed: %v", err)
	}

	m1, _ := addMessage(c.ID, "user", "Message 1")
	m2, _ := addMessage(c.ID, "model", "Response 1")
	m3, _ := addMessage(c.ID, "user", "Message 2")
	addMessage(c.ID, "model", "Response 2") // Not copied

	fork, err := forkConversation(c.ID, m3.ID)
	if err != nil {
		t.Fatalf("forkConversation failed: %v", err)
	}

	if fork.ParentID == nil || *fork.ParentID != c.ID {
		t.Errorf("Expected ParentID to be %s, got %v", c.ID, fork.ParentID)
	}

	msgs, err := getMessages(fork.ID)
	if err != nil {
		t.Fatalf("getMessages for fork failed: %v", err)
	}

	if len(msgs) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(msgs))
	}

	if msgs[0].Content != m1.Content || msgs[1].Content != m2.Content || msgs[2].Content != m3.Content {
		t.Errorf("Message content mismatch in fork: %+v", msgs)
	}
}

func TestForksTreeContextAndPagination(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	// Create 25 root conversations
	var rootIDs []string
	for i := 1; i <= 25; i++ {
		c, err := createConversation(strconv.Itoa(i), `{"temp":0.7}`)
		if err != nil {
			t.Fatalf("Failed to create conversation %d: %v", i, err)
		}
		rootIDs = append(rootIDs, c.ID)
	}

	// Test root pagination
	list1, err := getConversationsPaginated(20, 0)
	if err != nil {
		t.Fatalf("getConversationsPaginated failed: %v", err)
	}
	if len(list1) != 20 {
		t.Errorf("Expected 20 conversations in page 1, got %d", len(list1))
	}

	list2, err := getConversationsPaginated(20, 20)
	if err != nil {
		t.Fatalf("getConversationsPaginated page 2 failed: %v", err)
	}
	if len(list2) != 5 {
		t.Errorf("Expected 5 conversations in page 2, got %d", len(list2))
	}

	pID := rootIDs[0]

	// Create message in parent
	m1, _ := addMessage(pID, "user", "Hello")
	f1, err := forkConversation(pID, m1.ID)
	if err != nil {
		t.Fatalf("forkConversation failed: %v", err)
	}

	// Create message in f1 and fork again (deep nested)
	m2, _ := addMessage(f1.ID, "user", "Nested Hello")
	f2, err := forkConversation(f1.ID, m2.ID)
	if err != nil {
		t.Fatalf("nested fork failed: %v", err)
	}

	// Check forks
	forksOfP, err := getConversationForks(pID)
	if err != nil {
		t.Fatalf("getConversationForks failed: %v", err)
	}
	if len(forksOfP) != 1 || forksOfP[0].ID != f1.ID {
		t.Errorf("Expected f1 to be fork of parent, got: %+v", forksOfP)
	}

	forksOfF1, _ := getConversationForks(f1.ID)
	if len(forksOfF1) != 1 || forksOfF1[0].ID != f2.ID {
		t.Errorf("Expected f2 to be fork of f1, got: %+v", forksOfF1)
	}

	// Test Tree Context for deep nested f2
	tree, err := getConversationTreeContext(f2.ID)
	if err != nil {
		t.Fatalf("getConversationTreeContext failed: %v", err)
	}
	// Tree context of f2 should contain: f2, f1, and pID.
	// Plus their direct siblings (if any, none here).
	ids := make(map[string]bool)
	for _, tc := range tree {
		ids[tc.ID] = true
	}
	if !ids[pID] || !ids[f1.ID] || !ids[f2.ID] {
		t.Errorf("Tree context does not contain all ancestors, got: %+v", tree)
	}
}

func TestSPAHandler(t *testing.T) {
	// Create temporary directory structure to simulate web root
	tmpDir, err := os.MkdirTemp("", "web-test")
	if err != nil {
		t.Fatalf("Failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldWebRoot := webRoot
	webRoot = tmpDir
	defer func() { webRoot = oldWebRoot }()

	// Write mock index.html
	indexContent := `<!doctype html><html><head><title>Mock App</title></head><body><div id="root"></div></body></html>`
	if err := os.WriteFile(filepath.Join(tmpDir, "index.html"), []byte(indexContent), 0644); err != nil {
		t.Fatalf("Failed to write index.html: %v", err)
	}

	// Write mock asset file
	if err := os.MkdirAll(filepath.Join(tmpDir, "assets"), 0755); err != nil {
		t.Fatalf("Failed to create assets dir: %v", err)
	}
	assetContent := "body { color: red; }"
	if err := os.WriteFile(filepath.Join(tmpDir, "assets", "style.css"), []byte(assetContent), 0644); err != nil {
		t.Fatalf("Failed to write style.css: %v", err)
	}

	// 1. Test serving a static asset file
	req := httptest.NewRequest("GET", "/assets/style.css", nil)
	w := httptest.NewRecorder()
	spaHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for styles, got %d", resp.StatusCode)
	}
	if !strings.Contains(w.Body.String(), "body { color: red; }") {
		t.Errorf("Expected mock css content, got: %s", w.Body.String())
	}

	// 2. Test SPA fallback serves index.html with default base tag
	req = httptest.NewRequest("GET", "/conversations/123", nil)
	w = httptest.NewRecorder()
	spaHandler(w, req)

	resp = w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for SPA route, got %d", resp.StatusCode)
	}
	if !strings.Contains(w.Body.String(), `<base href="/">`) {
		t.Errorf("Expected base tag with /, got: %s", w.Body.String())
	}

	// 3. Test SPA fallback with X-Forwarded-Prefix injects correct base tag
	req = httptest.NewRequest("GET", "/conversations/123", nil)
	req.Header.Set("X-Forwarded-Prefix", "/u/omar/ai")
	w = httptest.NewRecorder()
	spaHandler(w, req)

	if !strings.Contains(w.Body.String(), `<base href="/u/omar/ai/">`) {
		t.Errorf("Expected base tag from X-Forwarded-Prefix, got: %s", w.Body.String())
	}

	// 4. Test SPA fallback for root path with prefix
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-Prefix", "/u/omar/ai")
	w = httptest.NewRecorder()
	spaHandler(w, req)

	if !strings.Contains(w.Body.String(), `<base href="/u/omar/ai/">`) {
		t.Errorf("Expected base tag for root, got: %s", w.Body.String())
	}
}
