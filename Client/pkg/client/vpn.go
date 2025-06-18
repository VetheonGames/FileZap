package client

import (
    "context"
    "fmt"

    "github.com/VetheonGames/FileZap/NetworkCore/pkg/network"
    "github.com/VetheonGames/FileZap/NetworkCore/pkg/vpn"
)

// VPNConfig holds the client-side VPN configuration
type VPNConfig struct {
    Enabled       bool   // Whether to enable VPN functionality
    NetworkCIDR   string // Network CIDR for VPN (e.g., "10.42.0.0/16")
    InterfaceName string // Name for TUN interface (e.g., "tun0")
    NetworkKey    string // Shared key for the VPN network
}

// DefaultVPNConfig returns default VPN settings
func DefaultVPNConfig() *VPNConfig {
    return &VPNConfig{
        Enabled:       false,
        NetworkCIDR:   "10.42.0.0/16",
        InterfaceName: "tun0",
        NetworkKey:    "MeshGenesisKey",
    }
}

// ConnectVPN initializes and connects to the VPN network
func (c *Client) ConnectVPN(cfg *VPNConfig) error {
    if !cfg.Enabled {
        return nil
    }

    // Check if we already have a running VPN
    if c.vpnManager != nil {
        return fmt.Errorf("VPN is already running")
    }

    // Update network engine config
    engineCfg := network.DefaultNetworkConfig()
    engineCfg.VPN = &network.VPNConfig{
        Enabled:       cfg.Enabled,
        NetworkCIDR:   cfg.NetworkCIDR,
        InterfaceName: cfg.InterfaceName,
        NetworkKey:    cfg.NetworkKey,
    }

    // Create network engine with VPN support
    engine, err := network.NewNetworkEngine(c.ctx, engineCfg)
    if err != nil {
        return fmt.Errorf("failed to create network engine with VPN: %w", err)
    }

    // Store reference to the VPN manager
    if vpnManager := engine.GetVPNManager(); vpnManager != nil {
        c.vpnManager = vpnManager
    }

    c.engine = engine
    return nil
}

// DisconnectVPN shuts down the VPN connection
func (c *Client) DisconnectVPN() error {
    if c.vpnManager == nil {
        return nil
    }

    return c.vpnManager.Close()
}

// GetVPNStatus returns the current VPN connection status
func (c *Client) GetVPNStatus() *VPNStatus {
    if c.vpnManager == nil {
        return &VPNStatus{
            Connected: false,
        }
    }

    peers := make([]VPNPeer, 0)
    // Add code to get peer information from VPN manager

    return &VPNStatus{
        Connected:   true,
        LocalIP:     c.vpnManager.GetLocalIP(),
        PeerCount:   len(peers),
        ActivePeers: peers,
    }
}

// VPNStatus represents the current state of the VPN connection
type VPNStatus struct {
    Connected   bool
    LocalIP     string
    PeerCount   int
    ActivePeers []VPNPeer
}

// VPNPeer represents a connected peer in the VPN network
type VPNPeer struct {
    ID       string
    IP       string
    LastSeen int64
}
