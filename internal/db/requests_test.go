package db

import (
	"testing"
)

func TestInsertContactRequest_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()
	db.CreateUser("user-1", "alice", "hash", false)

	err := db.InsertContactRequest("req-1", "user-1", "incoming", "Bob", "peer-bob", "/ip4/1.1.1.1/tcp/4001/p2p/peer-bob", "bob-uid", "enc-bob")
	if err != nil {
		t.Fatalf("InsertContactRequest failed: %v", err)
	}
}

func TestGetContactRequests_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()
	db.CreateUser("user-1", "alice", "hash", false)

	err := db.InsertContactRequest("req-1", "user-1", "incoming", "Bob", "peer-bob", "/ip4/1.1.1.1/tcp/4001/p2p/peer-bob", "bob-uid", "enc-bob")
	if err != nil {
		t.Fatalf("InsertContactRequest failed: %v", err)
	}

	reqs, err := db.GetContactRequests("user-1", "incoming")
	if err != nil {
		t.Fatalf("GetContactRequests failed: %v", err)
	}
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	if reqs[0].DisplayName != "Bob" {
		t.Errorf("expected display_name Bob, got %s", reqs[0].DisplayName)
	}
}

func TestGetContactRequest(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()
	db.CreateUser("user-1", "alice", "hash", false)
	db.InsertContactRequest("req-1", "user-1", "incoming", "Bob", "peer-bob", "/ip4/1.1.1.1/tcp/4001/p2p/peer-bob", "bob-uid", "enc-bob")

	req, err := db.GetContactRequest("user-1", "req-1")
	if err != nil {
		t.Fatalf("GetContactRequest failed: %v", err)
	}
	if req == nil {
		t.Fatal("expected request, got nil")
	}
}

func TestGetContactRequestByPeerID(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()
	db.CreateUser("user-1", "alice", "hash", false)
	db.InsertContactRequest("req-1", "user-1", "outgoing", "Bob", "peer-bob", "/ip4/1.1.1.1/tcp/4001/p2p/peer-bob", "bob-uid", "enc-bob")

	req, err := db.GetContactRequestByPeerID("user-1", "peer-bob")
	if err != nil {
		t.Fatalf("GetContactRequestByPeerID failed: %v", err)
	}
	if req == nil {
		t.Fatal("expected request, got nil")
	}
}

func TestDeleteContactRequest_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()
	db.CreateUser("user-1", "alice", "hash", false)
	db.InsertContactRequest("req-1", "user-1", "incoming", "Bob", "peer-bob", "/ip4/1.1.1.1/tcp/4001/p2p/peer-bob", "bob-uid", "enc-bob")

	if err := db.DeleteContactRequest("user-1", "req-1"); err != nil {
		t.Fatalf("DeleteContactRequest failed: %v", err)
	}
}

func TestDeleteContactRequest_RemovesRequest(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()
	db.CreateUser("user-1", "alice", "hash", false)
	db.InsertContactRequest("req-1", "user-1", "incoming", "Bob", "peer-bob", "/ip4/1.1.1.1/tcp/4001/p2p/peer-bob", "bob-uid", "enc-bob")

	if err := db.DeleteContactRequest("user-1", "req-1"); err != nil {
		t.Fatalf("DeleteContactRequest failed: %v", err)
	}

	reqs, err := db.GetContactRequests("user-1", "")
	if err != nil {
		t.Fatalf("GetContactRequests failed: %v", err)
	}
	if len(reqs) != 0 {
		t.Errorf("expected 0 requests after delete, got %d", len(reqs))
	}
}
