package network

import (
    "context"
    "fmt"
    "sync"
    "time"

    "github.com/libp2p/go-libp2p"
    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/peer"
    "github.com/libp2p/go-libp2p/p2p/security/noise"
    dht "github.com/libp2p/go-libp2p-kad-dht"
    pubsub "github.com/libp2p/go-libp2p-pubsub"
    libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"

    "github.com/VetheonGames/FileZap/NetworkCore/pkg/network/api"
)

// OverlayNetwork manages peer connections in the network overlay
type OverlayNetwork struct {
    neighbors map[peer.ID]time.Time
    maxPeers int
    mu       sync.RWMutex
}

// pubSubAdapter wraps pubsub.PubSub to implement the PubSub interface
type pubSubAdapter struct {
    *pubsub.PubSub
}

// Publish implements the PubSub interface
func (p *pubSubAdapter) Publish(topic string, data []byte) error {
    return p.PubSub.Publish(topic, data)
}

// Subscribe implements the PubSub interface
func (p *pubSubAdapter) Subscribe(topic string) (Subscription, error) {
    t, err := p.PubSub.Join(topic)
    if err != nil {
        return nil, err
    }
    sub, err := t.Subscribe()
    if err != nil {
        t.Close()
        return nil, err
    }
    return &subscriptionAdapter{sub, t}, nil
}

// subscriptionAdapter wraps pubsub.Subscription to implement the Subscription interface
type subscriptionAdapter struct {
    sub *pubsub.Subscription
    topic *pubsub.Topic
}

// Next implements the Subscription interface
func (s *subscriptionAdapter) Next() (Message, error) {
    msg, err := s.sub.Next(context.Background())
    if err != nil {
        return nil, err
    }
    return &messageAdapter{msg}, nil
}

// Cancel implements the Subscription interface
func (s *subscriptionAdapter) Cancel() {
    s.sub.Cancel()
    s.topic.Close()
}

// messageAdapter wraps pubsub.Message to implement the Message interface
type messageAdapter struct {
    *pubsub.Message
}

func (m *messageAdapter) Data() []byte {
    return m.Message.Data
}

func (m *messageAdapter) From() peer.ID {
    return m.Message.ReceivedFrom
}

func (m *messageAdapter) Topics() []string {
    if m.Message.Topic == nil {
        return nil
    }
    return []string{*m.Message.Topic}
}

// dhtAdapter wraps dht.IpfsDHT to implement the DHT interface
type dhtAdapter struct {
    *dht.IpfsDHT
}

// Bootstrap implements the DHT interface
func (d *dhtAdapter) Bootstrap(ctx context.Context) error {
    return d.IpfsDHT.Bootstrap(ctx)
}

// transportNode implements the NetworkNode interface
type transportNode struct {
    host     host.Host
    dht      *dhtAdapter  
    pubsub   *pubSubAdapter
    overlay  *OverlayNetwork
}

// DefaultTransportConfig returns default transport settings
func DefaultTransportConfig() *api.TransportConfig {
    return &api.TransportConfig{
        ListenAddrs: []string{
            "/ip4/0.0.0.0/tcp/0",
            "/ip4/0.0.0.0/udp/0/quic",
            "/ip6/::/tcp/0",
            "/ip6/::/udp/0/quic",
        },
        ListenPort:      0,
        EnableQUIC:      true,
        EnableTCP:       true,
        EnableRelay:     true,
        EnableAutoRelay: true,
        EnableHolePunch: true,
        QUICOpts: api.QUICOptions{
            MaxStreams:      100,
            KeepAlivePeriod: 30 * time.Second,
            HandshakeTimeout: 10 * time.Second,
            IdleTimeout:     60 * time.Second,
        },
    }
}

