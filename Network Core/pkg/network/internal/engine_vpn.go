package internal

import (
    "context"
    "fmt"

    "github.com/libp2p/go-libp2p/core/host"
    "github.com/VetheonGames/FileZap/NetworkCore/pkg/vpn"
)

func (e *networkEngine) initVPN(ctx context.Context, h host.Host, cfg *VPNConfig) error {
    // Convert config
    vpnConfig := vpn.DefaultConfig()
    vpnConfig.NetworkCIDR = cfg.NetworkCIDR
    vpnConfig.InterfaceName = cfg.InterfaceName

    // Create VPN manager
    vpnManager, err := vpn.NewVPNManager(ctx, h, vpnConfig)
    if err != nil {
        return fmt.Errorf("failed to create VPN manager: %w", err)
    }

    // Create VPN discovery service
    discovery, err := vpn.NewDiscovery(
        ctx,
        h, 
        e.transportNode.GetDHT(),
        e.transportNode.GetPubSub(),
        vpnManager,
    )
    if err != nil {
        vpnManager.Close()
        return fmt.Errorf("failed to create VPN discovery: %w", err)
    }

    e.vpnManager = vpnManager
    e.vpnDiscovery = discovery

    return nil
}
