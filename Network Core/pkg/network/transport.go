package network

import (
    "context"
    "fmt"
    "time"

    "github.com/libp2p/go-libp2p"
    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/peer"
    "github.com/libp2p/go-libp2p/p2p/security/noise"
    dht "github.com/libp2p/go-libp2p-kad-dht"
    pubsub "github.com/libp2p/go-libp2p-pubsub"
    libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"
)

// TransportConfig holds configuration for network transport
type TransportConfig struct {
    ListenAddrs     []string
    EnableQUIC      bool
    EnableTCP       bool
    EnableRelay     bool
    EnableAutoRelay bool
    EnableHolePunch bool
    QUICOpts        QUICOptions
}

// QUICOptions configures QUIC transport behavior
type QUICOptions struct {
    MaxStreams          uint32
    KeepAlivePeriod    time.Duration
    HandshakeTimeout   time.Duration
    IdleTimeout        time.Duration
}

// DefaultTransportConfig returns default transport settings
func DefaultTransportConfig() *TransportConfig {
    return &TransportConfig{
        ListenAddrs: []string{
            "/ip4/0.0.0.0/tcp/0",
            "/ip4/0.0.0.0/udp/0/quic",
            "/ip6/::/tcp/0",
            "/ip6/::/udp/0/quic",
        },
        EnableQUIC:      true,
        EnableTCP:       true,
        EnableRelay:     true,
        EnableAutoRelay: true,
        EnableHolePunch: true,
        QUICOpts: QUICOptions{
            MaxStreams:       100,
            KeepAlivePeriod: 30 * time.Second,
            HandshakeTimeout: 10 * time.Second,
            IdleTimeout:     60 * time.Second,
        },
    }
}

// NewTransportNode creates a new libp2p host with the specified transport configuration
func NewTransportNode(ctx context.Context, cfg *TransportConfig) (*NetworkNode, error) {
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

    // Always start in server mode

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

    node := &NetworkNode{
        host:    h,
        dht:     kdht,
        pubsub:  ps,
        overlay: NewOverlayNetwork(h.ID()),
    }

    return node, nil
}

// bootstrapDHT bootstraps the DHT with retries
func bootstrapDHT(ctx context.Context, kdht *dht.IpfsDHT) error {
    // Start DHT in server mode
    if err := kdht.Bootstrap(ctx); err != nil {
        return fmt.Errorf("failed to bootstrap DHT: %w", err)
    }

    // Wait for initial peer discovery
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
        // QUIC-optimized settings
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
