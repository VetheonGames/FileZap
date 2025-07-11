package network

import (
    "context"
    "encoding/json"
    "sync"
    "time"

    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/peer"
    pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// GossipManager handles peer discovery and network announcements
type GossipManager interface {
    Start() error
    Stop() error
    Broadcast(topic string, data []byte) error
    AnnounceStorageNode(info *StorageNodeInfo) error
    RemoveStorageNode(nodeID string) error
    NotifyStorageSuccess(req *StorageRequest) error
    NotifyStorageRejection(req *StorageRequest, reason string) error
    GetPeers() []peer.ID
}

const (
    // Protocol IDs
    GossipProtocolID         = "/filezap/gossip/1.0.0"
    PeerDiscoveryTopic      = "filezap-peers"
    GossipInterval          = 30 * time.Second
    PeerTimeoutInterval     = 5 * time.Minute
    MaxStoredPeerAddrs      = 1000
)

// PeerGossipInfo represents the information shared about peers
type PeerGossipInfo struct {
    ID            peer.ID     `json:"id"`
    Addresses     []string    `json:"addresses"`
    LastSeen      time.Time   `json:"last_seen"`
    ChunkCount    int         `json:"chunk_count"`
    Uptime        float64     `json:"uptime"`     // Uptime percentage
    ResponseTime  float64     `json:"resp_time"`  // Average response time in ms
    Version       string      `json:"version"`     // Protocol version
}

// GossipManagerImpl implements the GossipManager interface
type GossipManagerImpl struct {
    ctx           context.Context
    host          host.Host
    pubsub        *pubsub.PubSub
    topic         *pubsub.Topic
    subscription  *pubsub.Subscription
    peerStore     map[peer.ID]*PeerGossipInfo
    metrics       map[peer.ID]*PeerMetrics
    mu            sync.RWMutex
    
    // Channels for peer events
    peerDiscovered chan peer.ID
    peerLeft       chan peer.ID
    peerUpdated    chan peer.ID
}

// PeerMetrics tracks peer performance metrics
type PeerMetrics struct {
    successfulRequests uint64
    failedRequests    uint64
    totalResponseTime float64
    lastResponseTime  time.Time
    lastSeen         time.Time
    connectionStart  time.Time
}

// NewGossipManager creates a new gossip manager for peer discovery
func NewGossipManager(ctx context.Context, h host.Host, ps *pubsub.PubSub) (GossipManager, error) {
    // Create topic for peer discovery
    topic, err := ps.Join(PeerDiscoveryTopic)
    if err != nil {
        return nil, err
    }

    // Subscribe to peer updates
    subscription, err := topic.Subscribe()
    if err != nil {
        return nil, err
    }

    gm := &GossipManagerImpl{
        ctx:            ctx,
        host:           h,
        pubsub:         ps,
        topic:          topic,
        subscription:   subscription,
        peerStore:      make(map[peer.ID]*PeerGossipInfo),
        metrics:        make(map[peer.ID]*PeerMetrics),
        peerDiscovered: make(chan peer.ID, 100),
        peerLeft:       make(chan peer.ID, 100),
        peerUpdated:    make(chan peer.ID, 100),
    }

    // Start gossip protocol
    go gm.startGossiping()
    go gm.handlePeerUpdates()
    go gm.cleanupStaleEntries()

    return gm, nil
}

// Start begins the gossip protocol
func (gm *GossipManagerImpl) Start() error {
    go gm.startGossiping()
    go gm.handlePeerUpdates()
    go gm.cleanupStaleEntries()
    return nil
}

// Stop halts all gossip operations
func (gm *GossipManagerImpl) Stop() error {
    if gm.ctx != nil && gm.ctx.Err() == nil {
        gm.ctx.Done()
    }
    return nil
}

// Broadcast sends a message to the given topic
func (gm *GossipManagerImpl) Broadcast(topic string, data []byte) error {
    return gm.topic.Publish(gm.ctx, data)
}

// startGossiping periodically broadcasts peer information
func (gm *GossipManagerImpl) startGossiping() {
    ticker := time.NewTicker(GossipInterval)
    defer ticker.Stop()

    for {
        select {
        case <-gm.ctx.Done():
            return
        case <-ticker.C:
            gm.broadcastPeerInfo()
        }
    }
}

// broadcastPeerInfo shares this peer's information with the network
func (gm *GossipManagerImpl) broadcastPeerInfo() {
    addrs := make([]string, 0)
    for _, addr := range gm.host.Addrs() {
        addrs = append(addrs, addr.String())
    }

    info := &PeerGossipInfo{
        ID:        gm.host.ID(),
        Addresses: addrs,
        LastSeen:  time.Now(),
    }

    // Add metrics if available
    if metrics, ok := gm.metrics[gm.host.ID()]; ok {
        info.Uptime = gm.calculateUptime(metrics)
        info.ResponseTime = gm.calculateAverageResponseTime(metrics)
    }

    data, err := json.Marshal(info)
    if err != nil {
        return
    }

    gm.topic.Publish(gm.ctx, data)
}

// handlePeerUpdates processes incoming peer information
func (gm *GossipManagerImpl) handlePeerUpdates() {
    for {
        msg, err := gm.subscription.Next(gm.ctx)
        if err != nil {
            if gm.ctx.Err() != nil {
                return
            }
            continue
        }

        // Skip messages from ourselves
        if msg.ReceivedFrom == gm.host.ID() {
            continue
        }

        var info PeerGossipInfo
        if err := json.Unmarshal(msg.Data, &info); err != nil {
            continue
        }

        gm.updatePeerInfo(&info)
    }
}

// updatePeerInfo updates the stored peer information
func (gm *GossipManagerImpl) updatePeerInfo(info *PeerGossipInfo) {
    gm.mu.Lock()
    defer gm.mu.Unlock()

    // Update or add peer info
    existing, exists := gm.peerStore[info.ID]
    if !exists {
        gm.peerStore[info.ID] = info
        gm.metrics[info.ID] = &PeerMetrics{
            lastSeen:        time.Now(),
            connectionStart: time.Now(),
        }
        gm.peerDiscovered <- info.ID
    } else {
        // Update existing peer info
        existing.LastSeen = info.LastSeen
        existing.ChunkCount = info.ChunkCount
        existing.Uptime = info.Uptime
        existing.ResponseTime = info.ResponseTime
        gm.metrics[info.ID].lastSeen = time.Now()
        gm.peerUpdated <- info.ID
    }
}

// cleanupStaleEntries removes information about stale peers
func (gm *GossipManagerImpl) cleanupStaleEntries() {
    ticker := time.NewTicker(time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-gm.ctx.Done():
            return
        case <-ticker.C:
            gm.mu.Lock()
            now := time.Now()
            for id, metrics := range gm.metrics {
                if now.Sub(metrics.lastSeen) > PeerTimeoutInterval {
                    delete(gm.peerStore, id)
                    delete(gm.metrics, id)
                    gm.peerLeft <- id
                }
            }
            gm.mu.Unlock()
        }
    }
}

