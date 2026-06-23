package p2p

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"sync"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/magicbox/core/internal/logging"
	"github.com/multiformats/go-multiaddr"
)

const ProtocolID = protocol.ID("/magicbox/messaging/1.0.0")

type Libp2pService struct {
	privKey     crypto.PrivKey
	listenAddrs []string
	host        host.Host
	logger      *logging.Logger

	mu       sync.RWMutex
	handlers map[string]Handler
}

// NewLibp2pService creates a new libp2p service instance.
func NewLibp2pService(privKey crypto.PrivKey, listenAddrs []string, logger *logging.Logger) *Libp2pService {
	return &Libp2pService{
		privKey:     privKey,
		listenAddrs: listenAddrs,
		handlers:    make(map[string]Handler),
		logger:      logger,
	}
}

// Start boots the libp2p host.
func (s *Libp2pService) Start(ctx context.Context) error {
	opts := []libp2p.Option{
		libp2p.Identity(s.privKey),
	}

	if len(s.listenAddrs) > 0 {
		opts = append(opts, libp2p.ListenAddrStrings(s.listenAddrs...))
	} else {
		opts = append(opts, libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/tcp/4001",
			"/ip4/0.0.0.0/udp/4001/quic-v1",
		))
	}

	h, err := libp2p.New(opts...)
	if err != nil {
		// Fallback to random ports if defaults are bound
		fallbackOpts := []libp2p.Option{
			libp2p.Identity(s.privKey),
			libp2p.ListenAddrStrings(
				"/ip4/0.0.0.0/tcp/0",
				"/ip4/0.0.0.0/udp/0/quic-v1",
			),
		}
		h, err = libp2p.New(fallbackOpts...)
		if err != nil {
			return fmt.Errorf("libp2p: failed to start host: %w", err)
		}
	}

	s.host = h
	s.host.SetStreamHandler(ProtocolID, s.handleStream)

	s.logger.Info("P2P libp2p host started",
		logging.F("peer_id", s.HostID()),
		logging.F("addresses", s.Multiaddrs()),
	)

	return nil
}

// Stop closes the libp2p host.
func (s *Libp2pService) Stop() error {
	if s.host != nil {
		return s.host.Close()
	}
	return nil
}

// HostID returns the peer ID string.
func (s *Libp2pService) HostID() string {
	if s.host == nil {
		return ""
	}
	return s.host.ID().String()
}

// Multiaddrs returns full multiaddress strings including /p2p/PeerID.
func (s *Libp2pService) Multiaddrs() []string {
	if s.host == nil {
		return nil
	}
	var addrs []string
	peerID := s.HostID()
	for _, a := range s.host.Addrs() {
		addrs = append(addrs, fmt.Sprintf("%s/p2p/%s", a.String(), peerID))
	}
	return addrs
}

// RegisterHandler registers a message handler for a protocol type.
func (s *Libp2pService) RegisterHandler(protocolType string, handler Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[protocolType] = handler
}

// SendTo dials a target multiaddress and writes the message payload.
func (s *Libp2pService) SendTo(ctx context.Context, destMultiaddr string, msg *Message) error {
	if s.host == nil {
		return fmt.Errorf("libp2p: host not started")
	}

	addr, err := multiaddr.NewMultiaddr(destMultiaddr)
	if err != nil {
		return fmt.Errorf("libp2p: invalid multiaddress %q: %w", destMultiaddr, err)
	}

	info, err := peer.AddrInfoFromP2pAddr(addr)
	if err != nil {
		return fmt.Errorf("libp2p: failed to extract peer info from addr: %w", err)
	}

	if err := s.host.Connect(ctx, *info); err != nil {
		return fmt.Errorf("libp2p: failed to connect to peer: %w", err)
	}

	stream, err := s.host.NewStream(network.WithUseTransient(ctx, "transit"), info.ID, ProtocolID)
	if err != nil {
		return fmt.Errorf("libp2p: failed to open stream to peer: %w", err)
	}
	defer stream.Close()

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(msg); err != nil {
		return fmt.Errorf("libp2p: failed to encode message: %w", err)
	}

	return nil
}

// handleStream processes incoming streams from remote peers.
func (s *Libp2pService) handleStream(stream network.Stream) {
	defer stream.Close()

	fromPeer := stream.Conn().RemotePeer().String()
	decoder := json.NewDecoder(stream)

	for {
		var msg Message
		err := decoder.Decode(&msg)
		if err == io.EOF {
			break
		}
		if err != nil {
			s.logger.Error("libp2p: failed to decode incoming message",
				logging.F("from", fromPeer),
				logging.F("error", err.Error()))
			return
		}

		s.mu.RLock()
		handler, exists := s.handlers[msg.ProtocolType]
		s.mu.RUnlock()

		if !exists {
			s.logger.Error("libp2p: unhandled protocol message type",
				logging.F("type", msg.ProtocolType),
				logging.F("from", fromPeer))
			continue
		}

		ctx := context.Background()
		if err := handler(ctx, fromPeer, &msg); err != nil {
			s.logger.Error("libp2p: handler failed for protocol type",
				logging.F("type", msg.ProtocolType),
				logging.F("from", fromPeer),
				logging.F("error", err.Error()))
		}
	}
}

// ParsePEMToPrivKey decodes an RSA private key PEM block and unmarshals it to a libp2p PrivKey.
func ParsePEMToPrivKey(pemBytes []byte) (crypto.PrivKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, fmt.Errorf("invalid private key PEM type")
	}
	return crypto.UnmarshalRsaPrivateKey(block.Bytes)
}
