package internal

import (
    "time"

    "github.com/VetheonGames/FileZap/NetworkCore/pkg/network/api"
)

// NetworkConfig holds the internal network configuration
type NetworkConfig struct {
    Transport     TransportConfig
    MetadataStore string
    ChunkCacheDir string
    VPN          *api.VPNConfig
}

// TransportConfig holds internal transport configuration
type TransportConfig struct {
    ListenAddrs     []string
    ListenPort      int
    EnableQUIC      bool
    EnableTCP       bool
    EnableRelay     bool
    EnableAutoRelay bool
    EnableHolePunch bool
    QUICOpts        QUICOptions
}

// QUICOptions configures QUIC transport behavior
type QUICOptions struct {
    MaxStreams       uint32
    KeepAlivePeriod  time.Duration
    HandshakeTimeout time.Duration
    IdleTimeout      time.Duration
}

// Default values for various settings
const (
    DefaultMaxStreams        = 100
    DefaultKeepAlivePeriod  = 30 * time.Second
    DefaultHandshakeTimeout = 10 * time.Second
    DefaultIdleTimeout     = 60 * time.Second
)

// DefaultConfig returns a new NetworkConfig with default values
func DefaultConfig() *NetworkConfig {
    return &NetworkConfig{
        Transport: TransportConfig{
            EnableQUIC: true,
            EnableTCP:  true,
            QUICOpts: QUICOptions{
                MaxStreams:       DefaultMaxStreams,
                KeepAlivePeriod:  DefaultKeepAlivePeriod,
                HandshakeTimeout: DefaultHandshakeTimeout,
                IdleTimeout:      DefaultIdleTimeout,
            },
        },
        MetadataStore: "metadata",
        ChunkCacheDir: "chunks",
        VPN:          nil,  // VPN is disabled by default
    }
}
