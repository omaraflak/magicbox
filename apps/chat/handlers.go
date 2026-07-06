package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	pb "github.com/magicbox/core/api/proto/v1"
	"github.com/magicbox/core/sdk"
)

// Structures for REST API

type CreateConvRequest struct {
	Name           string   `json:"name"`
	ParticipantIDs []string `json:"participant_ids"`
}

type MessagePayload struct {
	ConversationID   string            `json:"conversation_id"`
	ConversationName string            `json:"conversation_name,omitempty"`
	Participants     []ParticipantInfo `json:"participants"`
	MessageID        string            `json:"message_id"`
	SenderID         string            `json:"sender_id"`
	SenderName       string            `json:"sender_name"`
	Text             string            `json:"text"`
	AttachmentName   string            `json:"attachment_name,omitempty"`
	AttachmentType   string            `json:"attachment_type,omitempty"`
	AttachmentData   []byte            `json:"attachment_data,omitempty"`
	SentAt           string            `json:"sent_at"`
}

type ParticipantInfo struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
	InviteLink  string `json:"invite_link,omitempty"`
}

// Helpers

func getCoreClient() (pb.MagicboxOSClient, io.Closer, context.Context, error) {
	if env == nil {
		return nil, nil, nil, os.ErrNotExist
	}
	client, conn, ctx, err := env.GetCoreClient()
	return client, conn, ctx, err
}


func jsonEncode(w http.ResponseWriter, data interface{}) error {
	return json.NewEncoder(w).Encode(data)
}

// --- Handler Functions ---

// GET /api/profile
func handleProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if err := env.EnsureScopes([]string{"profile:read"}, []string{"Read your basic user profile (username, user ID)"}); err != nil {
		if sdk.WriteConsentError(w, err) {
			return
		}
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	client, conn, ctx, err := getCoreClient()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to connect to core: "+err.Error())
		return
	}
	defer conn.Close()

	resp, err := client.GetProfile(ctx, &pb.GetProfileRequest{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get profile from core: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"user_id":    resp.UserId,
		"username":   resp.Username,
		"created_at": resp.CreatedAt,
	})
}

// GET /api/contacts
func handleContacts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if err := env.EnsureScopes([]string{"contacts:read"}, []string{"Access contacts to display names and profile invite links"}); err != nil {
		if sdk.WriteConsentError(w, err) {
			return
		}
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	client, conn, ctx, err := getCoreClient()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to connect to core: "+err.Error())
		return
	}
	defer conn.Close()

	resp, err := client.ListContacts(ctx, &pb.ListContactsRequest{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get contacts from core: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp.Contacts)
}

