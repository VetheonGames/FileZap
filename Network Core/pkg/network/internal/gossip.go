package internal

import (
    "context"
    "encoding/json"
    "fmt"
    "sync"
    "time"

    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/peer"
    pubsub "github.com/libp2p/go-libp2p-pubsub"
)

const (
    peerTopicName    = "/filezap/peers/1.0.0"
    storageTopicName = "/filezap/storage/1.0.0"
    announceInterval = time.Minute
)

// gossipManager implements the GossipManager interface
type gossipManager struct {
    ctx      context.Context
    host     host.Host
    pubsub   *pubsub.PubSub
    peers    map[peer.ID]*PeerGossipInfo
    topics   map[string]*pubsub.Topic
    subs     map[string]*pubsub.Subscription
    mu       sync.RWMutex
}

func newGossipManager(ctx context.Context, h host.Host, ps *pubsub.PubSub) (GossipManager, error) {
    gm := &gossipManager{
        ctx:    ctx,
        host:   h,
        pubsub: ps,
        peers:  make(map[peer.ID]*PeerGossipInfo),
        topics: make(map[string]*pubsub.Topic),
        subs:   make(map[string]*pubsub.Subscription),
    }

    // Join topics
    if err := gm.joinTopics(); err != nil {
        return nil, err
    }

    // Start periodic announcements
    go gm.announceLoop()

    return gm, nil
}

func (gm *gossipManager) joinTopics() error {
    // Join peer discovery topic
    peerTopic, err := gm.pubsub.Join(peerTopicName)
    if err != nil {
        return fmt.Errorf("failed to join peer topic: %w", err)
    }
    gm.topics[peerTopicName] = peerTopic

    // Subscribe to peer announcements
    peerSub, err := peerTopic.Subscribe()
    if err != nil {
        return fmt.Errorf("failed to subscribe to peer topic: %w", err)
    }
    gm.subs[peerTopicName] = peerSub
    go gm.handlePeerAnnouncements(peerSub)

    // Join storage node topic
    storageTopic, err := gm.pubsub.Join(storageTopicName)
    if err != nil {
        return fmt.Errorf("failed to join storage topic: %w", err)
    }
    gm.topics[storageTopicName] = storageTopic

    // Subscribe to storage announcements
    storageSub, err := storageTopic.Subscribe()
    if err != nil {
        return fmt.Errorf("failed to subscribe to storage topic: %w", err)
    }
    gm.subs[storageTopicName] = storageSub
    go gm.handleStorageAnnouncements(storageSub)

    return nil
}

func (gm *gossipManager) announceLoop() {
    ticker := time.NewTicker(announceInterval)
    defer ticker.Stop()

    for {
        select {
        case <-gm.ctx.Done():
            return
        case <-ticker.C:
            gm.announcePeer()
        }
    }
}

func (gm *gossipManager) announcePeer() {
    // Create peer info
    info := &PeerGossipInfo{
        ID:            gm.host.ID(),
        LastSeen:      time.Now(),
        Uptime:        100.0, // TODO: Calculate real uptime
        ResponseTime:  0,     // TODO: Calculate response time
    }

    // Marshal info
    data, err := json.Marshal(info)
    if err != nil {
        return
    }

    // Publish to peer topic
    topic := gm.topics[peerTopicName]
    if topic != nil {
        topic.Publish(gm.ctx, data)
    }
}

func (gm *gossipManager) handlePeerAnnouncements(sub *pubsub.Subscription) {
    for {
        msg, err := sub.Next(gm.ctx)
        if err != nil {
            return
        }

        // Skip messages from self
        if msg.ReceivedFrom == gm.host.ID() {
            continue
        }

        // Unmarshal peer info
        var info PeerGossipInfo
        if err := json.Unmarshal(msg.Data, &info); err != nil {
            continue
        }

        // Update peer info
        gm.mu.Lock()
        gm.peers[info.ID] = &info
        gm.mu.Unlock()
    }
}

func (gm *gossipManager) handleStorageAnnouncements(sub *pubsub.Subscription) {
    for {
        msg, err := sub.Next(gm.ctx)
        if err != nil {
            return
        }

        // Skip messages from self
        if msg.ReceivedFrom == gm.host.ID() {
            continue
        }

        // Handle storage node announcement
        var info StorageNodeInfo
        if err := json.Unmarshal(msg.Data, &info); err != nil {
            continue
        }

        // TODO: Handle storage node info
    }
}

// Implement GossipManager interface

func (gm *gossipManager) AnnounceStorageNode(info *StorageNodeInfo) error {
    data, err := json.Marshal(info)
    if err != nil {
        return fmt.Errorf("failed to marshal storage info: %w", err)
    }

    topic := gm.topics[storageTopicName]
    if topic == nil {
        return fmt.Errorf("storage topic not joined")
    }

    return topic.Publish(gm.ctx, data)
}

func (gm *gossipManager) RemoveStorageNode(id string) error {
    // TODO: Implement storage node removal announcement
    return nil
}

func (gm *gossipManager) GetPeers() []*PeerGossipInfo {
    gm.mu.RLock()
    defer gm.mu.RUnlock()

    peers := make([]*PeerGossipInfo, 0, len(gm.peers))
    for _, info := range gm.peers {
        peers = append(peers, info)
    }
    return peers
}

func (gm *gossipManager) NotifyStorageSuccess(req *StorageRequest) error {
    // TODO: Implement storage success notification
    return nil
}

func (gm *gossipManager) NotifyStorageRejection(req *StorageRequest, reason string) error {
    // TODO: Implement storage rejection notification
    return nil
}
