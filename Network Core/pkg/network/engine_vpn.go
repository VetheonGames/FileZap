package network

import (
    "context"
    "fmt"

    "github.com/libp2p/go-libp2p/core/host"
    "github.com/VetheonGames/FileZap/NetworkCore/pkg/vpn"
)

// VPNConfig holds configuration for the VPN overlay
type VPNConfig struct {
    Enabled       bool
    NetworkCIDR   string
    InterfaceName string
    NetworkKey    string // Shared key for deterministic peer IDs
}

// DefaultVPNConfig returns default VPN settings
func DefaultVPNConfig() *VPNConfig {
    return &VPNConfig{
        Enabled:       false,
        NetworkCIDR:   "10.42.0.0/16",
        InterfaceName: "tun0",
        NetworkKey:    "",
    }
}

// initVPN initializes the VPN overlay if enabled
func (e *NetworkEngine) initVPN(ctx context.Context, h host.Host, cfg *VPNConfig) error {
    if !cfg.Enabled {
        return nil
    }

    // Create VPN config
    vpnConfig := vpn.DefaultConfig()
    vpnConfig.NetworkCIDR = cfg.NetworkCIDR
    vpnConfig.InterfaceName = cfg.InterfaceName

    // Create VPN manager
    vpnManager, err := vpn.NewVPNManager(ctx, h, vpnConfig)
    if err != nil {
        return fmt.Errorf("failed to create VPN manager: %w", err)
    }

    // Create VPN discovery service
    discovery, err := vpn.NewDiscovery(ctx, h, e.transportNode.dht, e.transportNode.pubsub, vpnManager)
    if err != nil {
        vpnManager.Close()
        return fmt.Errorf("failed to create VPN discovery: %w", err)
    }

    e.vpnManager = vpnManager
    e.vpnDiscovery = discovery
    return nil
}
