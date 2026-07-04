package db

import (
	"testing"
)

func TestAddContact_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()

	userID := "user-123"
	if err := db.CreateUser(userID, "omar", "hash", false); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	contactID := "contact-456"
	contactName := "Alice"
	peerID := "12D3KooWTestPeerID"
	multiaddr := "/ip4/127.0.0.1/tcp/4001/p2p/12D3KooWTestPeerID"
	targetUserID := "alice-id"

	if err := db.AddContact(contactID, userID, contactName, peerID, multiaddr, targetUserID, "test-enc-pub-key", "test-master-pub-key"); err != nil {
		t.Fatalf("AddContact failed: %v", err)
	}
}

func TestGetContacts_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()

	userID := "user-123"
	if err := db.CreateUser(userID, "omar", "hash", false); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	contactID := "contact-456"
	contactName := "Alice"
	peerID := "12D3KooWTestPeerID"
	multiaddr := "/ip4/127.0.0.1/tcp/4001/p2p/12D3KooWTestPeerID"
	targetUserID := "alice-id"

	if err := db.AddContact(contactID, userID, contactName, peerID, multiaddr, targetUserID, "test-enc-pub-key", "test-master-pub-key"); err != nil {
		t.Fatalf("AddContact failed: %v", err)
	}

	contacts, err := db.GetContacts(userID)
	if err != nil {
		t.Fatalf("GetContacts failed: %v", err)
	}
	if len(contacts) != 1 {
		t.Fatalf("expected 1 contact, got %d", len(contacts))
	}
	if contacts[0].DisplayName != contactName || contacts[0].TargetUserID != targetUserID {
		t.Errorf("unexpected contact details: %+v", contacts[0])
	}
	if contacts[0].MasterPubKey != "test-master-pub-key" {
		t.Errorf("expected MasterPubKey %q, got %q", "test-master-pub-key", contacts[0].MasterPubKey)
	}
}

func TestGetContactByPeerID_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()

	userID := "user-123"
	if err := db.CreateUser(userID, "omar", "hash", false); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	peerID := "12D3KooWTestPeerID"
	if err := db.AddContact("contact-1", userID, "Alice", peerID, "/ip4/127.0.0.1/tcp/4001/p2p/12D3KooWTestPeerID", "alice-id", "enc-key", "master-key"); err != nil {
		t.Fatalf("AddContact failed: %v", err)
	}

	contact, err := db.GetContactByPeerID(userID, peerID)
	if err != nil {
		t.Fatalf("GetContactByPeerID failed: %v", err)
	}
	if contact == nil {
		t.Fatal("expected contact to be found, got nil")
	}
	if contact.PeerID != peerID {
		t.Errorf("expected PeerID %q, got %q", peerID, contact.PeerID)
	}
}

func TestGetContactByPeerID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()

	userID := "user-123"
	if err := db.CreateUser(userID, "omar", "hash", false); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	contact, err := db.GetContactByPeerID(userID, "nonexistent-peer")
	if err != nil {
		t.Fatalf("GetContactByPeerID failed: %v", err)
	}
	if contact != nil {
		t.Errorf("expected nil contact, got %+v", contact)
	}
}

func TestUpdateContactIdentity(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()
	db.CreateUser("user-1", "alice", "hash", false)
	db.AddContact("c-1", "user-1", "Bob", "peer-old", "/ip4/1.1.1.1/tcp/4001/p2p/peer-old", "bob-uid", "enc-old", "master-old")

	err := db.UpdateContactIdentity("c-1", "peer-new", "/ip4/1.1.1.1/tcp/4001/p2p/peer-new", "enc-new")
	if err != nil {
		t.Fatalf("UpdateContactIdentity failed: %v", err)
	}

	contact, err := db.GetContactByPeerID("user-1", "peer-new")
	if err != nil {
		t.Fatalf("GetContactByPeerID failed: %v", err)
	}
	if contact == nil {
		t.Fatal("expected to find contact by new peer ID")
	}
	if contact.EncPubKey != "enc-new" {
		t.Errorf("expected enc_pub_key enc-new, got %s", contact.EncPubKey)
	}
}

