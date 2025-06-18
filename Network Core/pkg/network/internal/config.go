package internal

import (
    "time"
)

// NetworkConfig holds the internal network configuration
type NetworkConfig struct {
    Transport     TransportConfig
    MetadataStore string
    ChunkCacheDir string
    VPN          *VPNConfig
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

// VPNConfig holds VPN configuration
type VPNConfig struct {
    Enabled       bool
    NetworkCIDR   string
    InterfaceName string
    NetworkKey    string
}

// Default values for various settings
const (
    DefaultMaxStreams        = 100
    DefaultKeepAlivePeriod  = 30 * time.Second
    DefaultHandshakeTimeout = 10 * time.Second
    DefaultIdleTimeout     = 60 * time.Second
)

// DefaultVPNConfig returns default VPN settings
func DefaultVPNConfig() *VPNConfig {
    return &VPNConfig{
        Enabled:       false,
        NetworkCIDR:   "10.42.0.0/16",
        InterfaceName: "tun0",
        NetworkKey:    "",
    }
}
