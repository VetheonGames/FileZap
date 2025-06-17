package network

import (
    "context"
    "fmt"
    "time"

    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/peer"
    ma "github.com/multiformats/go-multiaddr"
)

// NetworkConfig holds configuration for the network engine
type NetworkConfig struct {
    Transport     *TransportConfig
    MetadataStore string
    ChunkCacheDir string
}

// DefaultNetworkConfig returns default network settings
func DefaultNetworkConfig() *NetworkConfig {
    return &NetworkConfig{
        Transport: DefaultTransportConfig(),
        MetadataStore: "metadata",
        ChunkCacheDir: "chunks",
    }
}

// NewNetworkEngine creates a new network engine
func NewNetworkEngine(ctx context.Context, cfg *NetworkConfig) (*NetworkEngine, error) {
    ctx, cancel := context.WithCancel(ctx)

    // Create transport network (QUIC-enabled)
    transportNode, err := NewTransportNode(ctx, cfg.Transport)
    if err != nil {
        cancel()
        return nil, fmt.Errorf("failed to create transport node: %w", err)
    }

    // Create metadata network (TCP-based)
    metadataCfg := *cfg.Transport
    metadataCfg.EnableQUIC = false
    metadataNode, err := NewTransportNode(ctx, &metadataCfg)
    if err != nil {
        transportNode.Close()
        cancel()
        return nil, fmt.Errorf("failed to create metadata node: %w", err)
    }

    // Initialize gossip manager for peer discovery
    gossipMgr, err := NewGossipManager(ctx, transportNode.host, transportNode.pubsub)
    if err != nil {
        transportNode.Close()
        metadataNode.Close()
        cancel()
        return nil, fmt.Errorf("failed to create gossip manager: %w", err)
    }

    // Initialize manifest manager
    manifests := NewManifestManager(ctx, metadataNode.host.ID(), metadataNode.dht, metadataNode.pubsub)

    // Initialize chunk store
    chunkStore := NewChunkStore(transportNode.host)

    // Initialize quorum manager
    quorum, err := NewQuorumManager(ctx, transportNode.host, transportNode.pubsub, gossipMgr)
    if err != nil {
        transportNode.Close()
        metadataNode.Close()
        cancel()
        return nil, fmt.Errorf("failed to create quorum manager: %w", err)
    }

    // Initialize chunk validator
    validator := NewChunkValidator(ctx, quorum, chunkStore)

    engine := &NetworkEngine{
        ctx:           ctx,
        cancel:        cancel,
        transportNode: transportNode,
        metadataNode:  metadataNode,
        gossipMgr:     gossipMgr,
        quorum:        quorum,
        validator:     validator,
        manifests:     manifests,
        chunkStore:    chunkStore,
    }

    // Start monitoring network health
    go engine.monitorNetwork()

    return engine, nil
}

// Connect connects to a peer on both transport and metadata networks
func (e *NetworkEngine) Connect(addr ma.Multiaddr) error {
    peerInfo, err := peer.AddrInfoFromP2pAddr(addr)
    if err != nil {
        return fmt.Errorf("invalid peer address: %w", err)
    }

    // Connect on transport network
    if err := e.transportNode.host.Connect(e.ctx, *peerInfo); err != nil {
        return fmt.Errorf("failed to connect transport: %w", err)
    }

    // Connect on metadata network
    if err := e.metadataNode.host.Connect(e.ctx, *peerInfo); err != nil {
        return fmt.Errorf("failed to connect metadata: %w", err)
    }

    // Update gossip manager
    e.gossipMgr.updatePeerInfo(&PeerGossipInfo{
        ID:        peerInfo.ID,
        LastSeen:  time.Now(),
    })

    return nil
}

// AddZapFile adds a new .zap file to the network
func (e *NetworkEngine) AddZapFile(manifest *ManifestInfo, chunks map[string][]byte) error {
    // Validate all chunks before storing
    for hash, data := range chunks {
        if result := e.validator.ValidateChunk(data, hash, e.transportNode.host.ID()); result != ValidationSuccess {
            return fmt.Errorf("chunk validation failed: %v", result)
        }
        e.chunkStore.Store(hash, data)
    }

    // Store and replicate manifest
    return e.manifests.AddManifest(manifest)
}

