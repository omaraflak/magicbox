package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	pb "github.com/magicbox/core/api/proto/v1"
)

func TestHandleConversations_Get(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	nowStr := time.Now().Format(time.RFC3339)
	_ = createConversation("c-1", "Direct Message", nowStr)

	req := httptest.NewRequest("GET", "/api/conversations", nil)
	rr := httptest.NewRecorder()

	handleConversations(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", rr.Code)
	}

	var convs []Conversation
	if err := json.Unmarshal(rr.Body.Bytes(), &convs); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(convs) != 1 || convs[0].ID != "c-1" {
		t.Errorf("Unexpected conversations list: %+v", convs)
	}
}

func TestHandleConversations_Post(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Create 1-to-1 conversation with Alice
	body := `{"participant_ids":["c1"]}`
	req := httptest.NewRequest("POST", "/api/conversations", strings.NewReader(body))
	rr := httptest.NewRecorder()

	handleConversations(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected 201 Created, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var conv Conversation
	if err := json.Unmarshal(rr.Body.Bytes(), &conv); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(conv.Participants) != 2 {
		t.Errorf("Expected 2 participants (self + Alice), got %d", len(conv.Participants))
	}
}

func TestHandleConversationRoutes_Delete(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	nowStr := time.Now().Format(time.RFC3339)
	_ = createConversation("c-delete", "To Delete", nowStr)

	req := httptest.NewRequest("DELETE", "/api/conversations/c-delete", nil)
	rr := httptest.NewRecorder()

	handleConversationRoutes(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", rr.Code)
	}

	conv, err := getConversation("c-delete")
	if err != nil || conv != nil {
		t.Errorf("Expected conversation to be deleted, got %+v (err=%v)", conv, err)
	}
}

func TestHandleConversationRoutes_Rename(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	nowStr := time.Now().Format(time.RFC3339)
	_ = createConversation("c-rename", "Old Name", nowStr)
	// Add participants to make it a group chat (3 or more)
	_ = addParticipant("c-rename", "self-uid", "Self", "")
	_ = addParticipant("c-rename", "alice-uid", "Alice", "")
	_ = addParticipant("c-rename", "bob-uid", "Bob", "")

	body := `{"name":"New Group Name"}`
	req := httptest.NewRequest("POST", "/api/conversations/c-rename/rename", strings.NewReader(body))
	rr := httptest.NewRecorder()

	handleConversationRoutes(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	conv, _ := getConversation("c-rename")
	if conv.Name != "New Group Name" {
		t.Errorf("Expected name to be renamed to New Group Name, got %s", conv.Name)
	}
}

func TestHandleMessages_Get(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	nowStr := time.Now().Format(time.RFC3339)
	_ = createConversation("c-messages", "Chat", nowStr)
	_ = insertMessage(&Message{ID: "m1", ConversationID: "c-messages", SenderID: "u1", SenderName: "A", Text: "Hello", SentAt: nowStr})

	req := httptest.NewRequest("GET", "/api/conversations/c-messages/messages", nil)
	rr := httptest.NewRecorder()

	handleConversationRoutes(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", rr.Code)
	}

	var msgs []Message
	if err := json.Unmarshal(rr.Body.Bytes(), &msgs); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(msgs) != 1 || msgs[0].ID != "m1" {
		t.Errorf("Unexpected messages returned: %+v", msgs)
	}
}

func TestHandleMessages_Post(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	nowStr := time.Now().Format(time.RFC3339)
	_ = createConversation("c-send", "Chat", nowStr)
	_ = addParticipant("c-send", "test-user-id", "Self", "")
	_ = addParticipant("c-send", "alice-uid", "Alice", "")

	body := `{"text":"Hello Alice"}`
	req := httptest.NewRequest("POST", "/api/conversations/c-send/messages", strings.NewReader(body))
	rr := httptest.NewRecorder()

	handleConversationRoutes(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected 201 Created, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	// Verify message in database
	msgs, _ := getMessages("c-send", "", 10)
	if len(msgs) != 1 || msgs[0].Text != "Hello Alice" {
		t.Errorf("Expected message to be saved locally, got %+v", msgs)
	}

	// Verify delivery was queued
	pd, _ := getPendingDeliveries()
	if len(pd) != 1 || pd[0].RecipientID != "alice-uid" {
		t.Errorf("Expected delivery to be queued for Alice, got %+v", pd)
	}
}

func TestHandleConversations_Post_AppNotInstalled(t *testing.T) {
	mockServer, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Configure mock to return Installed: false
	mockServer.isAppInstalledFunc = func(ctx context.Context, req *pb.IsAppInstalledRequest) (*pb.IsAppInstalledResponse, error) {
		return &pb.IsAppInstalledResponse{Installed: false}, nil
	}

	body := `{"participant_ids":["c1"]}`
	req := httptest.NewRequest("POST", "/api/conversations", strings.NewReader(body))
	rr := httptest.NewRecorder()

	handleConversations(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request, got %d", rr.Code)
	}

	if !strings.Contains(rr.Body.String(), "does not have Magic Chat installed") {
		t.Errorf("Expected error message about app installation, got: %s", rr.Body.String())
	}
}
