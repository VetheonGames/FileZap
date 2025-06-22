package network

import (
    "time"

    "github.com/libp2p/go-libp2p/core/peer"
)

// NetworkConfig represents the configuration for the network
type NetworkConfig struct {
    Transport struct {
        ListenAddrs     []string
        ListenPort      int
        EnableQUIC      bool
        EnableTCP       bool
        EnableRelay     bool
        EnableAutoRelay bool
        EnableHolePunch bool
        QUICOpts        QUICOptions
    }
    MetadataStore string
    ChunkCacheDir string
    VPNConfig     *VPNConfig
}

// QUICOptions defines configuration for QUIC transport
type QUICOptions struct {
    MaxStreams       uint32
    KeepAlivePeriod time.Duration
    HandshakeTimeout time.Duration
    IdleTimeout      time.Duration
}

// VPNConfig defines VPN configuration options
type VPNConfig struct {
    Enabled       bool
    NetworkCIDR   string
    InterfaceName string
    NetworkKey    []byte
}

// VPNStatus represents the current state of VPN connections
type VPNStatus struct {
    Connected   bool
    LocalIP     string
    PeerCount   int
    ActivePeers []peer.ID
}

// DefaultNetworkConfig returns the default network configuration
func DefaultNetworkConfig() *NetworkConfig {
    return &NetworkConfig{
        ChunkCacheDir: "storage",
        MetadataStore: "metadata",
        Transport: struct {
            ListenAddrs     []string
            ListenPort      int
            EnableQUIC      bool
            EnableTCP       bool
            EnableRelay     bool
            EnableAutoRelay bool
            EnableHolePunch bool
            QUICOpts        QUICOptions
        }{
            ListenPort: 6001,
            EnableTCP:  true,
        },
    }
}
