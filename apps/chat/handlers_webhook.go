package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// POST /internal/magicbox-webhook
func handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Verify webhook token if needed
	if env != nil && !env.VerifyWebhook(r) {
		writeError(w, http.StatusUnauthorized, "invalid webhook secret")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body: "+err.Error())
		return
	}

	var payload MessagePayload
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("Webhook error: failed to unmarshal MessagePayload: %v", err)
		// Return 200 to prevent core from retrying corrupt payloads
		w.WriteHeader(http.StatusOK)
		return
	}

	// Check if conversation exists
	conv, err := getConversation(payload.ConversationID)
	if err != nil {
		log.Printf("Webhook DB error: failed to query conversation: %v", err)
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	// Create conversation and participants if they don't exist
	if conv == nil {
		log.Printf("Webhook: received message for new conversation %s (%s)", payload.ConversationName, payload.ConversationID)
		
		convName := ""
		if len(payload.Participants) > 2 {
			convName = payload.ConversationName
		}

		if err := createConversation(payload.ConversationID, convName, payload.SentAt); err != nil {
			log.Printf("Webhook DB error: failed to create conversation: %v", err)
			writeError(w, http.StatusInternalServerError, "db error")
			return
		}

		// Save participants
		for _, p := range payload.Participants {
			_ = addParticipant(payload.ConversationID, p.UserID, p.DisplayName, p.InviteLink)
		}
	} else {
		// Conversation already exists, check if name has been explicitly set/updated by a peer in a group chat (>=3 participants)
		if len(conv.Participants) >= 3 && payload.ConversationName != "" && conv.Name != payload.ConversationName {
			log.Printf("Webhook: updating conversation %s name to %s", conv.ID, payload.ConversationName)
			if err := renameConversation(conv.ID, payload.ConversationName); err == nil {
				conv.Name = payload.ConversationName
			}
		}
	}

	// Check if message is already in DB to prevent duplicates
	existingMsgs, err := getMessages(payload.ConversationID, "", 100)
	if err == nil {
		for _, m := range existingMsgs {
			if m.ID == payload.MessageID {
				log.Printf("Webhook info: ignored duplicate message ID %s", payload.MessageID)
				w.WriteHeader(http.StatusOK)
				return
			}
		}
	}

	var savedPath string
	if len(payload.AttachmentData) > 0 && payload.AttachmentName != "" {
		convDir := filepath.Join("/data/shared/storage/Chat", payload.ConversationID)
		if err := os.MkdirAll(convDir, 0755); err == nil {
			savedPath = filepath.Join(convDir, payload.AttachmentName)
			if err := os.WriteFile(savedPath, payload.AttachmentData, 0644); err != nil {
				log.Printf("Webhook warning: failed to write attachment to disk: %v", err)
				savedPath = ""
			}
		} else {
			log.Printf("Webhook warning: failed to create attachment folder: %v", err)
		}
	}

	// Insert message into local DB
	msg := &Message{
		ID:             payload.MessageID,
		ConversationID: payload.ConversationID,
		SenderID:       payload.SenderID,
		SenderName:     payload.SenderName,
		Text:           payload.Text,
		AttachmentName: payload.AttachmentName,
		AttachmentType: payload.AttachmentType,
		AttachmentPath: savedPath,
		SentAt:         payload.SentAt,
		IsRead:         false, // Receivers see it as unread
		IsSystem:       payload.IsSystem,
	}

	if err := insertMessage(msg); err != nil {
		log.Printf("Webhook DB error: failed to save message: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to save message")
		return
	}

	log.Printf("Webhook success: saved message %s from %s in conv %s", payload.MessageID, payload.SenderName, payload.ConversationID)

	// Notify active UI sessions
	notifyClients()

	w.WriteHeader(http.StatusOK)
}