// AnnounceStorageNode announces this node as a storage provider
func (gm *GossipManagerImpl) AnnounceStorageNode(info *StorageNodeInfo) error {
    data, err := json.Marshal(struct {
        Type string         `json:"type"`
        Info *StorageNodeInfo `json:"info"`
    }{
        Type: "storage_announce",
        Info: info,
    })
    if err != nil {
        return err
    }
    return gm.topic.Publish(gm.ctx, data)
}

// RemoveStorageNode removes this node from storage providers
func (gm *GossipManagerImpl) RemoveStorageNode(nodeID string) error {
    data, err := json.Marshal(struct {
        Type   string `json:"type"`
        NodeID string `json:"node_id"`
    }{
        Type:   "storage_remove",
        NodeID: nodeID,
    })
    if err != nil {
        return err
    }
    return gm.topic.Publish(gm.ctx, data)
}

// NotifyStorageRejection notifies network of rejected storage request
func (gm *GossipManagerImpl) NotifyStorageRejection(req *StorageRequest, reason string) error {
    data, err := json.Marshal(struct {
        Type   string         `json:"type"`
        Request *StorageRequest `json:"request"`
        Reason string         `json:"reason"`
    }{
        Type:    "storage_reject",
        Request: req,
        Reason:  reason,
    })
    if err != nil {
        return err
    }
    return gm.topic.Publish(gm.ctx, data)
}

// NotifyStorageSuccess notifies network of successful storage
func (gm *GossipManagerImpl) NotifyStorageSuccess(req *StorageRequest) error {
    data, err := json.Marshal(struct {
        Type    string         `json:"type"`
        Request *StorageRequest `json:"request"`
    }{
        Type:    "storage_success",
        Request: req,
    })
    if err != nil {
        return err
    }
    return gm.topic.Publish(gm.ctx, data)
}

// GetPeers returns all known peers
func (gm *GossipManagerImpl) GetPeers() []peer.ID {
    gm.mu.RLock()
    defer gm.mu.RUnlock()

    peers := make([]peer.ID, 0, len(gm.peerStore))
    for id := range gm.peerStore {
        peers = append(peers, id)
    }
    return peers
}

// GetPeerInfo returns detailed information about known peers
func (gm *GossipManagerImpl) GetPeerInfo() []*PeerGossipInfo {
    gm.mu.RLock()
    defer gm.mu.RUnlock()

    peers := make([]*PeerGossipInfo, 0, len(gm.peerStore))
    for _, info := range gm.peerStore {
        peers = append(peers, info)
    }
    return peers
}

// RecordSuccess records a successful interaction with a peer
func (gm *GossipManagerImpl) RecordSuccess(id peer.ID, responseTime time.Duration) {
    gm.mu.Lock()
    defer gm.mu.Unlock()

    if metrics, ok := gm.metrics[id]; ok {
        metrics.successfulRequests++
        metrics.totalResponseTime += float64(responseTime.Milliseconds())
        metrics.lastResponseTime = time.Now()
        metrics.lastSeen = time.Now()
    }
}

// RecordFailure records a failed interaction with a peer
func (gm *GossipManagerImpl) RecordFailure(id peer.ID) {
    gm.mu.Lock()
    defer gm.mu.Unlock()

    if metrics, ok := gm.metrics[id]; ok {
        metrics.failedRequests++
        metrics.lastSeen = time.Now()
    }
}

// calculateUptime calculates the peer's uptime percentage
func (gm *GossipManagerImpl) calculateUptime(metrics *PeerMetrics) float64 {
    total := metrics.successfulRequests + metrics.failedRequests
    if total == 0 {
        return 0
    }
    return float64(metrics.successfulRequests) / float64(total) * 100
}

// calculateAverageResponseTime calculates the peer's average response time
func (gm *GossipManagerImpl) calculateAverageResponseTime(metrics *PeerMetrics) float64 {
    if metrics.successfulRequests == 0 {
        return 0
    }
    return metrics.totalResponseTime / float64(metrics.successfulRequests)
}
