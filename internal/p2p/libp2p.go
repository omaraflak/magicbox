package p2p

import (
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/x509"

	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	corecrypto "github.com/magicbox/core/internal/crypto"
	"github.com/magicbox/core/internal/invite"
	"github.com/magicbox/core/internal/logging"
	"github.com/multiformats/go-multiaddr"
)

const ProtocolID = protocol.ID("/magicbox/messaging/1.0.0")

type Libp2pService struct {
	privKey        crypto.PrivKey
	encryptionPriv *ecdh.PrivateKey
	listenAddrs    []string
	host           host.Host
	dht            *dht.IpfsDHT
	logger         *logging.Logger

	mu             sync.RWMutex
	handlers       map[string]Handler
	defaultHandler Handler
}

// NewLibp2pService creates a new libp2p service instance.
func NewLibp2pService(privKey crypto.PrivKey, encPriv *ecdh.PrivateKey, listenAddrs []string, logger *logging.Logger) *Libp2pService {
	return &Libp2pService{
		privKey:        privKey,
		encryptionPriv: encPriv,
		listenAddrs:    listenAddrs,
		handlers:       make(map[string]Handler),
		logger:         logger,
	}
}

// Start boots the libp2p host.
func (s *Libp2pService) Start(ctx context.Context) error {
	bootstrapPeers := []string{
		"/ip4/35.197.199.183/tcp/4001/ws/p2p/12D3KooWCpWVnUkkBKwu4gGUBtww7URbswPgVc86yTGkpqnAnH4f",
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
		libp2p.ForceReachabilityPrivate(),
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
			libp2p.ForceReachabilityPrivate(),
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

// SendTo resolves the peer target (which must be a magicbox://invite link) and writes the message payload.
func (s *Libp2pService) SendTo(ctx context.Context, target string, msg *Message) error {
	if s.host == nil {
		return fmt.Errorf("libp2p: host not started")
	}

	payload, err := invite.Parse(target)
	if err != nil {
		return fmt.Errorf("libp2p: invalid invite link: %w", err)
	}

	addr, err := multiaddr.NewMultiaddr(payload.Multiaddr)
	if err != nil {
		return fmt.Errorf("libp2p: invalid multiaddress %q: %w", payload.Multiaddr, err)
	}

	info, err := peer.AddrInfoFromP2pAddr(addr)
	if err != nil {
		return fmt.Errorf("libp2p: failed to extract peer info: %w", err)
	}
	peerID := info.ID

	if err := s.host.Connect(ctx, *info); err != nil {
		return fmt.Errorf("libp2p: failed to connect to peer: %w", err)
	}

	pubBytes, err := hex.DecodeString(payload.EncPubKey)
	if err != nil {
		return fmt.Errorf("libp2p: invalid enc_pub_key hex string: %w", err)
	}

	curve := ecdh.X25519()
	recipientXPub, err := curve.NewPublicKey(pubBytes)
	if err != nil {
		return fmt.Errorf("libp2p: failed to parse X25519 public key: %w", err)
	}

	senderEdPriv, err := getEd25519PrivKey(s.privKey)
	if err != nil {
		return fmt.Errorf("libp2p: failed to parse local signing key: %w", err)
	}

	encryptedPayload, err := corecrypto.EncryptAndSign(senderEdPriv, recipientXPub, msg.Payload)
	if err != nil {
		return fmt.Errorf("libp2p: failed to encrypt/sign payload: %w", err)
	}

	encryptedMsg := &Message{
		AppID:        msg.AppID,
		TargetUserID: msg.TargetUserID,
		Payload:      encryptedPayload,
	}

	stream, err := s.host.NewStream(network.WithUseTransient(ctx, "transit"), peerID, ProtocolID)
	if err != nil {
		return fmt.Errorf("libp2p: failed to open stream to peer: %w", err)
	}
	defer stream.Close()

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(encryptedMsg); err != nil {
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

		senderPeerID, err := peer.Decode(fromPeer)
		if err != nil {
			s.logger.Error("libp2p: failed to decode sender peer ID",
				logging.F("from", fromPeer),
				logging.F("error", err.Error()))
			continue
		}

		// EXTRACT static signing key directly from Peer ID (Public Key Pinning / Cryptographic binding)
		senderPubKey, err := senderPeerID.ExtractPublicKey()
		if err != nil {
			s.logger.Error("libp2p: failed to extract sender public key from peer ID",
				logging.F("from", fromPeer),
				logging.F("error", err.Error()))
			continue
		}

		senderEdPub, err := getEd25519PubKey(senderPubKey)
		if err != nil {
			s.logger.Error("libp2p: failed to parse sender public key",
				logging.F("from", fromPeer),
				logging.F("error", err.Error()))
			continue
		}

		if s.encryptionPriv == nil {
			s.logger.Error("libp2p: local encryption private key is nil")
			continue
		}

		decryptedPayload, err := corecrypto.DecryptAndVerify(s.encryptionPriv, senderEdPub, msg.Payload)
		if err != nil {
			s.logger.Error("libp2p: failed to decrypt/verify incoming payload",
				logging.F("from", fromPeer),
				logging.F("error", err.Error()))
			continue
		}

		msg.Payload = decryptedPayload

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

// ParsePEMToPrivKey decodes an Ed25519 PKCS#8 private key PEM block and unmarshals it to a libp2p PrivKey.
func ParsePEMToPrivKey(pemBytes []byte) (crypto.PrivKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("invalid private key PEM type")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PKCS8 private key: %w", err)
	}
	edPriv, ok := parsed.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an Ed25519 private key")
	}
	return crypto.UnmarshalEd25519PrivateKey(edPriv)
}

// ParsePEMToX25519PrivKey decodes an X25519 PKCS#8 private key PEM block.
func ParsePEMToX25519PrivKey(pemBytes []byte) (*ecdh.PrivateKey, error) {
	return corecrypto.UnmarshalX25519PrivateKey(pemBytes)
}

func getEd25519PubKey(pubKey crypto.PubKey) (ed25519.PublicKey, error) {
	rawPubBytes, err := pubKey.Raw()
	if err != nil {
		return nil, fmt.Errorf("failed to get raw public key: %w", err)
	}
	if len(rawPubBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid Ed25519 public key size: %d", len(rawPubBytes))
	}
	return ed25519.PublicKey(rawPubBytes), nil
}

func getEd25519PrivKey(privKey crypto.PrivKey) (ed25519.PrivateKey, error) {
	rawPriv, err := privKey.Raw()
	if err != nil {
		return nil, fmt.Errorf("failed to get raw private key: %w", err)
	}
	// go-libp2p's Ed25519 private key Raw() returns the 64-byte private key (seed + pub)
	if len(rawPriv) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid Ed25519 private key size: %d", len(rawPriv))
	}
	return ed25519.PrivateKey(rawPriv), nil
}

