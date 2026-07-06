package main

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	var err error
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open sqlite memory database: %v", err)
	}
	db.SetMaxOpenConns(1)

	queries := []string{
		`PRAGMA foreign_keys = ON;`,
		`CREATE TABLE IF NOT EXISTS conversations (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS conversation_participants (
			conversation_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			display_name TEXT NOT NULL,
			invite_link TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (conversation_id, user_id),
			FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			conversation_id TEXT NOT NULL,
			sender_id TEXT NOT NULL,
			sender_name TEXT NOT NULL,
			text TEXT NOT NULL,
			attachment_name TEXT NOT NULL DEFAULT '',
			attachment_type TEXT NOT NULL DEFAULT '',
			attachment_path TEXT NOT NULL DEFAULT '',
			sent_at TEXT NOT NULL,
			is_read BOOLEAN NOT NULL DEFAULT 0,
			is_system BOOLEAN NOT NULL DEFAULT 0,
			FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS message_deliveries (
			message_id TEXT NOT NULL,
			recipient_id TEXT NOT NULL,
			status TEXT NOT NULL,
			attempts INTEGER NOT NULL DEFAULT 0,
			last_attempt TEXT,
			PRIMARY KEY (message_id, recipient_id),
			FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE
		)`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("Failed to run schema queries: %v", err)
		}
	}

	dbConn = db
	return db
}

func TestCreateConversationAndGet(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	convID := "conv-123"
	nowStr := time.Now().Format(time.RFC3339)

	err := createConversation(convID, "Alice Chat", nowStr)
	if err != nil {
		t.Fatalf("createConversation failed: %v", err)
	}

	err = addParticipant(convID, "alice-uid", "Alice", "magicbox://invite/alice-link")
	if err != nil {
		t.Fatalf("addParticipant failed: %v", err)
	}

	conv, err := getConversation(convID)
	if err != nil {
		t.Fatalf("getConversation failed: %v", err)
	}

	if conv.Name != "Alice Chat" {
		t.Errorf("Expected Alice Chat, got %+v", conv)
	}

	if len(conv.Participants) != 1 || conv.Participants[0].DisplayName != "Alice" {
		t.Errorf("Expected 1 participant named Alice, got %+v", conv.Participants)
	}
}

func TestRenameConversation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	convID := "conv-rename"
	nowStr := time.Now().Format(time.RFC3339)

	_ = createConversation(convID, "Old Name", nowStr)

	err := renameConversation(convID, "New Name")
	if err != nil {
		t.Fatalf("renameConversation failed: %v", err)
	}

	conv, err := getConversation(convID)
	if err != nil {
		t.Fatalf("getConversation failed: %v", err)
	}

	if conv.Name != "New Name" {
		t.Errorf("Expected New Name, got %s", conv.Name)
	}
}

func TestInsertAndGetMessages(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	convID := "conv-123"
	nowStr := time.Now().Format(time.RFC3339)

	_ = createConversation(convID, "Alice Chat", nowStr)
	_ = addParticipant(convID, "alice-uid", "Alice", "")

	msg := &Message{
		ID:             "msg-1",
		ConversationID: convID,
		SenderID:       "me-uid",
		SenderName:     "Me",
		Text:           "Hello Alice",
		SentAt:         nowStr,
		IsRead:         true,
	}

	err := insertMessage(msg)
	if err != nil {
		t.Fatalf("insertMessage failed: %v", err)
	}

	msgs, err := getMessages(convID, "", 50)
	if err != nil {
		t.Fatalf("getMessages failed: %v", err)
	}

	if len(msgs) != 1 || msgs[0].Text != "Hello Alice" {
		t.Errorf("Expected message Hello Alice, got %+v", msgs)
	}

	lastMsg, err := getLastMessage(convID)
	if err != nil {
		t.Fatalf("getLastMessage failed: %v", err)
	}
	if lastMsg.ID != "msg-1" {
		t.Errorf("Expected last message msg-1, got %+v", lastMsg)
	}
}

func TestUnreadCountAndMarkAsRead(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	convID := "conv-123"
	nowStr := time.Now().Format(time.RFC3339)

	_ = createConversation(convID, "Alice Chat", nowStr)

	// Received message (unread)
	msg1 := &Message{
		ID:             "msg-1",
		ConversationID: convID,
		SenderID:       "alice-uid",
		SenderName:     "Alice",
		Text:           "Hey there",
		SentAt:         nowStr,
		IsRead:         false,
	}
	_ = insertMessage(msg1)

	convs, err := listConversations()
	if err != nil {
		t.Fatalf("listConversations failed: %v", err)
	}

	if len(convs) != 1 || convs[0].UnreadCount != 1 {
		t.Errorf("Expected unread count to be 1, got %+v", convs)
	}

	_, err = markMessagesAsRead(convID)
	if err != nil {
		t.Fatalf("markMessagesAsRead failed: %v", err)
	}

	convs, err = listConversations()
	if err != nil {
		t.Fatalf("listConversations failed: %v", err)
	}

	if convs[0].UnreadCount != 0 {
		t.Errorf("Expected unread count to be 0 after marking as read, got %+v", convs)
	}
}

