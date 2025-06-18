package impl

import (
	"github.com/VetheonGames/FileZap/NetworkCore/pkg/network/api"
	"github.com/VetheonGames/FileZap/NetworkCore/pkg/network/internal"
)

// ConvertConfig converts API config to internal config
func ConvertConfig(cfg *api.NetworkConfig) *internal.NetworkConfig {
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

// ConvertVPNStatus converts internal VPN status to API VPN status
func ConvertVPNStatus(status *internal.VPNStatus) *api.VPNStatus {
    if status == nil {
        return &api.VPNStatus{Connected: false}
    }
    peers := make([]api.VPNPeer, len(status.ActivePeers))
    for i, p := range status.ActivePeers {
        peers[i] = api.VPNPeer{
            ID:       p,
            IP:       "", // Will be populated by the VPN manager
            LastSeen: 0,  // Will be populated by the VPN manager
        }
    }
    return &api.VPNStatus{
        Connected:   status.Connected,
        LocalIP:     status.LocalIP,
        PeerCount:   status.PeerCount,
        ActivePeers: peers,
    }
}
