package internal

import (
    "context"
    "fmt"

    "github.com/libp2p/go-libp2p"
    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/peer"
    "github.com/libp2p/go-libp2p/p2p/security/noise"
    dht "github.com/libp2p/go-libp2p-kad-dht"
    pubsub "github.com/libp2p/go-libp2p-pubsub"
    libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"
)

// networkNode is the concrete implementation of Node interface
type networkNode struct {
    host     host.Host
    dht      *dht.IpfsDHT
    pubsub   *pubsub.PubSub
}

func newNetworkNode(ctx context.Context, cfg *TransportConfig) (*networkNode, error) {
    // Create libp2p node options
    opts := []libp2p.Option{
        libp2p.ListenAddrStrings(cfg.ListenAddrs...),
        libp2p.Security(noise.ID, noise.New),
    }

    // Add QUIC transport if enabled
    if cfg.EnableQUIC {
        opts = append(opts,
            libp2p.Transport(libp2pquic.NewTransport),
            libp2p.DefaultTransports,
        )
    }

    // Create host
    h, err := libp2p.New(opts...)
    if err != nil {
        return nil, fmt.Errorf("failed to create host: %w", err)
    }

    // Create DHT
    kadDHT, err := dht.New(ctx, h, 
        dht.Mode(dht.ModeServer),
        dht.ProtocolPrefix("/filezap"),
    )
    if err != nil {
        h.Close()
        return nil, fmt.Errorf("failed to create DHT: %w", err)
    }

    // Bootstrap DHT
    if err := kadDHT.Bootstrap(ctx); err != nil {
        h.Close()
        kadDHT.Close()
        return nil, fmt.Errorf("failed to bootstrap DHT: %w", err)
    }

    // Create pubsub with QUIC optimizations if enabled
    psOpts := []pubsub.Option{
        pubsub.WithMessageSigning(true),
        pubsub.WithStrictSignatureVerification(true),
    }
    if cfg.EnableQUIC {
        psOpts = append(psOpts,
            pubsub.WithMaxMessageSize(10 * 1024 * 1024), // 10MB
            pubsub.WithValidateQueueSize(256),
        )
    }
    ps, err := pubsub.NewGossipSub(ctx, h, psOpts...)
    if err != nil {
        h.Close()
        kadDHT.Close()
        return nil, fmt.Errorf("failed to create pubsub: %w", err)
    }

    return &networkNode{
        host:    h,
        dht:     kadDHT,
        pubsub:  ps,
    }, nil
}

// Implement Node interface

func (n *networkNode) Close() error {
    var errs []error
    if err := n.pubsub.Close(); err != nil {
        errs = append(errs, err)
    }
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

func (n *networkNode) GetHost() host.Host {
    return n.host
}

func (n *networkNode) GetDHT() *dht.IpfsDHT {
    return n.dht
}

func (n *networkNode) GetPubSub() *pubsub.PubSub {
    return n.pubsub
}

func (n *networkNode) Connect(pi peer.AddrInfo) error {
    return n.host.Connect(context.Background(), pi)
}
