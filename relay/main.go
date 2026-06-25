package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	relay "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	ws "github.com/libp2p/go-libp2p/p2p/transport/websocket"
)

func main() {
	// Cloud Run sets the PORT environment variable
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	var opts []libp2p.Option
	opts = append(opts,
		libp2p.Transport(ws.New),
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%s/ws", port)),
	)

	// Check if a seed is set in env for deterministic peer ID
	seed := os.Getenv("RELAY_SEED")
	if seed != "" {
		privKey, err := generateKeyFromSeed(seed)
		if err != nil {
			log.Fatalf("failed to generate key from seed: %v", err)
		}
		opts = append(opts, libp2p.Identity(privKey))
		log.Println("Loaded deterministic P2P identity key from seed")
	} else {
		log.Println("WARNING: RELAY_SEED env var not set. Running with a temporary random P2P identity key.")
	}

	// 1. Initialize libp2p host with WebSocket transport
	h, err := libp2p.New(opts...)
	if err != nil {
		log.Fatalf("failed to start libp2p host: %v", err)
	}
	defer h.Close()

	// 2. Configure Circuit Relay v2 limits (to prevent abuse on public cloud hosting)
	relayLimits := relay.Resources{
		MaxReservations:        256,              // Max concurrent client reservations
		MaxCircuits:            64,               // Max concurrent active proxy streams
		BufferSize:             2048,             // Connection buffer size
		ReservationTTL:         30 * time.Minute, // Max duration for a reservation
		MaxReservationsPerPeer: 4,                // Limit reservations per peer ID
		MaxReservationsPerIP:   8,                // Limit reservations per IP address
		Limit: &relay.RelayLimit{
			Duration: 5 * time.Minute,         // Max duration for an active transfer
			Data:     100 * 1024 * 1024,       // Limit transfers to 100MB per session
		},
	}

	// 3. Start the relay service
	_, err = relay.New(h, relay.WithResources(relayLimits))
	if err != nil {
		log.Fatalf("failed to start relay service: %v", err)
	}

	log.Printf("Magicbox Relay Server listening on port: %s (WebSockets)", port)
	log.Println("P2P Multiaddresses:")
	for _, addr := range h.Addrs() {
		log.Printf("- %s/p2p/%s\n", addr.String(), h.ID().String())
	}

	// Handle graceful shutdown
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch

	log.Println("Shutting down relay service...")
}

func generateKeyFromSeed(seed string) (crypto.PrivKey, error) {
	// Hash the seed string to get a stable 32-byte key seed
	hash := sha256.Sum256([]byte(seed))

	// Create the private key from the seed
	privKey := ed25519.NewKeyFromSeed(hash[:])

	// Convert to libp2p's private key format
	return crypto.UnmarshalEd25519PrivateKey(privKey)
}
