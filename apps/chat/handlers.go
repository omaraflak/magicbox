package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	pb "github.com/magicbox/core/api/proto/v1"
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
	IsSystem         bool              `json:"is_system,omitempty"`
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

// Background routine to send the P2P messages
func broadcastMessage(conv *Conversation, msg *Message, attachmentBytes []byte, selfUserID, selfUsername string) {
	for _, p := range conv.Participants {
		if p.UserID == selfUserID {
			continue // Do not send to ourselves
		}
		if err := insertMessageDelivery(msg.ID, p.UserID, "pending"); err != nil {
			log.Printf("Warning: failed to insert message delivery entry: %v", err)
		}
	}
	go processDeliveryQueue()
}

var processingQueueMu sync.Mutex

func processDeliveryQueue() {
	processingQueueMu.Lock()
	defer processingQueueMu.Unlock()

	deliveries, err := getPendingDeliveries()
	if err != nil {
		log.Printf("Queue processing error: failed to query pending deliveries: %v", err)
		return
	}

	if len(deliveries) == 0 {
		return
	}

	client, conn, ctx, err := getCoreClient()
	if err != nil {
		log.Printf("Queue processing error: failed to get core client: %v", err)
		return
	}
	defer conn.Close()

	// Get contact list to resolve user ID to contact ID
	contactsResp, err := client.ListContacts(ctx, &pb.ListContactsRequest{})
	if err != nil {
		log.Printf("Queue processing error: failed to fetch contact list: %v", err)
		return
	}

	// Get our own profile details to populate invite link if needed
	profile, err := client.GetProfile(ctx, &pb.GetProfileRequest{})
	if err != nil {
		log.Printf("Queue processing error: failed to fetch profile: %v", err)
		return
	}

	inviteResp, inviteErr := client.GetInviteLink(ctx, &pb.GetInviteLinkRequest{})
	var ownInviteLink string
	if inviteErr == nil && inviteResp != nil {
		ownInviteLink = inviteResp.InviteLink
	}

	// Cache conversation payloads to avoid redundant queries / marshalling in this batch
	convPayloadCache := make(map[string]*MessagePayload)

	for _, d := range deliveries {
		// Find contact by target_user_id (d.RecipientID)
		var contactID string
		for _, c := range contactsResp.Contacts {
			if c.TargetUserId == d.RecipientID {
				contactID = c.Id
				break
			}
		}

		if contactID == "" {
			log.Printf("Queue processing warning: could not find contact ID for target user %s", d.RecipientID)
			// If recipient is no longer a contact, we should drop it or mark it as failed
			_ = updateMessageDeliveryStatus(d.MessageID, d.RecipientID, "failed")
			continue
		}

		// Reconstruct/fetch conversation payload
		payload, exists := convPayloadCache[d.MessageID]
		if !exists {
			conv, err := getConversation(d.ConversationID)
			if err != nil || conv == nil {
				log.Printf("Queue processing error: failed to load conversation %s for message %s: %v", d.ConversationID, d.MessageID, err)
				continue
			}

			// Load attachment bytes if applicable
			var attachBytes []byte
			if d.AttachmentPath != "" && d.AttachmentName != "" {
				attachBytes, err = os.ReadFile(d.AttachmentPath)
				if err != nil {
					log.Printf("Queue processing warning: failed to read attachment from path %s: %v", d.AttachmentPath, err)
				}
			}

			// Prepare participants payload
			partsPayload := []ParticipantInfo{}
			for _, p := range conv.Participants {
				inviteLnk := p.InviteLink
				if p.UserID == profile.UserId {
					inviteLnk = ownInviteLink
				}
				partsPayload = append(partsPayload, ParticipantInfo{
					UserID:      p.UserID,
					DisplayName: p.DisplayName,
					InviteLink:  inviteLnk,
				})
			}

			payload = &MessagePayload{
				ConversationID:   conv.ID,
				ConversationName: conv.Name,
				Participants:     partsPayload,
				MessageID:        d.MessageID,
				SenderID:         d.SenderID,
				SenderName:       d.SenderName,
				Text:             d.Text,
				AttachmentName:   d.AttachmentName,
				AttachmentType:   d.AttachmentType,
				AttachmentData:   attachBytes,
				SentAt:           d.SentAt,
				IsSystem:         d.IsSystem,
			}
			convPayloadCache[d.MessageID] = payload
		}

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			log.Printf("Queue processing error: failed to marshal payload: %v", err)
			continue
		}

		log.Printf("Queue processing: attempting delivery of message %s to recipient %s (contact ID %s) (attempt %d)", d.MessageID, d.RecipientID, contactID, d.Attempts+1)
		sendResp, sendErr := client.SendToContact(ctx, &pb.SendToContactRequest{
			ContactId: contactID,
			AppId:     appID,
			Payload:   payloadBytes,
		})

		if sendErr != nil {
			log.Printf("Queue processing error: send failed: %v", sendErr)
			_ = updateMessageDeliveryStatus(d.MessageID, d.RecipientID, "failed")
		} else if !sendResp.Success {
			log.Printf("Queue processing error: send rejected: %s", sendResp.StatusMessage)
			_ = updateMessageDeliveryStatus(d.MessageID, d.RecipientID, "failed")
		} else {
			log.Printf("Queue processing: successfully delivered message %s to recipient %s", d.MessageID, d.RecipientID)
			// Delivery succeeded, remove from queue
			_ = deleteMessageDelivery(d.MessageID, d.RecipientID)
		}
	}
}
