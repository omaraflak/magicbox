package p2p

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
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
	dht         *dht.IpfsDHT
	logger      *logging.Logger

	mu             sync.RWMutex
	handlers       map[string]Handler
	defaultHandler Handler
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
	bootstrapPeers := []string{
		"/dns4/magicbox-relay-626811923438.europe-west2.run.app/tcp/443/wss/p2p/12D3KooWB9NpjhNXDQK1GTWWftN7ZBSuWZ3XePSuLvPaATQnYTfE",
	}

	var staticRelays []peer.AddrInfo
	for _, addrStr := range bootstrapPeers {
		addr, err := multiaddr.NewMultiaddr(addrStr)
		if err != nil {
			continue
		}
		pi, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			continue
		}
		staticRelays = append(staticRelays, *pi)
	}

	opts := []libp2p.Option{
		libp2p.Identity(s.privKey),
		libp2p.EnableRelay(),
		libp2p.EnableAutoRelayWithStaticRelays(staticRelays),
		libp2p.EnableHolePunching(),
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
			libp2p.EnableRelay(),
			libp2p.EnableAutoRelayWithStaticRelays(staticRelays),
			libp2p.EnableHolePunching(),
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

	// Start DHT in server mode so we can publish/route IP mappings
	kademliaDHT, err := dht.New(ctx, s.host, dht.Mode(dht.ModeServer))
	if err != nil {
		return fmt.Errorf("libp2p: failed to start DHT: %w", err)
	}
	s.dht = kademliaDHT

	// Bootstrap the DHT
	if err := s.dht.Bootstrap(ctx); err != nil {
		return fmt.Errorf("libp2p: failed to bootstrap DHT: %w", err)
	}

	// Dial bootstrap peers asynchronously
	for _, info := range staticRelays {
		go func(info peer.AddrInfo) {
			dialCtx, dialCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer dialCancel()
			if err := s.host.Connect(dialCtx, info); err != nil {
				s.logger.Debug("libp2p: failed to connect to bootstrap node",
					logging.F("peer", info.ID.String()),
					logging.F("error", err.Error()),
				)
			} else {
				s.logger.Info("libp2p: connected to bootstrap node",
					logging.F("peer", info.ID.String()),
				)
			}
		}(info)
	}

	s.logger.Info("P2P libp2p host started",
		logging.F("peer_id", s.HostID()),
		logging.F("addresses", s.Multiaddrs()),
	)

	return nil
}

// Stop closes the libp2p host and DHT.
func (s *Libp2pService) Stop() error {
	var dhtErr error
	if s.dht != nil {
		dhtErr = s.dht.Close()
	}
	var hostErr error
	if s.host != nil {
		hostErr = s.host.Close()
	}
	if dhtErr != nil {
		return dhtErr
	}
	return hostErr
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
func (s *Libp2pService) RegisterHandler(appID string, handler Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[appID] = handler
}

// SetDefaultHandler registers a fallback handler for any unhandled protocol type.
func (s *Libp2pService) SetDefaultHandler(handler Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.defaultHandler = handler
}

// SendTo resolves the peer target (which can be a Peer ID, a magicbox://invite link, or a full multiaddress) and writes the message payload.
func (s *Libp2pService) SendTo(ctx context.Context, target string, msg *Message) error {
	if s.host == nil {
		return fmt.Errorf("libp2p: host not started")
	}

	// 1. Clean the target (support magicbox://invite/ schema)
	cleanTarget := strings.TrimPrefix(target, "magicbox://invite/")

	// Strip query string parameters like ?user_id=...
	if idx := strings.Index(cleanTarget, "?"); idx != -1 {
		cleanTarget = cleanTarget[:idx]
	}

	var peerID peer.ID

	// 2. Determine if cleanTarget is a valid Peer ID
	if id, parseErr := peer.Decode(cleanTarget); parseErr == nil {
		peerID = id
		s.logger.Info("libp2p: target is Peer ID, performing DHT lookup", logging.F("peer_id", peerID.String()))
		
		// Query DHT for peer address info
		info, dhtErr := s.dht.FindPeer(ctx, peerID)
		if dhtErr != nil {
			return fmt.Errorf("libp2p: failed to resolve peer ID %q via DHT: %w", peerID.String(), dhtErr)
		}

		if err := s.host.Connect(ctx, info); err != nil {
			return fmt.Errorf("libp2p: failed to connect to resolved peer: %w", err)
		}
	} else {
		// 3. Otherwise, treat as a full multiaddress
		addr, err := multiaddr.NewMultiaddr(cleanTarget)
		if err != nil {
			return fmt.Errorf("libp2p: invalid destination multiaddress/peer ID %q: %w", cleanTarget, err)
		}

		info, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			return fmt.Errorf("libp2p: failed to extract peer info from addr: %w", err)
		}
		peerID = info.ID

		if err := s.host.Connect(ctx, *info); err != nil {
			return fmt.Errorf("libp2p: failed to connect to peer: %w", err)
		}
	}

	stream, err := s.host.NewStream(network.WithUseTransient(ctx, "transit"), peerID, ProtocolID)
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
		handler, exists := s.handlers[msg.AppID]
		if !exists {
			handler = s.defaultHandler
			exists = handler != nil
		}
		s.mu.RUnlock()

		if !exists {
			s.logger.Error("libp2p: unhandled protocol message type",
				logging.F("type", msg.AppID),
				logging.F("from", fromPeer))
			continue
		}

		ctx := context.Background()
		if err := handler(ctx, fromPeer, &msg); err != nil {
			s.logger.Error("libp2p: handler failed for protocol type",
				logging.F("type", msg.AppID),
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
