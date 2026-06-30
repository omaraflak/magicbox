package main

import (
	"database/sql"
	"strconv"
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

// --- DB Settings Tests ---

func TestGetSetting_NotExists(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	val := getSetting("non_existent")
	if val != "" {
		t.Errorf("Expected empty string for non-existent setting, got %s", val)
	}
}

func TestSetSetting_NewAndGet(t *testing.T) {
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
}

func TestSetSetting_Overwrite(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	if err := setSetting("api_key", "key-1"); err != nil {
		t.Fatalf("first setSetting failed: %v", err)
	}
	if err := setSetting("api_key", "key-2"); err != nil {
		t.Fatalf("second setSetting failed: %v", err)
	}

	val := getSetting("api_key")
	if val != "key-2" {
		t.Errorf("Expected key-2, got %s", val)
	}
}

// --- DB Conversations Tests ---

func TestCreateConversation(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	c, err := createConversation("Test Chat", `{"temp":0.7}`)
	if err != nil {
		t.Fatalf("createConversation failed: %v", err)
	}
	if c == nil {
		t.Fatal("Expected created conversation to be non-nil")
	}
	if c.Title != "Test Chat" {
		t.Errorf("Expected Title 'Test Chat', got %s", c.Title)
	}
	if c.Params != `{"temp":0.7}` {
		t.Errorf("Expected params '{\"temp\":0.7}', got %s", c.Params)
	}
	if c.ParentID != nil {
		t.Errorf("Expected ParentID to be nil, got %v", c.ParentID)
	}
}

func TestGetConversation_NotExists(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	c, err := getConversation("non_existent_id")
	if err == nil || c != nil {
		t.Errorf("Expected error and nil conversation for non-existent ID, got err=%v, conv=%v", err, c)
	}
}

func TestGetConversation_Exists(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	cCreated, err := createConversation("Test Chat", `{"temp":0.7}`)
	if err != nil {
		t.Fatalf("createConversation failed: %v", err)
	}

	cFetched, err := getConversation(cCreated.ID)
	if err != nil {
		t.Fatalf("getConversation failed: %v", err)
	}
	if cFetched.ID != cCreated.ID || cFetched.Title != cCreated.Title {
		t.Errorf("Fetched conversation does not match created: %+v vs %+v", cFetched, cCreated)
	}
}

func TestUpdateConversationParams(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	c, err := createConversation("Test Chat", `{"temp":0.7}`)
	if err != nil {
		t.Fatalf("createConversation failed: %v", err)
	}

	err = updateConversationParams(c.ID, `{"temp":0.9}`)
	if err != nil {
		t.Fatalf("updateConversationParams failed: %v", err)
	}

	cUpdated, err := getConversation(c.ID)
	if err != nil {
		t.Fatalf("getConversation failed: %v", err)
	}
	if cUpdated.Params != `{"temp":0.9}` {
		t.Errorf("Expected params to be updated to '{\"temp\":0.9}', got %s", cUpdated.Params)
	}
}

func TestUpdateConversationTitle(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	c, err := createConversation("Test Chat", `{"temp":0.7}`)
	if err != nil {
		t.Fatalf("createConversation failed: %v", err)
	}

	err = updateConversationTitle(c.ID, "New Chat Title")
	if err != nil {
		t.Fatalf("updateConversationTitle failed: %v", err)
	}

	cUpdated, err := getConversation(c.ID)
	if err != nil {
		t.Fatalf("getConversation failed: %v", err)
	}
	if cUpdated.Title != "New Chat Title" {
		t.Errorf("Expected title to be updated to 'New Chat Title', got %s", cUpdated.Title)
	}
}

func TestGetConversationsPaginated(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	// Ensure empty db returns empty list
	list, err := getConversationsPaginated(10, 0)
	if err != nil {
		t.Fatalf("getConversationsPaginated failed: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("Expected empty list, got %d items", len(list))
	}

	// Create 25 root conversations
	for i := 1; i <= 25; i++ {
		_, err := createConversation(strconv.Itoa(i), `{"temp":0.7}`)
		if err != nil {
			t.Fatalf("Failed to create conversation %d: %v", i, err)
		}
	}

	// Test pagination limits & offsets
	list1, err := getConversationsPaginated(20, 0)
	if err != nil {
		t.Fatalf("getConversationsPaginated page 1 failed: %v", err)
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
}

func TestGetConversationsPaginated_OnlyRoots(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	parent, err := createConversation("Parent", `{"temp":0.7}`)
	if err != nil {
		t.Fatalf("failed to create parent: %v", err)
	}

	m, _ := addMessage(parent.ID, "user", "hi")
	_, err = forkConversation(parent.ID, m.ID)
	if err != nil {
		t.Fatalf("failed to fork: %v", err)
	}

	// Paginated list should only contain parent, not forks (because parent_id is not null for forks)
	list, err := getConversationsPaginated(10, 0)
	if err != nil {
		t.Fatalf("failed to get paginated conversations: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("Expected only 1 root conversation, got %d", len(list))
	}
	if list[0].ID != parent.ID {
		t.Errorf("Expected conversation in list to be parent, got %s", list[0].ID)
	}
}

// --- DB Messages Tests ---

func TestAddMessageAndGet(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	c, _ := createConversation("Test Chat", `{"temp":0.7}`)

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
	if len(msgs) != 1 || msgs[0].Content != "Hello AI" || msgs[0].Role != "user" {
		t.Errorf("Expected 1 message with content 'Hello AI' and role 'user', got %+v", msgs)
	}
}

func TestGetMessages_Empty(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	c, _ := createConversation("Test Chat", `{"temp":0.7}`)
	msgs, err := getMessages(c.ID)
	if err != nil {
		t.Fatalf("getMessages failed: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(msgs))
	}
}

func TestDeleteMessage(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	c, _ := createConversation("Test Chat", `{"temp":0.7}`)
	m, err := addMessage(c.ID, "user", "Hello AI")
	if err != nil {
		t.Fatalf("addMessage failed: %v", err)
	}

	err = deleteMessage(m.ID)
	if err != nil {
		t.Fatalf("deleteMessage failed: %v", err)
	}

	msgs, err := getMessages(c.ID)
	if err != nil {
		t.Fatalf("getMessages failed: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("Expected 0 messages after deleting the only message, got %d", len(msgs))
	}
}

func TestDeleteConversation_Cascade(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	c, _ := createConversation("Test Chat", `{"temp":0.7}`)
	_, _ = addMessage(c.ID, "user", "Hello AI")

	err := deleteConversation(c.ID)
	if err != nil {
		t.Fatalf("deleteConversation failed: %v", err)
	}

	cDeleted, err := getConversation(c.ID)
	if err == nil && cDeleted != nil {
		t.Errorf("Expected error or nil when fetching deleted conversation")
	}

	msgs, err := getMessages(c.ID)
	if err != nil {
		t.Fatalf("getMessages failed: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("Expected 0 messages after conversation deletion, got %d", len(msgs))
	}
}

// --- DB Presets Tests ---

func TestCreateAndGetPresets(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	p, err := createPreset("Preset 1", `{"temp":0.5}`)
	if err != nil {
		t.Fatalf("createPreset failed: %v", err)
	}
	if p.Name != "Preset 1" || p.Params != `{"temp":0.5}` {
		t.Errorf("Unexpected preset fields: %+v", p)
	}

	presets, err := getPresets()
	if err != nil {
		t.Fatalf("getPresets failed: %v", err)
	}
	if len(presets) != 1 || presets[0].Name != "Preset 1" {
		t.Errorf("Unexpected presets list: %+v", presets)
	}
}

func TestUpdatePreset(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	p, _ := createPreset("Preset 1", `{"temp":0.5}`)

	err := updatePreset(p.ID, "Updated Preset", `{"temp":0.8}`)
	if err != nil {
		t.Fatalf("updatePreset failed: %v", err)
	}

	presets, err := getPresets()
	if err != nil {
		t.Fatalf("getPresets failed: %v", err)
	}
	if len(presets) != 1 || presets[0].Name != "Updated Preset" || presets[0].Params != `{"temp":0.8}` {
		t.Errorf("Unexpected preset fields after update: %+v", presets[0])
	}
}

func TestDeletePreset(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	p, _ := createPreset("Preset 1", `{"temp":0.5}`)

	err := deletePreset(p.ID)
	if err != nil {
		t.Fatalf("deletePreset failed: %v", err)
	}

	presets, err := getPresets()
	if err != nil {
		t.Fatalf("getPresets failed: %v", err)
	}
	if len(presets) != 0 {
		t.Errorf("Expected 0 presets, got %d", len(presets))
	}
}

// --- DB Forks / Tree Context / Undo Turn Tests ---

func TestForkConversation_Success(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	parent, _ := createConversation("Parent Chat", `{"temp":0.5}`)
	m1, _ := addMessage(parent.ID, "user", "Message 1")
	m2, _ := addMessage(parent.ID, "model", "Response 1")
	m3, _ := addMessage(parent.ID, "user", "Message 2")
	addMessage(parent.ID, "model", "Response 2") // Not copied

	fork, err := forkConversation(parent.ID, m3.ID)
	if err != nil {
		t.Fatalf("forkConversation failed: %v", err)
	}

	if fork.ParentID == nil || *fork.ParentID != parent.ID {
		t.Errorf("Expected ParentID to be %s, got %v", parent.ID, fork.ParentID)
	}
	if fork.Title != "Parent Chat (Fork)" {
		t.Errorf("Expected fork title to be 'Parent Chat (Fork)', got %s", fork.Title)
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

func TestGetConversationForks(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	parent, _ := createConversation("Parent", `{"temp":0.5}`)
	m1, _ := addMessage(parent.ID, "user", "Hello")

	fork1, err := forkConversation(parent.ID, m1.ID)
	if err != nil {
		t.Fatalf("fork 1 failed: %v", err)
	}

	fork2, err := forkConversation(parent.ID, m1.ID)
	if err != nil {
		t.Fatalf("fork 2 failed: %v", err)
	}

	forks, err := getConversationForks(parent.ID)
	if err != nil {
		t.Fatalf("getConversationForks failed: %v", err)
	}

	if len(forks) != 2 {
		t.Errorf("Expected 2 forks, got %d", len(forks))
	}

	ids := map[string]bool{forks[0].ID: true, forks[1].ID: true}
	if !ids[fork1.ID] || !ids[fork2.ID] {
		t.Errorf("Forks do not match created fork IDs: %+v", forks)
	}
}

func TestGetConversationTreeContext(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	parent, _ := createConversation("Parent", `{"temp":0.5}`)
	m1, _ := addMessage(parent.ID, "user", "Hello")

	fork1, _ := forkConversation(parent.ID, m1.ID)
	m2, _ := addMessage(fork1.ID, "user", "Nested Hello")

	fork2, _ := forkConversation(fork1.ID, m2.ID)

	tree, err := getConversationTreeContext(fork2.ID)
	if err != nil {
		t.Fatalf("getConversationTreeContext failed: %v", err)
	}

	ids := make(map[string]bool)
	for _, tc := range tree {
		ids[tc.ID] = true
	}

	if !ids[parent.ID] || !ids[fork1.ID] || !ids[fork2.ID] {
		t.Errorf("Tree context does not contain all ancestors, got: %+v", tree)
	}
}

func TestRemoveLastTurn_ModelMessage(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	c, _ := createConversation("Test Chat", `{"temp":0.7}`)
	_, _ = addMessage(c.ID, "user", "User Message")
	_, _ = addMessage(c.ID, "model", "Model Response")

	content, err := removeLastTurn(c.ID)
	if err != nil {
		t.Fatalf("removeLastTurn failed: %v", err)
	}
	if content != "User Message" {
		t.Errorf("Expected returned content to be 'User Message', got '%s'", content)
	}

	msgs, err := getMessages(c.ID)
	if err != nil {
		t.Fatalf("getMessages failed: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("Expected both user and model message to be removed, remaining: %d", len(msgs))
	}
}

func TestRemoveLastTurn_UserMessageOnly(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	c, _ := createConversation("Test Chat", `{"temp":0.7}`)
	_, _ = addMessage(c.ID, "user", "User Message Only")

	content, err := removeLastTurn(c.ID)
	if err != nil {
		t.Fatalf("removeLastTurn failed: %v", err)
	}
	if content != "User Message Only" {
		t.Errorf("Expected returned content to be 'User Message Only', got '%s'", content)
	}

	msgs, err := getMessages(c.ID)
	if err != nil {
		t.Fatalf("getMessages failed: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("Expected user message to be removed, remaining: %d", len(msgs))
	}
}

func TestRemoveLastTurn_Empty(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	c, _ := createConversation("Test Chat", `{"temp":0.7}`)

	_, err := removeLastTurn(c.ID)
	if err == nil {
		t.Errorf("Expected error when removing last turn of empty conversation")
	}
}

func TestRemoveLastTurn_LastNotFromUser(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	c, _ := createConversation("Test Chat", `{"temp":0.7}`)
	_, _ = addMessage(c.ID, "model", "Model Message first")

	_, err := removeLastTurn(c.ID)
	if err == nil {
		t.Errorf("Expected error when last user message is not found after model message deletion")
	}
}
