package network

import (
    "context"
    "fmt"

    "github.com/libp2p/go-libp2p/core/host"
    "github.com/VetheonGames/FileZap/NetworkCore/pkg/vpn"
)

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

// GetVPNManager returns the VPN manager if enabled
func (e *NetworkEngine) GetVPNManager() *vpn.VPNManager {
    return e.vpnManager
}

// GetVPNStatus returns the VPN status if enabled
func (e *NetworkEngine) GetVPNStatus() *vpn.VPNStatus {
    if e.vpnManager == nil {
        return nil
    }

    return &vpn.VPNStatus{
        Connected:   true,
        LocalIP:     e.vpnManager.GetLocalIP(),
        PeerCount:   len(e.vpnManager.GetPeers()),
        ActivePeers: e.vpnManager.GetActivePeers(),
    }
}