// GET /api/conversations
// POST /api/conversations
func handleConversations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := listConversations()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list conversations: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, list)

	case http.MethodPost:
		var req CreateConvRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}

		if err := env.EnsureScopes([]string{"profile:read", "contacts:read"}, []string{"Read your basic user profile", "Access contacts to display names and profile invite links"}); err != nil {
			if sdk.WriteConsentError(w, err) {
				return
			}
			writeError(w, http.StatusForbidden, err.Error())
			return
		}

		client, conn, ctx, err := getCoreClient()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to connect to core: "+err.Error())
			return
		}
		defer conn.Close()

		// Get all contacts to map contact IDs to target user IDs
		contactsResp, err := client.ListContacts(ctx, &pb.ListContactsRequest{})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load contacts: "+err.Error())
			return
		}

		// Get our own profile to add ourself as participant
		profile, err := client.GetProfile(ctx, &pb.GetProfileRequest{})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load profile: "+err.Error())
			return
		}

		// Create participant map and slice
		participants := []Participant{}
		
		// Add ourself first
		participants = append(participants, Participant{
			UserID:      profile.UserId,
			DisplayName: profile.Username,
			InviteLink:  "", // Will be populated during broadcast
		})

		// Find target details for selected contact IDs
		for _, cid := range req.ParticipantIDs {
			var foundContact *pb.Contact
			for _, c := range contactsResp.Contacts {
				if c.Id == cid {
					foundContact = c
					break
				}
			}
			if foundContact != nil {
				participants = append(participants, Participant{
					UserID:      foundContact.TargetUserId,
					DisplayName: foundContact.DisplayName,
					InviteLink:  foundContact.InviteLink,
				})
			}
		}

		// Ensure we have at least 1 other participant
		if len(participants) < 2 {
			writeError(w, http.StatusBadRequest, "must have at least one recipient participant")
			return
		}

		// Deduplicate participants by User ID
		uniqueParts := []Participant{}
		seen := make(map[string]bool)
		for _, p := range participants {
			if !seen[p.UserID] {
				seen[p.UserID] = true
				uniqueParts = append(uniqueParts, p)
			}
		}

		// Conversation Name: if 1-to-1, it is always empty in DB. 
		// If group, use the explicit name provided, otherwise save as empty.
		convName := ""
		if len(uniqueParts) > 2 {
			convName = strings.TrimSpace(req.Name)
		}

		// Check if a 1-to-1 conversation with this contact already exists to reuse it
		if len(uniqueParts) == 2 {
			var recipientID string
			for _, p := range uniqueParts {
				if p.UserID != profile.UserId {
					recipientID = p.UserID
					break
				}
			}

			existingConvs, err := listConversations()
			if err == nil {
				for _, ec := range existingConvs {
					if len(ec.Participants) == 2 {
						for _, ep := range ec.Participants {
							if ep.UserID == recipientID {
								// Found existing 1-to-1 conversation, return it!
								writeJSON(w, http.StatusOK, ec)
								return
							}
						}
					}
				}
			}
		}

		convID := uuid.NewString()
		nowStr := time.Now().Format(time.RFC3339)

		if err := createConversation(convID, convName, nowStr); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create conversation: "+err.Error())
			return
		}

		for _, p := range uniqueParts {
			_ = addParticipant(convID, p.UserID, p.DisplayName, p.InviteLink)
		}

		newConv, err := getConversation(convID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load new conversation: "+err.Error())
			return
		}

		writeJSON(w, http.StatusCreated, newConv)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// Routes starting with /api/conversations/
func handleConversationRoutes(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/conversations/"), "/")
	if len(parts) < 1 || parts[0] == "" {
		writeError(w, http.StatusBadRequest, "invalid path")
		return
	}

	convID := parts[0]
	conv, err := getConversation(convID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check conversation: "+err.Error())
		return
	}
	if conv == nil {
		writeError(w, http.StatusNotFound, "conversation not found")
		return
	}

	// Route based on subpath
	if len(parts) == 1 {
		// DELETE /api/conversations/{id}
		if r.Method == http.MethodDelete {
			if err := deleteConversation(convID); err != nil {
				writeError(w, http.StatusInternalServerError, "failed to delete conversation: "+err.Error())
				return
			}
			// Clean up files for this conversation
			os.RemoveAll(filepath.Join("/data/shared/storage/Chat", convID))
			writeJSON(w, http.StatusOK, map[string]string{"message": "conversation deleted"})
			return
		}
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	subRoute := parts[1]
	switch subRoute {
	case "messages":
		handleMessages(w, r, conv)
	case "attachments":
		handleAttachments(w, r, conv)
	case "read":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if err := markMessagesAsRead(convID); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to mark as read: "+err.Error())
			return
		}
		notifyClients()
		writeJSON(w, http.StatusOK, map[string]string{"message": "messages marked as read"})
	case "rename":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}
		name := strings.TrimSpace(req.Name)
		if name == "" {
			writeError(w, http.StatusBadRequest, "name cannot be empty")
			return
		}

		if len(conv.Participants) < 3 {
			writeError(w, http.StatusBadRequest, "cannot rename 1-to-1 conversations")
			return
		}

		client, conn, ctx, err := getCoreClient()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to contact core client: "+err.Error())
			return
		}
		defer conn.Close()

		profile, err := client.GetProfile(ctx, &pb.GetProfileRequest{})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to get profile: "+err.Error())
			return
		}

		if err := renameConversation(conv.ID, name); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to rename conversation: "+err.Error())
			return
		}

		sysMsg := &Message{
			ID:             uuid.New().String(),
			ConversationID: conv.ID,
			SenderID:       profile.UserId,
			SenderName:     profile.Username,
			Text:           profile.Username + " renamed the chat to \"" + name + "\"",
			SentAt:         time.Now().Format(time.RFC3339),
			IsRead:         true,
		}
		if err := insertMessage(sysMsg); err != nil {
			log.Printf("Rename warning: failed to save system message: %v", err)
		}

		conv.Name = name
		go broadcastMessage(conv, sysMsg, nil, profile.UserId, profile.Username)

		notifyClients()
		writeJSON(w, http.StatusOK, map[string]string{"message": "conversation renamed successfully"})
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

