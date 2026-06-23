package p2p

import (
	"context"
)

// Message represents a payload sent between peers.
type Message struct {
	AppID        string `json:"app_id"`
	TargetUserID string `json:"target_user_id"`
	Payload      []byte `json:"payload"`
}

// Handler handles incoming messages from remote peers.
type Handler func(ctx context.Context, fromPeerID string, msg *Message) error

// Service defines the interface for our libp2p P2P communication layer.
type Service interface {
	// Start starts the libp2p host and begins listening for incoming streams.
	Start(ctx context.Context) error

	// Stop stops the libp2p host.
	Stop() error

	// HostID returns the local host's peer ID string.
	HostID() string

	// Multiaddrs returns the list of listening multiaddresses for this host.
	Multiaddrs() []string

	// RegisterHandler registers a callback handler for a specific message protocol.
	RegisterHandler(appID string, handler Handler)

	// SendTo dials a remote peer using their multiaddress invitation and sends a message.
	SendTo(ctx context.Context, destMultiaddr string, msg *Message) error
}
