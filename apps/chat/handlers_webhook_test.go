package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandleWebhook_IncomingMessage(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	payload := MessagePayload{
		ConversationID:   "c-webhook",
		ConversationName: "Webhook Chat",
		Participants: []ParticipantInfo{
			{UserID: "alice-uid", DisplayName: "Alice"},
			{UserID: "test-user-id", DisplayName: "Self"},
		},
		MessageID:  "msg-webhook-1",
		SenderID:   "alice-uid",
		SenderName: "Alice",
		Text:       "Hello from remote peer",
		SentAt:     time.Now().Format(time.RFC3339),
	}

	payloadBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/internal/magicbox-webhook", bytes.NewReader(payloadBytes))
	req.Header.Set("X-Magicbox-Webhook-Secret", "mock-webhook-secret")
	rr := httptest.NewRecorder()

	handleWebhook(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	// Verify conversation was created
	conv, _ := getConversation("c-webhook")
	if conv == nil {
		t.Fatalf("Expected conversation to be created via webhook")
	}

	// Verify message was saved
	msgs, _ := getMessages("c-webhook", "", 10)
	if len(msgs) != 1 || msgs[0].ID != "msg-webhook-1" || msgs[0].Text != "Hello from remote peer" {
		t.Errorf("Expected message to be saved locally, got %+v", msgs)
	}
}
