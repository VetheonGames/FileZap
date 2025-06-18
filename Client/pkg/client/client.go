package client

import (
    "context"
    "fmt"

    "github.com/libp2p/go-libp2p/core/peer"
    ma "github.com/multiformats/go-multiaddr"

    "github.com/VetheonGames/FileZap/NetworkCore/pkg/network"
    "github.com/VetheonGames/FileZap/NetworkCore/pkg/vpn"
)

// Client manages the FileZap client operations
type Client struct {
    ctx        context.Context
    cancel     context.CancelFunc
    engine     *network.NetworkEngine
    vpnManager *vpn.VPNManager
    config     *Config
}

// Config holds the client configuration
type Config struct {
    NetworkCIDR   string
    StorageDir    string
    MetadataDir   string
    ListenPort    int
    EnableVPN     bool
    VPNConfig     *VPNConfig
}

// DefaultConfig returns default client settings
func DefaultConfig() *Config {
    return &Config{
        NetworkCIDR:   "10.42.0.0/16",
        StorageDir:    "storage",
        MetadataDir:   "metadata",
        ListenPort:    6001,
        EnableVPN:     false,
        VPNConfig:     DefaultVPNConfig(),
    }
}

// NewClient creates a new FileZap client
func NewClient(ctx context.Context, cfg *Config) (*Client, error) {
    ctx, cancel := context.WithCancel(ctx)

    // Create network engine config
    engineCfg := network.DefaultNetworkConfig()
    engineCfg.ChunkCacheDir = cfg.StorageDir
    engineCfg.MetadataStore = cfg.MetadataDir
    engineCfg.Transport.ListenPort = cfg.ListenPort

    // Configure VPN if enabled
    if cfg.EnableVPN {
        engineCfg.VPN = &network.VPNConfig{
            Enabled:       true,
            NetworkCIDR:   cfg.VPNConfig.NetworkCIDR,
            InterfaceName: cfg.VPNConfig.InterfaceName,
            NetworkKey:    cfg.VPNConfig.NetworkKey,
        }
    }

    // Create network engine
    engine, err := network.NewNetworkEngine(ctx, engineCfg)
    if err != nil {
        cancel()
        return nil, fmt.Errorf("failed to create network engine: %w", err)
    }

    client := &Client{
        ctx:        ctx,
        cancel:     cancel,
        engine:     engine,
        config:     cfg,
    }

    // Store VPN manager reference if enabled
    if cfg.EnableVPN {
        if vpnManager := engine.GetVPNManager(); vpnManager != nil {
            client.vpnManager = vpnManager
        }
    }

    return client, nil
}

// Connect connects to a FileZap peer
func (c *Client) Connect(addr ma.Multiaddr) error {
    return c.engine.Connect(addr)
}

// Bootstrap connects to initial peers
func (c *Client) Bootstrap(addrs []string) error {
    peerInfos := make([]peer.AddrInfo, 0, len(addrs))
    
    for _, addr := range addrs {
        maddr, err := ma.NewMultiaddr(addr)
        if err != nil {
            return fmt.Errorf("invalid multiaddr %s: %w", addr, err)
        }
        
        peerInfo, err := peer.AddrInfoFromP2pAddr(maddr)
        if err != nil {
            return fmt.Errorf("invalid peer address %s: %w", addr, err)
        }
        
        peerInfos = append(peerInfos, *peerInfo)
    }

    return c.engine.Bootstrap(peerInfos)
}

// Close shuts down the client
func (c *Client) Close() error {
    c.cancel()
    return c.engine.Close()
}

// GetLocalPeerID returns the local peer ID
func (c *Client) GetLocalPeerID() string {
    return c.engine.GetNodeID()
}

// GetConnectedPeers returns a list of connected peers
func (c *Client) GetConnectedPeers() []peer.ID {
    return c.engine.GetPeers()
}

// GetConfig returns the current client configuration
func (c *Client) GetConfig() *Config {
    return c.config
}