// GET /api/conversations/{id}/messages
// POST /api/conversations/{id}/messages
func handleMessages(w http.ResponseWriter, r *http.Request, conv *Conversation) {
	switch r.Method {
	case http.MethodGet:
		// Mark messages as read first when opening conversation
		_ = markMessagesAsRead(conv.ID)
		notifyClients()

		q := r.URL.Query().Get("q")
		if q != "" {
			msgs, err := searchMessages(conv.ID, q)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to search messages: "+err.Error())
				return
			}
			writeJSON(w, http.StatusOK, msgs)
			return
		}

		before := r.URL.Query().Get("before")
		limitStr := r.URL.Query().Get("limit")
		limit := 50
		if limitStr != "" {
			if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
				limit = parsed
			}
		}

		msgs, err := getMessages(conv.ID, before, limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to get messages: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, msgs)

	case http.MethodPost:
		// Send message
		var text string
		var attachName string
		var attachType string
		var attachBytes []byte

		// Check if it is a multipart request (with attachments)
		if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB max in-memory
				writeError(w, http.StatusBadRequest, "failed to parse multipart form: "+err.Error())
				return
			}
			text = r.FormValue("text")

			file, header, err := r.FormFile("attachment")
			if err == nil {
				defer file.Close()
				attachName = filepath.Base(header.Filename)
				attachType = header.Header.Get("Content-Type")

				// Read all bytes
				bytes, readErr := io.ReadAll(file)
				if readErr != nil {
					writeError(w, http.StatusInternalServerError, "failed to read file: "+readErr.Error())
					return
				}
				attachBytes = bytes
			}
		} else {
			// Plain JSON payload
			var msgReq struct {
				Text string `json:"text"`
			}
			if err := json.NewDecoder(r.Body).Decode(&msgReq); err != nil {
				writeError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
				return
			}
			text = msgReq.Text
		}

		if text == "" && len(attachBytes) == 0 {
			writeError(w, http.StatusBadRequest, "empty message")
			return
		}

		// Check scopes dynamic requirements
		var scopes []string
		var reasons []string
		scopes = append(scopes, "profile:read")
		reasons = append(reasons, "Read your basic user profile to set sender ID and display name")

		if len(attachBytes) > 0 {
			scopes = append(scopes, "shared:storage:rw")
			reasons = append(reasons, "Access storage to save and send file attachments")
		}

		if err := env.EnsureScopes(scopes, reasons); err != nil {
			if sdk.WriteConsentError(w, err) {
				return
			}
			writeError(w, http.StatusForbidden, err.Error())
			return
		}

		// Connect to Core for metadata
		client, conn, ctx, err := getCoreClient()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to connect to core: "+err.Error())
			return
		}
		defer conn.Close()

		profile, err := client.GetProfile(ctx, &pb.GetProfileRequest{})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to get user profile: "+err.Error())
			return
		}

		messageID := uuid.NewString()
		nowStr := time.Now().Format(time.RFC3339)
		var savedPath string

		if len(attachBytes) > 0 {
			// Save the file on host disk in the conversation folder
			convDir := filepath.Join("/data/shared/storage/Chat", conv.ID)
			if err := os.MkdirAll(convDir, 0755); err != nil {
				writeError(w, http.StatusInternalServerError, "failed to create chat storage directory: "+err.Error())
				return
			}

			savedPath = filepath.Join(convDir, attachName)
			if err := os.WriteFile(savedPath, attachBytes, 0644); err != nil {
				writeError(w, http.StatusInternalServerError, "failed to save attachment: "+err.Error())
				return
			}
		}

		// Insert into local DB
		msg := &Message{
			ID:             messageID,
			ConversationID: conv.ID,
			SenderID:       profile.UserId,
			SenderName:     profile.Username,
			Text:           text,
			AttachmentName: attachName,
			AttachmentType: attachType,
			AttachmentPath: savedPath,
			SentAt:         nowStr,
			IsRead:         true, // Sent messages are read by the sender
		}

		if err := insertMessage(msg); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to save message locally: "+err.Error())
			return
		}

		// Trigger active event source clients reload
		notifyClients()

		// Send message in background to all other participants
		go broadcastMessage(conv, msg, attachBytes, profile.UserId, profile.Username)

		writeJSON(w, http.StatusCreated, msg)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// GET /api/conversations/{id}/attachments
