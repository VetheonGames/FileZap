package network

import (
    "context"
    "time"

    "github.com/VetheonGames/FileZap/NetworkCore/pkg/network/api"
    "github.com/VetheonGames/FileZap/NetworkCore/pkg/network/internal"
)

// NewNetworkBuilder creates a builder for configuring and creating a Network instance
func NewNetworkBuilder() api.NetworkBuilder {
    return api.NewNetworkBuilder()
}

// DefaultConfig returns a default Network configuration
func DefaultConfig() *api.NetworkConfig {
    return &api.NetworkConfig{
        Transport: api.TransportConfig{
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
                MaxStreams:       100,
                KeepAlivePeriod: 30 * time.Second,
                HandshakeTimeout: 10 * time.Second,
                IdleTimeout:      60 * time.Second,
            },
        },
        MetadataStore: "metadata",
        ChunkCacheDir: "chunks",
        VPNConfig:    nil,
    }
}

// Network settings and constants
const (
    DefaultRelayLimit = 5
    DefaultPeerLimit  = 50
)

// Factory function for creating network instances
func newNetwork(cfg *api.NetworkConfig, enableVPN bool) (api.Network, error) {
    // Create context
    ctx := context.Background()

    // Convert public config to internal config
    internalCfg := &internal.NetworkConfig{
        Transport: internal.TransportConfig{
            ListenAddrs:     cfg.Transport.ListenAddrs,
            ListenPort:      cfg.Transport.ListenPort,
            EnableQUIC:      cfg.Transport.EnableQUIC,
            EnableTCP:       cfg.Transport.EnableTCP,
            EnableRelay:     cfg.Transport.EnableRelay,
            EnableAutoRelay: cfg.Transport.EnableAutoRelay,
            EnableHolePunch: cfg.Transport.EnableHolePunch,
            QUICOpts: internal.QUICOptions{
                MaxStreams:       cfg.Transport.QUICOpts.MaxStreams,
                KeepAlivePeriod:  cfg.Transport.QUICOpts.KeepAlivePeriod,
                HandshakeTimeout: cfg.Transport.QUICOpts.HandshakeTimeout,
                IdleTimeout:      cfg.Transport.QUICOpts.IdleTimeout,
            },
        },
        MetadataStore: cfg.MetadataStore,
        ChunkCacheDir: cfg.ChunkCacheDir,
    }

    // Set up VPN if enabled
    if enableVPN {
        if cfg.VPNConfig == nil {
            cfg.VPNConfig = &api.VPNConfig{
                Enabled:       true,
                NetworkCIDR:   "10.42.0.0/16",
                InterfaceName: "tun0",
            }
        }
        internalCfg.VPN = &internal.VPNConfig{
            Enabled:       cfg.VPNConfig.Enabled,
            NetworkCIDR:   cfg.VPNConfig.NetworkCIDR,
            InterfaceName: cfg.VPNConfig.InterfaceName,
            NetworkKey:    cfg.VPNConfig.NetworkKey,
        }
    }

    // Create network instance
    return internal.NewNetworkEngine(ctx, internalCfg)
}