// NewTransportNode creates a new libp2p host with the specified transport configuration
func NewTransportNode(ctx context.Context, cfg *api.TransportConfig) (*transportNode, error) {
    opts := []libp2p.Option{
        libp2p.ListenAddrStrings(cfg.ListenAddrs...),
        libp2p.Security(noise.ID, noise.New),
        libp2p.EnableHolePunching(),
    }

    // Configure QUIC transport
    if cfg.EnableQUIC {
        opts = append(opts, 
            libp2p.Transport(libp2pquic.NewTransport),
            libp2p.DefaultTransports,
        )
    }

    // Create libp2p host
    h, err := libp2p.New(opts...)
    if err != nil {
        return nil, fmt.Errorf("failed to create libp2p host: %w", err)
    }

    // Create DHT with QUIC-specific settings
    dhtOpts := []dht.Option{
        dht.Mode(dht.ModeServer),
        dht.BootstrapPeers(dht.GetDefaultBootstrapPeerAddrInfos()...),
        dht.ProtocolPrefix("/filezap"),
    }

    kdht, err := dht.New(ctx, h, dhtOpts...)
    if err != nil {
        h.Close()
        return nil, fmt.Errorf("failed to create DHT: %w", err)
    }

    // Bootstrap DHT with retries
    if err := bootstrapDHT(ctx, kdht); err != nil {
        h.Close()
        return nil, err
    }

    // Create pubsub with QUIC-optimized settings
    ps, err := createPubSub(ctx, h, cfg.EnableQUIC)
    if err != nil {
        h.Close()
        kdht.Close()
        return nil, err
    }

    node := &transportNode{
        host:    h,
        dht:     &dhtAdapter{kdht},
        pubsub:  &pubSubAdapter{ps},
        overlay: NewOverlayNetwork(h.ID()),
    }

    return node, nil
}

// GetHost returns the libp2p host
func (n *transportNode) GetHost() host.Host {
    return n.host
}

// GetDHT returns the DHT instance
func (n *transportNode) GetDHT() DHT {
    return n.dht
}

// GetPubSub returns the pubsub instance
func (n *transportNode) GetPubSub() PubSub {
    return n.pubsub
}

// Close shuts down the network node
func (n *transportNode) Close() error {
    var errs []error
    if err := n.dht.Close(); err != nil {
        errs = append(errs, err)
    }
    if err := n.host.Close(); err != nil {
        errs = append(errs, err)
    }
    if len(errs) > 0 {
        return fmt.Errorf("errors closing node: %v", errs)
    }
    return nil
}

// bootstrapDHT bootstraps the DHT with retries
func bootstrapDHT(ctx context.Context, kdht *dht.IpfsDHT) error {
    if err := kdht.Bootstrap(ctx); err != nil {
        return fmt.Errorf("failed to bootstrap DHT: %w", err)
    }

    timeout := time.After(30 * time.Second)
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-timeout:
            if len(kdht.RoutingTable().ListPeers()) == 0 {
                return fmt.Errorf("failed to find any peers during bootstrap")
            }
            return nil
        case <-ticker.C:
            if len(kdht.RoutingTable().ListPeers()) > 0 {
                return nil
            }
        case <-ctx.Done():
            return ctx.Err()
        }
    }
}

// createPubSub creates a new pubsub service with optimized settings
func createPubSub(ctx context.Context, h host.Host, quicEnabled bool) (*pubsub.PubSub, error) {
    opts := []pubsub.Option{
        pubsub.WithMessageSigning(true),
        pubsub.WithStrictSignatureVerification(true),
    }

    if quicEnabled {
        opts = append(opts,
            pubsub.WithMaxMessageSize(10 * 1024 * 1024), // 10MB max message size
            pubsub.WithValidateQueueSize(256),
        )
    }

    return pubsub.NewGossipSub(ctx, h, opts...)
}

// NewOverlayNetwork creates a new overlay network
func NewOverlayNetwork(selfID peer.ID) *OverlayNetwork {
    return &OverlayNetwork{
        neighbors:  make(map[peer.ID]time.Time),
        maxPeers:   50,
    }
}
