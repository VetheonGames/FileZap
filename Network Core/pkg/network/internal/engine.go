package internal

import (
    "context"
    "fmt"
    "sync"

    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/peer"
    ma "github.com/multiformats/go-multiaddr"

    "github.com/VetheonGames/FileZap/NetworkCore/pkg/vpn"
    "github.com/VetheonGames/FileZap/NetworkCore/pkg/network/api"
)

// NetworkEngine represents the core network engine
type networkEngine struct {
    ctx           context.Context
    cancel        context.CancelFunc
    transportNode *networkNode
    metadataNode  *networkNode
    gossipMgr     GossipManager
    quorum        QuorumManager
    validator     ChunkValidator
    manifests     ManifestManager
    chunkStore    ChunkStore
    vpnManager    *vpn.VPNManager
    vpnDiscovery  *vpn.Discovery
    mu            sync.RWMutex
}

// NewNetworkEngine creates a new network engine
func NewNetworkEngine(ctx context.Context, cfg *NetworkConfig) (NetworkEngine, error) {
    ctx, cancel := context.WithCancel(ctx)

    engine := &networkEngine{
        ctx:    ctx,
        cancel: cancel,
    }

    var err error

    // Create transport network
    engine.transportNode, err = newNetworkNode(ctx, &cfg.Transport)
    if err != nil {
        cancel()
        return nil, fmt.Errorf("failed to create transport node: %w", err)
    }

    // Create metadata network
    metadataCfg := cfg.Transport
    metadataCfg.EnableQUIC = false
    engine.metadataNode, err = newNetworkNode(ctx, &metadataCfg)
    if err != nil {
        engine.transportNode.Close()
        cancel()
        return nil, fmt.Errorf("failed to create metadata node: %w", err)
    }

    // Initialize components
    if err := engine.initializeComponents(); err != nil {
        engine.Close()
        return nil, fmt.Errorf("failed to initialize components: %w", err)
    }

    // Initialize VPN if enabled
    if cfg.VPN != nil && cfg.VPN.Enabled {
        if err := engine.initVPN(ctx, engine.transportNode.GetHost(), cfg.VPN); err != nil {
            engine.Close()
            return nil, fmt.Errorf("failed to initialize VPN: %w", err)
        }
    }

    return engine, nil
}

func (e *networkEngine) initializeComponents() error {
    var err error

    // Initialize gossip manager
    e.gossipMgr, err = newGossipManager(e.ctx, e.transportNode.GetHost(), e.transportNode.GetPubSub())
    if err != nil {
        return fmt.Errorf("failed to create gossip manager: %w", err)
    }

    // Initialize quorum manager
    e.quorum, err = newQuorumManager(e.ctx, e.transportNode.GetHost(), e.transportNode.GetPubSub())
    if err != nil {
        return fmt.Errorf("failed to create quorum manager: %w", err)
    }

    // Initialize chunk store
    e.chunkStore = newChunkStore(e.ctx, e.transportNode.GetHost())

    // Initialize chunk validator
    e.validator = newChunkValidator(
        e.ctx,
        e.quorum.(*quorumManager),
        e.chunkStore.(*chunkStore),
    )

    // Initialize manifest manager
    e.manifests, err = newManifestManager(
        e.ctx,
        e.metadataNode.GetHost().ID(),
        e.metadataNode.GetDHT(),
        e.metadataNode.GetPubSub(),
    )
    if err != nil {
        return fmt.Errorf("failed to create manifest manager: %w", err)
    }

    return nil
}

// Implement NetworkEngine interface

func (e *networkEngine) Connect(addr ma.Multiaddr) error {
    info, err := peer.AddrInfoFromP2pAddr(addr)
    if err != nil {
        return fmt.Errorf("invalid peer address: %w", err)
    }
    return e.transportNode.Connect(*info)
}

