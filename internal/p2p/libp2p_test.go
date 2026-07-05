package p2p

import (
	"context"
	"encoding/hex"
	"sync"
	"testing"
	"time"

	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	internalcrypto "github.com/magicbox/core/internal/crypto"
	"github.com/magicbox/core/internal/logging"
)

func TestLibp2pServiceFlow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Generate local identities
	mnemonic1, _ := internalcrypto.GenerateMnemonic()
	edPriv1, err := internalcrypto.DeriveIdentityKey(mnemonic1, 0)
	if err != nil {
		t.Fatalf("failed to derive identity key 1: %v", err)
	}
	xPriv1, err := internalcrypto.DeriveEncryptionKey(mnemonic1, 0)
	if err != nil {
		t.Fatalf("failed to derive encryption key 1: %v", err)
	}
	p2pKey1, err := libp2pcrypto.UnmarshalEd25519PrivateKey(edPriv1)
	if err != nil {
		t.Fatalf("failed to convert key 1: %v", err)
	}

	mnemonic2, _ := internalcrypto.GenerateMnemonic()
	edPriv2, err := internalcrypto.DeriveIdentityKey(mnemonic2, 0)
	if err != nil {
		t.Fatalf("failed to derive identity key 2: %v", err)
	}
	xPriv2, err := internalcrypto.DeriveEncryptionKey(mnemonic2, 0)
	if err != nil {
		t.Fatalf("failed to derive encryption key 2: %v", err)
	}
	p2pKey2, err := libp2pcrypto.UnmarshalEd25519PrivateKey(edPriv2)
	if err != nil {
		t.Fatalf("failed to convert key 2: %v", err)
	}

	// 2. Instantiate services
	logger1, err := logging.New(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create logger 1: %v", err)
	}
	defer logger1.Close()
	logger2, err := logging.New(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create logger 2: %v", err)
	}
	defer logger2.Close()

	srv1 := NewLibp2pService(p2pKey1, xPriv1, []string{"/ip4/127.0.0.1/tcp/0"}, logger1)
	srv2 := NewLibp2pService(p2pKey2, xPriv2, []string{"/ip4/127.0.0.1/tcp/0"}, logger2)

	// Start Node 1
	if err := srv1.Start(ctx); err != nil {
		t.Fatalf("failed to start srv1: %v", err)
	}
	defer srv1.Stop()

	// Start Node 2
	if err := srv2.Start(ctx); err != nil {
		t.Fatalf("failed to start srv2: %v", err)
	}
	defer srv2.Stop()

	// 3. Register message handler on Node 2
	receivedPayload := make(chan []byte, 1)
	receivedSender := make(chan string, 1)

	srv2.RegisterHandler("com.magicbox.test", func(ctx context.Context, fromPeerID string, msg *Message) error {
		receivedSender <- fromPeerID
		receivedPayload <- msg.Payload
		return nil
	})

	// 4. Send message from Node 1 to Node 2
	addrs2 := srv2.Multiaddrs()
	if len(addrs2) == 0 {
		t.Fatalf("node 2 has no listening multiaddresses")
	}

	var rawAddr string
	for _, a := range addrs2 {
		rawAddr = a
		break
	}

	encPubKeyHex := hex.EncodeToString(xPriv2.PublicKey().Bytes())

	testMsg := &Message{
		AppID:        "com.magicbox.test",
		TargetUserID: "user-456",
		Payload:      []byte("Hello from peer 1!"),
	}

	err = srv1.SendTo(ctx, rawAddr, encPubKeyHex, testMsg)
	if err != nil {
		t.Fatalf("failed to send message: %v", err)
	}

	// 5. Assert message reception and decryption
	select {
	case sender := <-receivedSender:
		if sender != srv1.HostID() {
			t.Errorf("expected sender peer ID %q, got %q", srv1.HostID(), sender)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for sender peer ID verification")
	}

	select {
	case payload := <-receivedPayload:
		if string(payload) != "Hello from peer 1!" {
			t.Errorf("expected payload %q, got %q", "Hello from peer 1!", string(payload))
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for payload verification")
	}
}

func TestLibp2pServiceUnhandledProtocol(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mnemonic1, _ := internalcrypto.GenerateMnemonic()
	edPriv1, _ := internalcrypto.DeriveIdentityKey(mnemonic1, 0)
	xPriv1, _ := internalcrypto.DeriveEncryptionKey(mnemonic1, 0)
	p2pKey1, _ := libp2pcrypto.UnmarshalEd25519PrivateKey(edPriv1)

	mnemonic2, _ := internalcrypto.GenerateMnemonic()
	edPriv2, _ := internalcrypto.DeriveIdentityKey(mnemonic2, 0)
	xPriv2, _ := internalcrypto.DeriveEncryptionKey(mnemonic2, 0)
	p2pKey2, _ := libp2pcrypto.UnmarshalEd25519PrivateKey(edPriv2)

	logger1, _ := logging.New(t.TempDir())
	defer logger1.Close()
	logger2, _ := logging.New(t.TempDir())
	defer logger2.Close()

	srv1 := NewLibp2pService(p2pKey1, xPriv1, []string{"/ip4/127.0.0.1/tcp/0"}, logger1)
	srv2 := NewLibp2pService(p2pKey2, xPriv2, []string{"/ip4/127.0.0.1/tcp/0"}, logger2)

	_ = srv1.Start(ctx)
	defer srv1.Stop()
	_ = srv2.Start(ctx)
	defer srv2.Stop()

	addrs2 := srv2.Multiaddrs()

	encPubKeyHex := hex.EncodeToString(xPriv2.PublicKey().Bytes())

	testMsg := &Message{
		AppID:        "com.magicbox.unhandled",
		TargetUserID: "user-456",
		Payload:      []byte("Some data"),
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := srv1.SendTo(ctx, addrs2[0], encPubKeyHex, testMsg)
		if err != nil {
			t.Errorf("unexpected error on unhandled protocol send: %v", err)
		}
	}()
	wg.Wait()
}

func TestLibp2pServiceLoopback(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mnemonic, _ := internalcrypto.GenerateMnemonic()
	edPriv, _ := internalcrypto.DeriveIdentityKey(mnemonic, 0)
	xPriv, _ := internalcrypto.DeriveEncryptionKey(mnemonic, 0)
	p2pKey, _ := libp2pcrypto.UnmarshalEd25519PrivateKey(edPriv)

	logger, _ := logging.New(t.TempDir())
	defer logger.Close()

	srv := NewLibp2pService(p2pKey, xPriv, []string{"/ip4/127.0.0.1/tcp/0"}, logger)
	if err := srv.Start(ctx); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer srv.Stop()

	receivedPayload := make(chan []byte, 1)
	receivedSender := make(chan string, 1)

	srv.RegisterHandler("com.magicbox.loopback", func(ctx context.Context, fromPeerID string, msg *Message) error {
		receivedSender <- fromPeerID
		receivedPayload <- msg.Payload
		return nil
	})

	addrs := srv.Multiaddrs()
	if len(addrs) == 0 {
		t.Fatalf("no listening addresses")
	}

	encPubKeyHex := hex.EncodeToString(xPriv.PublicKey().Bytes())
	testMsg := &Message{
		AppID:        "com.magicbox.loopback",
		TargetUserID: "user-123",
		Payload:      []byte("Hello self!"),
	}

	// Send to our own listening multiaddress
	err := srv.SendTo(ctx, addrs[0], encPubKeyHex, testMsg)
	if err != nil {
		t.Fatalf("failed to send: %v", err)
	}

	select {
	case sender := <-receivedSender:
		if sender != srv.HostID() {
			t.Errorf("expected sender %q, got %q", srv.HostID(), sender)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for loopback sender ID")
	}

	select {
	case payload := <-receivedPayload:
		if string(payload) != "Hello self!" {
			t.Errorf("expected payload %q, got %q", "Hello self!", string(payload))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for loopback payload")
	}
}

func TestParsePEMToLibp2pKey_ValidKey(t *testing.T) {
	mnemonic, _ := internalcrypto.GenerateMnemonic()
	edPriv, _ := internalcrypto.DeriveIdentityKey(mnemonic, 0)

	pemBytes, err := internalcrypto.MarshalPrivateKey(edPriv)
	if err != nil {
		t.Fatalf("failed to marshal key: %v", err)
	}

	libp2pKey, err := ParsePEMToLibp2pKey(pemBytes)
	if err != nil {
		t.Fatalf("ParsePEMToLibp2pKey failed: %v", err)
	}

	if libp2pKey == nil {
		t.Fatal("expected non-nil libp2p key")
	}

	// Verify the key matches by comparing peer IDs
	peerID, err := libp2pcrypto.UnmarshalEd25519PrivateKey(edPriv)
	if err != nil {
		t.Fatalf("failed to unmarshal for comparison: %v", err)
	}

	rawOriginal, _ := peerID.Raw()
	rawParsed, _ := libp2pKey.Raw()
	if string(rawOriginal) != string(rawParsed) {
		t.Error("parsed key does not match original")
	}
}

func TestParsePEMToLibp2pKey_InvalidPEMFails(t *testing.T) {
	_, err := ParsePEMToLibp2pKey([]byte("not a valid PEM"))
	if err == nil {
		t.Error("expected error for invalid PEM, got nil")
	}
}

func TestParsePEMToLibp2pKey_X25519KeyFails(t *testing.T) {
	mnemonic, _ := internalcrypto.GenerateMnemonic()
	xPriv, _ := internalcrypto.DeriveEncryptionKey(mnemonic, 0)

	pemBytes, err := internalcrypto.MarshalPrivateKey(xPriv)
	if err != nil {
		t.Fatalf("failed to marshal X25519 key: %v", err)
	}

	_, err = ParsePEMToLibp2pKey(pemBytes)
	if err == nil {
		t.Error("expected error when passing X25519 PEM to ParsePEMToLibp2pKey, got nil")
	}
}
