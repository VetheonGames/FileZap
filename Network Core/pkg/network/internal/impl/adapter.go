package impl

import (
    "github.com/VetheonGames/FileZap/NetworkCore/pkg/network/api"
    "github.com/VetheonGames/FileZap/NetworkCore/pkg/network/internal"
)

// Convert API config to internal config
func convertConfig(cfg *api.NetworkConfig) *internal.NetworkConfig {
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

    if cfg.VPNConfig != nil {
        internalCfg.VPN = &internal.VPNConfig{
            Enabled:       cfg.VPNConfig.Enabled,
            NetworkCIDR:   cfg.VPNConfig.NetworkCIDR,
            InterfaceName: cfg.VPNConfig.InterfaceName,
            NetworkKey:    cfg.VPNConfig.NetworkKey,
        }
    }

    return internalCfg
}

// Convert internal VPN status to API VPN status
func convertVPNStatus(status *internal.VPNStatus) *api.VPNStatus {
    if status == nil {
        return &api.VPNStatus{Connected: false}
    }
    return &api.VPNStatus{
        Connected:   status.Connected,
        LocalIP:     status.LocalIP,
        PeerCount:   status.PeerCount,
        ActivePeers: status.ActivePeers,
    }
}