func handleAttachments(w http.ResponseWriter, r *http.Request, conv *Conversation) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	before := r.URL.Query().Get("before")
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	msgs, err := getSharedMedia(conv.ID, before, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get shared media: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, msgs)
}

// Background routine to send the P2P messages
func broadcastMessage(conv *Conversation, msg *Message, attachmentBytes []byte, selfUserID, selfUsername string) {
	client, conn, ctx, err := getCoreClient()
	if err != nil {
		log.Printf("P2P Broadcast error: failed to get core client: %v", err)
		return
	}
	defer conn.Close()

	// Get all contacts from core to resolve participant target User ID -> Contact ID
	contactsResp, err := client.ListContacts(ctx, &pb.ListContactsRequest{})
	if err != nil {
		log.Printf("P2P Broadcast error: failed to fetch contact list: %v", err)
		return
	}

	// Prepare participants list for payload
	partsPayload := []ParticipantInfo{}
	for _, p := range conv.Participants {
		partsPayload = append(partsPayload, ParticipantInfo{
			UserID:      p.UserID,
			DisplayName: p.DisplayName,
			InviteLink:  p.InviteLink,
		})
	}

	// Populate our own invite link for the payload
	inviteResp, inviteErr := client.GetInviteLink(ctx, &pb.GetInviteLinkRequest{})
	if inviteErr == nil && inviteResp.InviteLink != "" {
		for i := range partsPayload {
			if partsPayload[i].UserID == selfUserID {
				partsPayload[i].InviteLink = inviteResp.InviteLink
				break
			}
		}
	}

	// Build the JSON payload to send over libp2p
	payload := MessagePayload{
		ConversationID:   conv.ID,
		ConversationName: conv.Name,
		Participants:     partsPayload,
		MessageID:        msg.ID,
		SenderID:         selfUserID,
		SenderName:       selfUsername,
		Text:             msg.Text,
		AttachmentName:   msg.AttachmentName,
		AttachmentType:   msg.AttachmentType,
		AttachmentData:   attachmentBytes,
		SentAt:           msg.SentAt,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("P2P Broadcast error: failed to marshal payload: %v", err)
		return
	}

	// Send to every other participant
	for _, p := range conv.Participants {
		if p.UserID == selfUserID {
			continue // Do not send to ourselves
		}

		// Find contact by target_user_id
		var contactID string
		for _, c := range contactsResp.Contacts {
			if c.TargetUserId == p.UserID {
				contactID = c.Id
				break
			}
		}

		if contactID == "" {
			log.Printf("P2P Broadcast warning: could not find contact ID for target user %s (%s)", p.DisplayName, p.UserID)
			continue
		}

		log.Printf("P2P Broadcast: sending message %s to %s (%s) for contact ID %s", msg.ID, p.DisplayName, p.UserID, contactID)
		sendResp, sendErr := client.SendToContact(ctx, &pb.SendToContactRequest{
			ContactId: contactID,
			AppId:     appID,
			Payload:   payloadBytes,
		})

		if sendErr != nil {
			log.Printf("P2P Broadcast error: send failed to %s: %v", p.DisplayName, sendErr)
		} else if !sendResp.Success {
			log.Printf("P2P Broadcast error: send rejected by %s: %s", p.DisplayName, sendResp.StatusMessage)
		} else {
			log.Printf("P2P Broadcast: successfully sent message to %s", p.DisplayName)
		}
	}
}

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

// POST /api/contacts/add
func handleAddContact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		InviteLink  string `json:"invite_link"`
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.InviteLink == "" || req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "invite_link and display_name are required")
		return
	}

	if err := env.EnsureScopes([]string{"contacts:write"}, []string{"Send contact requests directly to other users"}); err != nil {
		if sdk.WriteConsentError(w, err) {
			return
		}
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	client, conn, ctx, err := getCoreClient()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to connect to core: "+err.Error())
		return
	}
	defer conn.Close()

	resp, err := client.SendContactRequest(ctx, &pb.SendContactRequestRequest{
		InviteLink:  req.InviteLink,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to send contact request: "+err.Error())
		return
	}

	if !resp.Success {
		writeError(w, http.StatusBadRequest, resp.StatusMessage)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": resp.StatusMessage,
	})
}
