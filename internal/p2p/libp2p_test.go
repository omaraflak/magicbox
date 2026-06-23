package p2p

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"sync"
	"testing"
	"time"

	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	internalcrypto "github.com/magicbox/core/internal/crypto"
	"github.com/magicbox/core/internal/logging"
)

func rsaToLibp2pKey(rsaPriv *rsa.PrivateKey) (libp2pcrypto.PrivKey, error) {
	der := x509.MarshalPKCS1PrivateKey(rsaPriv)
	return libp2pcrypto.UnmarshalRsaPrivateKey(der)
}

func TestLibp2pServiceFlow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Generate local identities
	rsaPriv1, err := internalcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("failed to generate rsa key 1: %v", err)
	}
	p2pKey1, err := rsaToLibp2pKey(rsaPriv1)
	if err != nil {
		t.Fatalf("failed to convert key 1: %v", err)
	}

	rsaPriv2, err := internalcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("failed to generate rsa key 2: %v", err)
	}
	p2pKey2, err := rsaToLibp2pKey(rsaPriv2)
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

	srv1 := NewLibp2pService(p2pKey1, []string{"/ip4/127.0.0.1/tcp/0"}, logger1)
	srv2 := NewLibp2pService(p2pKey2, []string{"/ip4/127.0.0.1/tcp/0"}, logger2)

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
	// Node 2 addresses has the format: /ip4/127.0.0.1/tcp/xxxxx/p2p/PeerID
	addrs2 := srv2.Multiaddrs()
	if len(addrs2) == 0 {
		t.Fatalf("node 2 has no listening multiaddresses")
	}

	// Use localhost address for test reliability
	var targetAddr string
	for _, a := range addrs2 {
		targetAddr = a
		break
	}

	testMsg := &Message{
		AppID:   "com.magicbox.test",
		Payload: []byte("Hello from peer 1!"),
	}

	err = srv1.SendTo(ctx, targetAddr, testMsg)
	if err != nil {
		t.Fatalf("failed to send message: %v", err)
	}

	// 5. Assert message reception
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

	rsaPriv1, _ := internalcrypto.GenerateKeyPair()
	p2pKey1, _ := rsaToLibp2pKey(rsaPriv1)
	rsaPriv2, _ := internalcrypto.GenerateKeyPair()
	p2pKey2, _ := rsaToLibp2pKey(rsaPriv2)

	logger1, _ := logging.New(t.TempDir())
	defer logger1.Close()
	logger2, _ := logging.New(t.TempDir())
	defer logger2.Close()

	srv1 := NewLibp2pService(p2pKey1, []string{"/ip4/127.0.0.1/tcp/0"}, logger1)
	srv2 := NewLibp2pService(p2pKey2, []string{"/ip4/127.0.0.1/tcp/0"}, logger2)

	_ = srv1.Start(ctx)
	defer srv1.Stop()
	_ = srv2.Start(ctx)
	defer srv2.Stop()

	// Wait, we don't register any handler for "com.magicbox.unhandled" on srv2.
	// We want to verify it doesn't crash.
	addrs2 := srv2.Multiaddrs()
	targetAddr := addrs2[0]

	testMsg := &Message{
		AppID:   "com.magicbox.unhandled",
		Payload: []byte("Some data"),
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := srv1.SendTo(ctx, targetAddr, testMsg)
		if err != nil {
			t.Errorf("unexpected error on unhandled protocol send: %v", err)
		}
	}()
	wg.Wait()
}
