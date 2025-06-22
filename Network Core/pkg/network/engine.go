package network

import (
    "context"
    "fmt"
    "time"

    "github.com/VetheonGames/FileZap/NetworkCore/pkg/vpn"
    "github.com/ipfs/go-cid"
    "github.com/libp2p/go-libp2p"
    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/peer"
    pubsub "github.com/libp2p/go-libp2p-pubsub"
    dht "github.com/libp2p/go-libp2p-kad-dht"
)

var (
    // VPNProviderKey is used to identify VPN service providers in the DHT
    VPNProviderKey = cid.NewCidV1(cid.Raw, []byte("vpn-provider"))
)

// NetworkEngine manages network operations
type NetworkEngine struct {
    ctx           context.Context
    cancel        context.CancelFunc
    startTime     time.Time
    config        *NetworkConfig
    transportHost host.Host
    metadataHost  host.Host
    nodeID        peer.ID
    gossipMgr     GossipManager
    quorum        QuorumManager
    validator     *ChunkValidator
    manifests     ManifestManager
    chunkStore    *ChunkStore
    vpnManager    *vpn.VPNManager
    dht           *dht.IpfsDHT
    pubsub        *pubsub.PubSub
}

// NewNetworkEngine creates a new network engine instance
func NewNetworkEngine(ctx context.Context, cfg *NetworkConfig) (*NetworkEngine, error) {
    // Create the transport host
    transportHost, err := libp2p.New(
        libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", cfg.Transport.ListenPort)),
        libp2p.DisableRelay(),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create transport host: %v", err)
    }

    // Create the metadata host (using a different port)
    metadataHost, err := libp2p.New(
        libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", cfg.Transport.ListenPort+1)),
        libp2p.DisableRelay(),
    )
    if err != nil {
        transportHost.Close()
        return nil, fmt.Errorf("failed to create metadata host: %v", err)
    }

    ctx, cancel := context.WithCancel(ctx)
    engine := &NetworkEngine{
        ctx:          ctx,
        cancel:       cancel,
        startTime:    time.Now(),
        config:       cfg,
        transportHost: transportHost,
        metadataHost: metadataHost,
        nodeID:       transportHost.ID(),
    }

    return engine, nil
}

// GetNodeID returns the node's peer ID
func (e *NetworkEngine) GetNodeID() peer.ID {
    return e.nodeID
}

// GetTransportHost returns the transport layer host
func (e *NetworkEngine) GetTransportHost() host.Host {
    return e.transportHost
}

// GetMetadataHost returns the metadata layer host
func (e *NetworkEngine) GetMetadataHost() host.Host {
    return e.metadataHost
}

// Close shuts down the network engine
func (e *NetworkEngine) Close() error {
    if err := e.transportHost.Close(); err != nil {
        return fmt.Errorf("failed to close transport host: %v", err)
    }
    if err := e.metadataHost.Close(); err != nil {
        return fmt.Errorf("failed to close metadata host: %v", err)
    }
    if e.vpnManager != nil {
        e.vpnManager.Close()
    }
    e.cancel()
    return nil
}

// initVPN initializes VPN functionality
func (e *NetworkEngine) initVPN(ctx context.Context, h host.Host, cfg *VPNConfig) error {
    vpnConfig := &vpn.Config{
        NetworkCIDR:   cfg.NetworkCIDR,
        InterfaceName: cfg.InterfaceName,
        MTU:          vpn.DefaultMTU,
    }

    var err error
    e.vpnManager, err = vpn.NewVPNManager(ctx, h, vpnConfig)
    if err != nil {
        return fmt.Errorf("failed to create VPN manager: %w", err)
    }

    return nil
}

// GetVPNManager returns the VPN manager if enabled
func (e *NetworkEngine) GetVPNManager() *vpn.VPNManager {
    return e.vpnManager
}

// GetVPNStatus returns the current VPN status
func (e *NetworkEngine) GetVPNStatus() *VPNStatus {
    if e.vpnManager == nil {
        return &VPNStatus{Connected: false}
    }

    peers := e.vpnManager.GetPeers()
    activePeers := make([]peer.ID, len(peers))
    copy(activePeers, peers)

    return &VPNStatus{
        Connected:   len(activePeers) > 0,
        LocalIP:     e.vpnManager.GetLocalIP(),
        PeerCount:   len(activePeers),
        ActivePeers: activePeers,
    }
}

// File operations
func (e *NetworkEngine) AddZapFile(manifest *ManifestInfo, chunks map[string][]byte) error {
    if err := e.manifests.AddManifest(manifest); err != nil {
        return fmt.Errorf("failed to add manifest: %w", err)
    }

    for hash, data := range chunks {
        if !e.chunkStore.Store(hash, data) {
            return fmt.Errorf("failed to store chunk %s", hash)
        }
    }

    return nil
}

func (e *NetworkEngine) GetZapFile(name string) (*ManifestInfo, map[string][]byte, error) {
    manifest, err := e.manifests.GetManifest(name)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to get manifest: %w", err)
    }

    chunks := make(map[string][]byte)
    for _, hash := range manifest.ChunkHashes {
        data, ok := e.chunkStore.Get(hash)
        if !ok {
            return nil, nil, fmt.Errorf("chunk not found: %s", hash)
        }
        chunks[hash] = data
    }

    return manifest, chunks, nil
}

func (e *NetworkEngine) ReportBadFile(name string, reason string) error {
    return e.quorum.StartVote(VoteRemoveFile, name, e.transportHost.ID())
}

// Storage operations
func (e *NetworkEngine) RegisterStorageNode() error {
    info := &StorageNodeInfo{
        ID:             e.transportHost.ID().String(),
        AvailableSpace: maxStorageSize,
        TotalSpace:     maxStorageSize,
        Uptime:         100.0, // TODO: Calculate actual uptime
        Version:        "0.1.0",
        Location:       "", // TODO: Add location support
    }
    return e.gossipMgr.AnnounceStorageNode(info)
}

func (e *NetworkEngine) UnregisterStorageNode() error {
    nodeID := e.transportHost.ID()
    return e.gossipMgr.RemoveStorageNode(nodeID.String())
}

func (e *NetworkEngine) GetStorageRequest() (*StorageRequest, error) {
    return e.chunkStore.GetPendingRequest()
}

func (e *NetworkEngine) ValidateChunkRequest(req *StorageRequest) error {
    result := e.validator.ValidateChunk(req.Data, req.ChunkHash, peer.ID(req.Owner))
    if result != ValidationSuccess {
        return ErrInvalidChunk
    }
    return nil
}

func (e *NetworkEngine) StoreChunk(req *StorageRequest) error {
    if !e.chunkStore.Store(req.ChunkHash, req.Data) {
        return ErrStorageFull
    }
    return nil
}

func (e *NetworkEngine) RejectStorageRequest(req *StorageRequest, reason string) error {
    return e.gossipMgr.NotifyStorageRejection(req, reason)
}

func (e *NetworkEngine) AcknowledgeStorage(req *StorageRequest) error {
    return e.gossipMgr.NotifyStorageSuccess(req)
}