func TestGetAllContacts(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()

	if err := db.CreateUser("u1", "alice", "hash", false); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if err := db.CreateUser("u2", "bob", "hash", false); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	_ = db.AddContact("c1", "u1", "Charlie", "peer1", "/ip4/1.1.1.1/tcp/4001/p2p/peer1", "char-uid", "enc1", "master1")
	_ = db.AddContact("c2", "u2", "Dave", "peer2", "/ip4/1.1.1.2/tcp/4001/p2p/peer2", "dave-uid", "enc2", "master2")

	contacts, err := db.GetAllContacts()
	if err != nil {
		t.Fatalf("GetAllContacts failed: %v", err)
	}
	if len(contacts) != 2 {
		t.Fatalf("expected 2 contacts, got %d", len(contacts))
	}

	// Verify order by display_name
	if contacts[0].DisplayName != "Charlie" || contacts[1].DisplayName != "Dave" {
		t.Errorf("unexpected sorting: %s, %s", contacts[0].DisplayName, contacts[1].DisplayName)
	}
}

func TestContactStatusAndQueries(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()

	userID := "user-123"
	if err := db.CreateUser(userID, "omar", "hash", false); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	contactID := "contact-456"
	contactName := "Alice"
	peerID := "peer-alice"
	multiaddr := "/ip4/127.0.0.1/tcp/4001/p2p/peer-alice"
	targetUserID := "alice-id"

	if err := db.AddContact(contactID, userID, contactName, peerID, multiaddr, targetUserID, "enc-key", "master-key"); err != nil {
		t.Fatalf("AddContact failed: %v", err)
	}

	// 1. Verify default status is 'active'
	c, err := db.GetContactByID(contactID, userID)
	if err != nil {
		t.Fatalf("GetContactByID failed: %v", err)
	}
	if c.Status != "active" {
		t.Errorf("expected default status 'active', got %q", c.Status)
	}

	// 2. GetContactByTargetUserID
	c2, err := db.GetContactByTargetUserID(userID, targetUserID)
	if err != nil {
		t.Fatalf("GetContactByTargetUserID failed: %v", err)
	}
	if c2 == nil {
		t.Fatal("expected contact to be found by target user ID, got nil")
	}
	if c2.ID != contactID {
		t.Errorf("expected contact ID %q, got %q", contactID, c2.ID)
	}

	// 3. UpdateContactStatus to 'revoked'
	if err := db.UpdateContactStatus(contactID, "revoked"); err != nil {
		t.Fatalf("UpdateContactStatus failed: %v", err)
	}

	c3, err := db.GetContactByID(contactID, userID)
	if err != nil {
		t.Fatalf("GetContactByID failed: %v", err)
	}
	if c3.Status != "revoked" {
		t.Errorf("expected status 'revoked', got %q", c3.Status)
	}

	// 4. UpdateContactFromRequest
	err = db.UpdateContactFromRequest(contactID, "peer-new", "/ip4/1.1.1.1/p2p/peer-new", "enc-new", "master-new")
	if err != nil {
		t.Fatalf("UpdateContactFromRequest failed: %v", err)
	}

	c4, err := db.GetContactByID(contactID, userID)
	if err != nil {
		t.Fatalf("GetContactByID failed: %v", err)
	}
	if c4.PeerID != "peer-new" || c4.Multiaddr != "/ip4/1.1.1.1/p2p/peer-new" || c4.EncPubKey != "enc-new" || c4.MasterPubKey != "master-new" {
		t.Errorf("unexpected updated fields: %+v", c4)
	}
	if c4.Status != "active" {
		t.Errorf("expected status reset to 'active', got %q", c4.Status)
	}
}

