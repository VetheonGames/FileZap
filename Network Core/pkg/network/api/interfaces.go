package api

import (
    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/peer"
    ma "github.com/multiformats/go-multiaddr"

    "github.com/VetheonGames/FileZap/NetworkCore/pkg/vpn"
)

// Network defines the main interface for network operations
type Network interface {
    // Core operations
    Connect(addr ma.Multiaddr) error
    Close() error
    Bootstrap(addrs []peer.AddrInfo) error
    
    // Identity and peer operations
    GetNodeID() string
    GetPeers() []peer.ID
    GetTransportHost() host.Host
    GetMetadataHost() host.Host

    // File operations
    AddZapFile(manifest *ManifestInfo, chunks map[string][]byte) error
    GetZapFile(name string) (*ManifestInfo, map[string][]byte, error)
    ReportBadFile(name string, reason string) error

    // Storage operations
    RegisterStorageNode() error
    UnregisterStorageNode() error
    GetStorageRequest() (*StorageRequest, error)
    ValidateChunkRequest(req *StorageRequest) error
    StoreChunk(req *StorageRequest) error
    RejectStorageRequest(req *StorageRequest, reason string) error
    AcknowledgeStorage(req *StorageRequest) error

    // VPN operations
    GetVPNManager() *vpn.VPNManager
    GetVPNStatus() *VPNStatus
}

// VPNStatus represents the current state of the VPN connection
type VPNStatus struct {
    Connected   bool
    LocalIP     string
    PeerCount   int
    ActivePeers []VPNPeer
}

// VPNPeer represents a connected peer in the VPN
type VPNPeer struct {
    ID       string
    IP       string
    LastSeen int64
}

// NetworkBuilder defines interface for creating Network instances
type NetworkBuilder interface {
    WithConfig(cfg *NetworkConfig) NetworkBuilder
    WithVPNSupport() NetworkBuilder
    Build() (Network, error)
}

// NewNetworkBuilder creates a new NetworkBuilder instance
func NewNetworkBuilder() NetworkBuilder {
    return &networkBuilder{}
}

type networkBuilder struct {
    config *NetworkConfig
    vpn    bool
}

func (b *networkBuilder) WithConfig(cfg *NetworkConfig) NetworkBuilder {
    b.config = cfg
    return b
}

func (b *networkBuilder) WithVPNSupport() NetworkBuilder {
    b.vpn = true
    return b
}

func (b *networkBuilder) Build() (Network, error) {
    // Actual implementation is in the internal package
    return newNetwork(b.config, b.vpn)
}

// Factory function implemented in internal package
func newNetwork(cfg *NetworkConfig, enableVPN bool) (Network, error)
