package main

import (
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
			IsSystem:       true,
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
