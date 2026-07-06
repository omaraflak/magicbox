package protocol

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/magicbox/core/internal/db"
	"github.com/magicbox/core/internal/logging"
	"github.com/magicbox/core/internal/p2p"
)

// AppCheckRequestPayload is sent in the system:app-check message.
type AppCheckRequestPayload struct {
	RequestID    string `json:"request_id"`
	SenderUserID string `json:"sender_user_id"`
	AppID        string `json:"app_id"`
}

// AppCheckResponsePayload is sent in the system:app-check-response message.
type AppCheckResponsePayload struct {
	RequestID string `json:"request_id"`
	AppID     string `json:"app_id"`
	Installed bool   `json:"installed"`
}

var (
	pendingRequestsMu sync.Mutex
	pendingRequests   = make(map[string]chan bool)
)

// RegisterAppCheckRequest registers a request ID and returns a channel to wait on.
func RegisterAppCheckRequest(requestID string) chan bool {
	pendingRequestsMu.Lock()
	defer pendingRequestsMu.Unlock()
	ch := make(chan bool, 1)
	pendingRequests[requestID] = ch
	return ch
}

// DeregisterAppCheckRequest removes a request ID.
func DeregisterAppCheckRequest(requestID string) {
	pendingRequestsMu.Lock()
	defer pendingRequestsMu.Unlock()
	delete(pendingRequests, requestID)
}

// ResolveAppCheckRequest resolves a request ID with the result.
func ResolveAppCheckRequest(requestID string, installed bool) {
	pendingRequestsMu.Lock()
	defer pendingRequestsMu.Unlock()
	if ch, ok := pendingRequests[requestID]; ok {
		ch <- installed
		close(ch)
		delete(pendingRequests, requestID)
	}
}

// newAppCheckHandler returns a handler for the system:app-check protocol.
func newAppCheckHandler(database *db.DB, service p2p.Service, logger *logging.Logger) p2p.Handler {
	return func(ctx context.Context, fromPeerID string, msg *p2p.Message) error {
		var req AppCheckRequestPayload
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			logger.Error("Failed to unmarshal app-check payload",
				logging.F("from_peer", fromPeerID),
				logging.F("error", err.Error()))
			return err
		}

		app, err := database.GetAppByAppIDAndUserID(req.AppID, msg.TargetUserID)
		installed := (err == nil && app != nil)
		if err != nil {
			logger.Error("database query error in appcheck handler", logging.F("error", err.Error()))
		}

		contact, err := database.GetContactByTargetUserID(msg.TargetUserID, req.SenderUserID)
		if err != nil {
			logger.Error("Failed to look up contact for app-check response",
				logging.F("user_id", msg.TargetUserID),
				logging.F("sender_user_id", req.SenderUserID),
				logging.F("error", err.Error()))
			return err
		}
		if contact == nil {
			logger.Warn("App-check from non-contact or contact not found",
				logging.F("user_id", msg.TargetUserID),
				logging.F("sender_user_id", req.SenderUserID))
			return nil
		}

		respPayload := AppCheckResponsePayload{
			RequestID: req.RequestID,
			AppID:     req.AppID,
			Installed: installed,
		}
		payloadBytes, err := json.Marshal(respPayload)
		if err != nil {
			return err
		}

		respMsg := &p2p.Message{
			AppID:        AppIDAppCheckResponse,
			TargetUserID: req.SenderUserID,
			Payload:      payloadBytes,
		}

		go func() {
			sendCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := service.SendTo(sendCtx, contact.Multiaddr, contact.EncPubKey, respMsg); err != nil {
				logger.Error("Failed to send appcheck response to peer",
					logging.F("peer", fromPeerID),
					logging.F("error", err.Error()))
			}
		}()

		return nil
	}
}

// newAppCheckResponseHandler returns a handler for the system:app-check-response protocol.
func newAppCheckResponseHandler(logger *logging.Logger) p2p.Handler {
	return func(ctx context.Context, fromPeerID string, msg *p2p.Message) error {
		var resp AppCheckResponsePayload
		if err := json.Unmarshal(msg.Payload, &resp); err != nil {
			logger.Error("Failed to unmarshal appcheck response payload",
				logging.F("from_peer", fromPeerID),
				logging.F("error", err.Error()))
			return err
		}

		ResolveAppCheckRequest(resp.RequestID, resp.Installed)
		return nil
	}
}