func (e *networkEngine) Close() error {
    e.cancel()

    var errs []error

    if e.vpnManager != nil {
        if err := e.vpnManager.Close(); err != nil {
            errs = append(errs, fmt.Errorf("failed to close VPN: %w", err))
        }
    }

    if err := e.transportNode.Close(); err != nil {
        errs = append(errs, fmt.Errorf("failed to close transport: %w", err))
    }

    if err := e.metadataNode.Close(); err != nil {
        errs = append(errs, fmt.Errorf("failed to close metadata: %w", err))
    }

    if len(errs) > 0 {
        return fmt.Errorf("errors during shutdown: %v", errs)
    }
    return nil
}

func (e *networkEngine) GetNodeID() string {
    return e.transportNode.GetHost().ID().String()
}

func (e *networkEngine) GetTransportHost() host.Host {
    return e.transportNode.GetHost()
}

func (e *networkEngine) GetMetadataHost() host.Host {
    return e.metadataNode.GetHost()
}

func (e *networkEngine) GetPeers() []peer.ID {
    return e.transportNode.GetHost().Network().Peers()
}

func (e *networkEngine) GetVPNManager() *vpn.VPNManager {
    return e.vpnManager
}

func (e *networkEngine) GetVPNStatus() *api.VPNStatus {
    if e.vpnManager == nil {
        return &api.VPNStatus{Connected: false}
    }

    peers := e.vpnManager.GetPeers()
    activePeers := make([]api.VPNPeer, len(peers))
    for i, p := range peers {
        activePeers[i] = api.VPNPeer{
            ID: p.String(),
            IP: "", // Will be populated by the VPN manager
        }
    }

    return &api.VPNStatus{
        Connected:   true,
        PeerCount:   len(peers),
        ActivePeers: activePeers,
    }
}

func (e *networkEngine) Bootstrap(addrs []peer.AddrInfo) error {
    for _, addr := range addrs {
        if err := e.transportNode.Connect(addr); err != nil {
            return fmt.Errorf("failed to connect transport to bootstrap peer: %w", err)
        }
        if err := e.metadataNode.Connect(addr); err != nil {
            return fmt.Errorf("failed to connect metadata to bootstrap peer: %w", err)
        }
    }
    return nil
}

// File operations

func (e *networkEngine) AddZapFile(manifest *ManifestInfo, chunks map[string][]byte) error {
    return e.manifests.AddManifest(manifest)
}

func (e *networkEngine) GetZapFile(name string) (*ManifestInfo, map[string][]byte, error) {
    return e.manifests.GetManifest(name)
}

func (e *networkEngine) ReportBadFile(name string, reason string) error {
    manifest, _, err := e.manifests.GetManifest(name)
    if err != nil {
        return err
    }

    // Remove chunks and manifest
    for _, hash := range manifest.ChunkHashes {
        e.chunkStore.Remove(hash)
    }
    e.manifests.RemoveManifest(name)

    return nil
}

// Storage operations

func (e *networkEngine) RegisterStorageNode() error {
    // TODO: Implement storage node registration
    return nil
}

func (e *networkEngine) UnregisterStorageNode() error {
    // TODO: Implement storage node unregistration
    return nil
}

func (e *networkEngine) GetStorageRequest() (*StorageRequest, error) {
    return e.chunkStore.GetPendingRequest()
}

func (e *networkEngine) ValidateChunkRequest(req *StorageRequest) error {
    result := e.validator.ValidateChunk(req.Data, req.ChunkHash, req.Owner)
    if result != ValidationSuccess {
        return fmt.Errorf("chunk validation failed: %v", result)
    }
    return nil
}

func (e *networkEngine) StoreChunk(req *StorageRequest) error {
    if !e.chunkStore.Store(req.ChunkHash, req.Data) {
        return fmt.Errorf("failed to store chunk")
    }
    return nil
}

func (e *networkEngine) RejectStorageRequest(req *StorageRequest, reason string) error {
    return e.gossipMgr.NotifyStorageRejection(req, reason)
}

func (e *networkEngine) AcknowledgeStorage(req *StorageRequest) error {
    return e.gossipMgr.NotifyStorageSuccess(req)
}