func TestSearchMessages(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	convID := "conv-search"
	nowStr := time.Now().Format(time.RFC3339)
	_ = createConversation(convID, "Search Test", nowStr)

	m1 := &Message{ID: "m1", ConversationID: convID, SenderID: "u1", SenderName: "A", Text: "Hello secret world", SentAt: nowStr, IsRead: true}
	m2 := &Message{ID: "m2", ConversationID: convID, SenderID: "u1", SenderName: "A", Text: "Just normal chat", SentAt: nowStr, IsRead: true}
	_ = insertMessage(m1)
	_ = insertMessage(m2)

	msgs, err := searchMessages(convID, "secret")
	if err != nil {
		t.Fatalf("searchMessages failed: %v", err)
	}

	if len(msgs) != 1 || msgs[0].ID != "m1" {
		t.Errorf("Expected only message m1 to match 'secret', got %+v", msgs)
	}
}

func TestGetSharedMedia(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	convID := "conv-media-test"
	nowStr := time.Now().Format(time.RFC3339)
	_ = createConversation(convID, "Media Test", nowStr)

	m1 := &Message{ID: "m1", ConversationID: convID, SenderID: "u1", SenderName: "A", Text: "", AttachmentName: "image.png", AttachmentType: "image/png", SentAt: nowStr, IsRead: true}
	m2 := &Message{ID: "m2", ConversationID: convID, SenderID: "u1", SenderName: "A", Text: "Hello standard text", SentAt: nowStr, IsRead: true}
	_ = insertMessage(m1)
	_ = insertMessage(m2)

	msgs, err := getSharedMedia(convID, "", 20)
	if err != nil {
		t.Fatalf("getSharedMedia failed: %v", err)
	}

	if len(msgs) != 1 || msgs[0].ID != "m1" {
		t.Errorf("Expected only media message m1, got %+v", msgs)
	}
}

func TestMessageDeliveries(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	convID := "conv-delivery-test"
	nowStr := time.Now().Format(time.RFC3339)
	_ = createConversation(convID, "Delivery Test", nowStr)

	m1 := &Message{ID: "m1", ConversationID: convID, SenderID: "u1", SenderName: "A", Text: "Hi", SentAt: nowStr, IsRead: true}
	_ = insertMessage(m1)

	// Insert delivery entries
	err := insertMessageDelivery("m1", "recipient1", "pending")
	if err != nil {
		t.Fatalf("insertMessageDelivery failed: %v", err)
	}
	_ = insertMessageDelivery("m1", "recipient2", "pending")

	// Get pending deliveries
	pd, err := getPendingDeliveries()
	if err != nil {
		t.Fatalf("getPendingDeliveries failed: %v", err)
	}
	if len(pd) != 2 {
		t.Errorf("Expected 2 pending deliveries, got %d", len(pd))
	}

	// Update delivery status
	err = updateMessageDeliveryStatus("m1", "recipient1", "failed")
	if err != nil {
		t.Fatalf("updateMessageDeliveryStatus failed: %v", err)
	}

	pd, err = getPendingDeliveries()
	if err != nil {
		t.Fatalf("getPendingDeliveries failed: %v", err)
	}
	if len(pd) != 2 {
		t.Errorf("Expected still 2 pending/failed deliveries, got %d", len(pd))
	}
	if pd[0].RecipientID == "recipient1" && pd[0].Attempts != 1 {
		t.Errorf("Expected recipient1 attempts to be 1, got %d", pd[0].Attempts)
	}

	// Delete successful delivery
	err = deleteMessageDelivery("m1", "recipient2")
	if err != nil {
		t.Fatalf("deleteMessageDelivery failed: %v", err)
	}

	pd, err = getPendingDeliveries()
	if err != nil {
		t.Fatalf("getPendingDeliveries failed: %v", err)
	}
	if len(pd) != 1 {
		t.Errorf("Expected 1 pending/failed delivery left, got %d", len(pd))
	}
	if pd[0].RecipientID != "recipient1" {
		t.Errorf("Expected only recipient1 left in queue, got %s", pd[0].RecipientID)
	}
}