// GetZapFile retrieves a .zap file from the network
func (e *NetworkEngine) GetZapFile(name string) (*ManifestInfo, map[string][]byte, error) {
    // Get manifest from DHT
    manifest, err := e.manifests.GetManifest(name)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to get manifest: %w", err)
    }

    // Download and validate chunks
    chunks := make(map[string][]byte)
    for _, hash := range manifest.ChunkHashes {
        // Check if we own this chunk locally first
        if data, exists := e.chunkStore.Get(hash); exists {
            // Validate even local chunks
            if result := e.validator.ValidateChunk(data, hash, e.transportNode.host.ID()); result == ValidationSuccess {
                chunks[hash] = data
                continue
            }
        }

        // Download from remote peer
        data, err := e.chunkStore.transfers.Download(manifest.Owner, hash)
        if err != nil {
            return nil, nil, fmt.Errorf("failed to download chunk %s: %w", hash, err)
        }

        // Validate downloaded chunk
        if result := e.validator.ValidateChunk(data, hash, manifest.Owner); result != ValidationSuccess {
            return nil, nil, fmt.Errorf("downloaded chunk validation failed: %v", result)
        }

        chunks[hash] = data
    }

    return manifest, chunks, nil
}

// ReportBadFile reports a malicious file to the network
func (e *NetworkEngine) ReportBadFile(name string, reason string) error {
    manifest, err := e.manifests.GetManifest(name)
    if err != nil {
        return fmt.Errorf("failed to get manifest: %w", err)
    }

    // Propose vote to remove the file
    if err := e.quorum.ProposeVote(VoteRemoveFile, name, reason, nil); err != nil {
        return fmt.Errorf("failed to propose file removal: %w", err)
    }

    // Update reputation of the file owner
    e.quorum.UpdatePeerReputation(manifest.Owner, -20)

    return nil
}

// monitorNetwork monitors network health and peer behavior
func (e *NetworkEngine) monitorNetwork() {
    ticker := time.NewTicker(time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-e.ctx.Done():
            return
        case <-ticker.C:
            e.checkPeerHealth()
        case id := <-e.quorum.peerBanned:
            e.handleBannedPeer(id)
        case name := <-e.quorum.fileRemoved:
            e.handleRemovedFile(name)
        }
    }
}

// checkPeerHealth checks the health of connected peers
func (e *NetworkEngine) checkPeerHealth() {
    peers := e.gossipMgr.GetPeers()
    for _, info := range peers {
        // Check if peer is responsive
        if time.Since(info.LastSeen) > time.Hour {
            e.quorum.UpdatePeerReputation(info.ID, -1)
        }

        // Check peer's uptime
        if info.Uptime < 50 {
            e.quorum.UpdatePeerReputation(info.ID, -1)
        }

        // Check response time
        if info.ResponseTime > 1000 { // >1s average response
            e.quorum.UpdatePeerReputation(info.ID, -1)
        }
    }
}

// handleBannedPeer handles cleanup when a peer is banned
func (e *NetworkEngine) handleBannedPeer(id peer.ID) {
    // Disconnect from peer
    if err := e.transportNode.host.Network().ClosePeer(id); err != nil {
        return
    }
    if err := e.metadataNode.host.Network().ClosePeer(id); err != nil {
        return
    }

    // Remove stored chunks from banned peer
    // TODO: Implement chunk cleanup
}

// handleRemovedFile handles cleanup when a file is removed
func (e *NetworkEngine) handleRemovedFile(name string) {
    // Remove manifest
    if manifest, err := e.manifests.GetManifest(name); err == nil {
        // Remove associated chunks
        for _, hash := range manifest.ChunkHashes {
            e.chunkStore.Remove(hash)
        }
    }
}

// GetNodeID returns this node's ID
func (e *NetworkEngine) GetNodeID() string {
    return e.transportNode.host.ID().String()
}

// GetPeers returns a list of connected peers
func (e *NetworkEngine) GetPeers() []peer.ID {
    return e.transportNode.host.Network().Peers()
}

// GetStoredChunks returns all chunks stored by this node
func (e *NetworkEngine) GetStoredChunks() [][]byte {
    chunks := make([][]byte, 0)
    for _, data := range e.chunkStore.chunks {
        chunks = append(chunks, data)
    }
    return chunks
}

