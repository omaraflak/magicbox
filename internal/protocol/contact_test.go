package protocol

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/magicbox/core/internal/p2p"
)

func TestContactRequestHandler_StoresIncomingRequest(t *testing.T) {
	database, logger := setupTest(t)
	database.CreateUser("user-1", "alice", "hash", false)

	handler := newContactRequestHandler(database, logger)

	payload, _ := json.Marshal(ContactRequestPayload{
		DisplayName: "Bob",
		Multiaddr:   "/ip4/1.1.1.1/tcp/4001/p2p/peer-bob",
		EncPubKey:   "bob-enc-pub",
		UserID:      "bob-uid",
	})

	msg := &p2p.Message{
		AppID:        AppIDContactRequest,
		TargetUserID: "user-1",
		Payload:      payload,
	}

	if err := handler(context.Background(), "peer-bob", msg); err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	reqs, err := database.GetContactRequests("user-1", "incoming")
	if err != nil {
		t.Fatalf("GetContactRequests failed: %v", err)
	}
	if len(reqs) != 1 {
		t.Fatalf("expected 1 incoming request, got %d", len(reqs))
	}
	if reqs[0].DisplayName != "Bob" {
		t.Errorf("expected display_name Bob, got %s", reqs[0].DisplayName)
	}
	if reqs[0].PeerID != "peer-bob" {
		t.Errorf("expected peer_id peer-bob, got %s", reqs[0].PeerID)
	}
}

func TestContactAcceptHandler_CreatesContact(t *testing.T) {
	database, logger := setupTest(t)
	database.CreateUser("user-1", "alice", "hash", false)

	// Simulate an outgoing request that Alice sent to Bob.
	database.InsertContactRequest(
		"req-1", "user-1", "outgoing", "Bob",
		"peer-bob", "/ip4/1.1.1.1/tcp/4001/p2p/peer-bob",
		"bob-uid", "bob-enc-pub",
	)

	handler := newContactAcceptHandler(database, logger)

	payload, _ := json.Marshal(ContactAcceptPayload{
		DisplayName: "Bob",
		Multiaddr:   "/ip4/1.1.1.1/tcp/4001/p2p/peer-bob",
		EncPubKey:   "bob-enc-pub",
		UserID:      "bob-uid",
	})

	msg := &p2p.Message{
		AppID:        AppIDContactAccept,
		TargetUserID: "user-1",
		Payload:      payload,
	}

	if err := handler(context.Background(), "peer-bob", msg); err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	// Verify contact was created.
	contacts, err := database.GetContacts("user-1")
	if err != nil {
		t.Fatalf("GetContacts failed: %v", err)
	}
	if len(contacts) != 1 {
		t.Fatalf("expected 1 contact, got %d", len(contacts))
	}
	if contacts[0].DisplayName != "Bob" {
		t.Errorf("expected display_name Bob, got %s", contacts[0].DisplayName)
	}

	// Verify outgoing request was deleted.
	reqs, _ := database.GetContactRequests("user-1", "outgoing")
	if len(reqs) != 0 {
		t.Errorf("expected 0 outgoing requests after accept, got %d", len(reqs))
	}
}

func TestContactAcceptHandler_NoMatchingRequest(t *testing.T) {
	database, logger := setupTest(t)
	database.CreateUser("user-1", "alice", "hash", false)

	handler := newContactAcceptHandler(database, logger)

	payload, _ := json.Marshal(ContactAcceptPayload{
		DisplayName: "Bob",
		Multiaddr:   "/ip4/1.1.1.1/tcp/4001/p2p/peer-bob",
		EncPubKey:   "bob-enc-pub",
		UserID:      "bob-uid",
	})

	msg := &p2p.Message{
		AppID:        AppIDContactAccept,
		TargetUserID: "user-1",
		Payload:      payload,
	}

	// Should not error — just logs a warning and returns nil.
	if err := handler(context.Background(), "peer-unknown", msg); err != nil {
		t.Fatalf("expected no error for unknown peer, got %v", err)
	}
}
