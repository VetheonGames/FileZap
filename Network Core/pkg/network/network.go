package network

import (
    "context"
    "fmt"

    "github.com/libp2p/go-libp2p/core/peer"
    "github.com/libp2p/go-libp2p/core/host"
    dht "github.com/libp2p/go-libp2p-kad-dht"
    pubsub "github.com/libp2p/go-libp2p-pubsub"
    ma "github.com/multiformats/go-multiaddr"

    "github.com/VetheonGames/FileZap/NetworkCore/pkg/vpn"
)

// networkImpl implements the Network interface using internal components
type networkImpl struct {
    transportHost host.Host
    metadataHost host.Host
    dht          *dht.IpfsDHT
    pubsub       *pubsub.PubSub
    engine       *NetworkEngine
}

// NewNetwork creates a new Network implementation
func NewNetwork(ctx context.Context, cfg *NetworkConfig) (Network, error) {
    engine, err := NewNetworkEngine(ctx, cfg)
    if err != nil {
        return nil, fmt.Errorf("failed to create network engine: %w", err)
    }

    return &networkImpl{
        transportHost: engine.transportHost,
        metadataHost: engine.metadataHost,
        dht:          engine.dht,
        pubsub:       engine.pubsub,
        engine:       engine,
    }, nil
}

// Implement Network interface

func (n *networkImpl) Connect(addr ma.Multiaddr) error {
    pinfo, err := peer.AddrInfoFromP2pAddr(addr)
    if err != nil {
        return fmt.Errorf("invalid peer address: %w", err)
    }
    return n.transportHost.Connect(n.engine.ctx, *pinfo)
}

func (n *networkImpl) Close() error {
    return n.engine.Close()
}

func (n *networkImpl) GetNodeID() string {
    return n.engine.GetNodeID().String()
}

func (n *networkImpl) GetTransportHost() host.Host {
    return n.engine.GetTransportHost()
}

func (n *networkImpl) GetMetadataHost() host.Host {
    return n.engine.GetMetadataHost()
}

func (n *networkImpl) GetVPNManager() *vpn.VPNManager {
    return n.engine.GetVPNManager()
}

func (n *networkImpl) Bootstrap(addrs []peer.AddrInfo) error {
    return n.engine.dht.Bootstrap(n.engine.ctx)
}

func (n *networkImpl) GetPeers() []peer.ID {
    return n.engine.gossipMgr.GetPeers()
}

func (n *networkImpl) GetVPNStatus() *VPNStatus {
    return n.engine.GetVPNStatus()
}

// File operations

func (n *networkImpl) AddZapFile(manifest *ManifestInfo, chunks map[string][]byte) error {
    return n.engine.AddZapFile(manifest, chunks)
}

func (n *networkImpl) GetZapFile(name string) (*ManifestInfo, map[string][]byte, error) {
    return n.engine.GetZapFile(name)
}

func (n *networkImpl) ReportBadFile(name string, reason string) error {
    return n.engine.ReportBadFile(name, reason)
}

// Storage operations

func (n *networkImpl) RegisterStorageNode() error {
    return n.engine.RegisterStorageNode()
}

func (n *networkImpl) UnregisterStorageNode() error {
    return n.engine.UnregisterStorageNode()
}

func (n *networkImpl) GetStorageRequest() (*StorageRequest, error) {
    return n.engine.GetStorageRequest()
}

func (n *networkImpl) ValidateChunkRequest(req *StorageRequest) error {
    return n.engine.ValidateChunkRequest(req)
}

func (n *networkImpl) StoreChunk(req *StorageRequest) error {
    return n.engine.StoreChunk(req)
}

func (n *networkImpl) RejectStorageRequest(req *StorageRequest, reason string) error {
    return n.engine.RejectStorageRequest(req, reason)
}

func (n *networkImpl) AcknowledgeStorage(req *StorageRequest) error {
    return n.engine.AcknowledgeStorage(req)
}
