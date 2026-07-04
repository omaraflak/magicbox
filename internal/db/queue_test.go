package db

import (
	"testing"
	"time"
)

func TestEnqueueMessage_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()

	db.CreateUser("user-1", "alice", "hash", false)
	db.AddContact("contact-1", "user-1", "Bob", "peer-123", "/ip4/127.0.0.1/tcp/4001/p2p/peer-123", "bob-uid", "enc-pub-hex")

	err := db.EnqueueMessage("msg-1", "contact-1", "system:key-update", []byte("new-key-hex"), 5)
	if err != nil {
		t.Fatalf("EnqueueMessage failed: %v", err)
	}
}

func TestGetPendingMessages_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()

	db.CreateUser("user-1", "alice", "hash", false)
	db.AddContact("contact-1", "user-1", "Bob", "peer-123", "/ip4/127.0.0.1/tcp/4001/p2p/peer-123", "bob-uid", "enc-pub-hex")

	err := db.EnqueueMessage("msg-1", "contact-1", "system:key-update", []byte("new-key-hex"), 5)
	if err != nil {
		t.Fatalf("EnqueueMessage failed: %v", err)
	}

	msgs, err := db.GetPendingMessages()
	if err != nil {
		t.Fatalf("GetPendingMessages failed: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 pending message, got %d", len(msgs))
	}

	if msgs[0].ID != "msg-1" {
		t.Errorf("expected id msg-1, got %s", msgs[0].ID)
	}
}

func TestDeleteMessage_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()
	db.CreateUser("user-1", "alice", "hash", false)
	db.AddContact("contact-1", "user-1", "Bob", "peer-123", "/ip4/127.0.0.1/tcp/4001/p2p/peer-123", "bob-uid", "enc-pub-hex")
	db.EnqueueMessage("msg-1", "contact-1", "system:key-update", []byte("payload"), 5)

	if err := db.DeleteMessage("msg-1"); err != nil {
		t.Fatalf("DeleteMessage failed: %v", err)
	}
}

func TestDeleteMessage_RemovesFromPending(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()
	db.CreateUser("user-1", "alice", "hash", false)
	db.AddContact("contact-1", "user-1", "Bob", "peer-123", "/ip4/127.0.0.1/tcp/4001/p2p/peer-123", "bob-uid", "enc-pub-hex")
	db.EnqueueMessage("msg-1", "contact-1", "system:key-update", []byte("payload"), 5)

	if err := db.DeleteMessage("msg-1"); err != nil {
		t.Fatalf("DeleteMessage failed: %v", err)
	}

	msgs, err := db.GetPendingMessages()
	if err != nil {
		t.Fatalf("GetPendingMessages failed: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages after delete, got %d", len(msgs))
	}
}

func TestIncrementMessageAttempts_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()
	db.CreateUser("user-1", "alice", "hash", false)
	db.AddContact("contact-1", "user-1", "Bob", "peer-123", "/ip4/127.0.0.1/tcp/4001/p2p/peer-123", "bob-uid", "enc-pub-hex")
	db.EnqueueMessage("msg-1", "contact-1", "system:key-update", []byte("payload"), 5)

	futureTime := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	if err := db.IncrementMessageAttempts("msg-1", futureTime); err != nil {
		t.Fatalf("IncrementMessageAttempts failed: %v", err)
	}
}

func TestIncrementMessageAttempts_SchedulesFuture(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()
	db.CreateUser("user-1", "alice", "hash", false)
	db.AddContact("contact-1", "user-1", "Bob", "peer-123", "/ip4/127.0.0.1/tcp/4001/p2p/peer-123", "bob-uid", "enc-pub-hex")
	db.EnqueueMessage("msg-1", "contact-1", "system:key-update", []byte("payload"), 5)

	futureTime := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	if err := db.IncrementMessageAttempts("msg-1", futureTime); err != nil {
		t.Fatalf("IncrementMessageAttempts failed: %v", err)
	}

	msgs, err := db.GetPendingMessages()
	if err != nil {
		t.Fatalf("GetPendingMessages failed: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("expected 0 pending messages (future retry), got %d", len(msgs))
	}
}

func TestCleanExpiredMessages_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()

	_, err := db.CleanExpiredMessages()
	if err != nil {
		t.Fatalf("CleanExpiredMessages failed: %v", err)
	}
}

func TestCleanExpiredMessages_RemovesExpired(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()
	db.CreateUser("user-1", "alice", "hash", false)
	db.AddContact("contact-1", "user-1", "Bob", "peer-123", "/ip4/127.0.0.1/tcp/4001/p2p/peer-123", "bob-uid", "enc-pub-hex")
	db.EnqueueMessage("msg-1", "contact-1", "system:key-update", []byte("payload"), 1)

	db.IncrementMessageAttempts("msg-1", time.Now().UTC().Format(time.RFC3339))

	deleted, err := db.CleanExpiredMessages()
	if err != nil {
		t.Fatalf("CleanExpiredMessages failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}
}

func TestGetPendingMessages_DeletedContactExcluded(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()
	db.CreateUser("user-1", "alice", "hash", false)
	db.AddContact("contact-1", "user-1", "Bob", "peer-123", "/ip4/127.0.0.1/tcp/4001/p2p/peer-123", "bob-uid", "enc-pub-hex")
	db.EnqueueMessage("msg-1", "contact-1", "system:key-update", []byte("payload"), 5)

	db.DeleteContact("contact-1", "user-1")

	msgs, _ := db.GetPendingMessages()
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages after contact deleted, got %d", len(msgs))
	}
}

func TestGetPendingMessages_ContactRequests(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()

	db.CreateUser("user-1", "alice", "hash", false)
	db.InsertContactRequest("req-1", "user-1", "outgoing", "Bob", "peer-bob", "/ip4/127.0.0.1/tcp/4001/p2p/peer-bob", "bob-uid", "enc-bob")
	db.EnqueueMessage("msg-1", "req-1", "system:contact-request", []byte("payload"), 5)

	msgs, err := db.GetPendingMessages()
	if err != nil {
		t.Fatalf("GetPendingMessages failed: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message targeting request, got %d", len(msgs))
	}
}
