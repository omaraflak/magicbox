package protocol

import (
	"testing"
	"time"
)

func TestNextRetryDelay(t *testing.T) {
	tests := []struct {
		attempts int
		want     time.Duration
	}{
		{0, 1 * time.Minute},
		{1, 2 * time.Minute},
		{2, 4 * time.Minute},
		{3, 8 * time.Minute},
		{10, 1024 * time.Minute},
		{20, 24 * time.Hour}, // capped
		{100, 24 * time.Hour}, // capped
	}

	for _, tt := range tests {
		got := nextRetryDelay(tt.attempts)
		if got != tt.want {
			t.Errorf("nextRetryDelay(%d) = %v, want %v", tt.attempts, got, tt.want)
		}
	}
}

func TestEnqueueForContacts(t *testing.T) {
	database, _ := setupTest(t)

	database.CreateUser("user-1", "alice", "hash", false)
	database.AddContact("c-1", "user-1", "Bob", "peer-1", "/ip4/1.1.1.1/tcp/4001/p2p/peer-1", "bob-uid", "enc1", "master1")
	database.AddContact("c-2", "user-1", "Carol", "peer-2", "/ip4/2.2.2.2/tcp/4001/p2p/peer-2", "carol-uid", "enc2", "master2")

	contacts, err := database.GetContacts("user-1")
	if err != nil {
		t.Fatalf("GetContacts failed: %v", err)
	}

	if err := EnqueueForContacts(database, contacts, AppIDKeyUpdate, []byte("new-key")); err != nil {
		t.Fatalf("EnqueueForContacts failed: %v", err)
	}

	msgs, err := database.GetPendingMessages()
	if err != nil {
		t.Fatalf("GetPendingMessages failed: %v", err)
	}
	if len(msgs) != 2 {
		t.Errorf("expected 2 queued messages, got %d", len(msgs))
	}
}