// GetRequestCount returns the number of storage requests handled
func (e *NetworkEngine) GetRequestCount() int {
    // TODO: Implement proper request counting
    return len(e.chunkStore.chunks)
}

// GetTransportHost returns the transport network host
func (e *NetworkEngine) GetTransportHost() host.Host {
    return e.transportNode.host
}

// GetMetadataHost returns the metadata network host
func (e *NetworkEngine) GetMetadataHost() host.Host {
    return e.metadataNode.host
}

// Bootstrap connects to initial peers to join the network
func (e *NetworkEngine) Bootstrap(addrs []peer.AddrInfo) error {
    for _, addr := range addrs {
        if err := e.transportNode.host.Connect(e.ctx, addr); err != nil {
            return fmt.Errorf("failed to connect to bootstrap peer: %w", err)
        }
        if err := e.metadataNode.host.Connect(e.ctx, addr); err != nil {
            return fmt.Errorf("failed to connect metadata to bootstrap peer: %w", err)
        }
    }
    return nil
}

// RegisterStorageNode registers this node as a storage provider
func (e *NetworkEngine) RegisterStorageNode() error {
    // Create storage node info
    nodeInfo := &StorageNodeInfo{
        ID:             e.transportNode.host.ID().String(),
        AvailableSpace: maxTotalSize - int64(e.chunkStore.totalSize),
        Uptime:        100.0, // Start with perfect uptime
        LastSeen:      time.Now(),
        ChunksStored:  make([]string, 0),
    }

    // Announce to the network via gossip
    return e.gossipMgr.AnnounceStorageNode(nodeInfo)
}

// UnregisterStorageNode removes this node from the storage provider list
func (e *NetworkEngine) UnregisterStorageNode() error {
    return e.gossipMgr.RemoveStorageNode(e.transportNode.host.ID().String())
}

// ValidateChunkRequest validates if a chunk can be stored
func (e *NetworkEngine) ValidateChunkRequest(req *StorageRequest) error {
    // Check available space
    if int64(e.chunkStore.totalSize)+req.Size > maxTotalSize {
        return fmt.Errorf("insufficient storage space")
    }
    return nil
}

// StoreChunk stores a chunk from a storage request
func (e *NetworkEngine) StoreChunk(req *StorageRequest) error {
    if !e.chunkStore.Store(req.ChunkHash, req.Data) {
        return fmt.Errorf("failed to store chunk")
    }
    return nil
}

// RejectStorageRequest rejects a chunk storage request
func (e *NetworkEngine) RejectStorageRequest(req *StorageRequest, reason string) error {
    return e.gossipMgr.NotifyStorageRejection(req, reason)
}

// AcknowledgeStorage confirms successful chunk storage
func (e *NetworkEngine) AcknowledgeStorage(req *StorageRequest) error {
    return e.gossipMgr.NotifyStorageSuccess(req)
}

// GetStorageRequest gets a pending chunk storage request
func (e *NetworkEngine) GetStorageRequest() (*StorageRequest, error) {
    return e.chunkStore.GetPendingRequest()
}

// ReportBadPeer reports a malicious peer to the network
func (e *NetworkEngine) ReportBadPeer(peerID peer.ID, reason string) error {
    // Propose vote to remove peer
    if err := e.quorum.ProposeVote(VoteRemovePeer, string(peerID), reason, nil); err != nil {
        return fmt.Errorf("failed to propose peer removal: %w", err)
    }

    // Update peer's reputation
    e.quorum.UpdatePeerReputation(peerID, -50)

    // Disconnect from peer
    if err := e.transportNode.host.Network().ClosePeer(peerID); err != nil {
        return fmt.Errorf("failed to disconnect from peer: %w", err)
    }

    return nil
}

// Close shuts down the network engine
func (e *NetworkEngine) Close() error {
    e.cancel()

    var errs []error
    if err := e.transportNode.Close(); err != nil {
        errs = append(errs, fmt.Errorf("failed to close transport node: %w", err))
    }
    if err := e.metadataNode.Close(); err != nil {
        errs = append(errs, fmt.Errorf("failed to close metadata node: %w", err))
    }

    if len(errs) > 0 {
        return fmt.Errorf("errors during shutdown: %v", errs)
    }
    return nil
}
