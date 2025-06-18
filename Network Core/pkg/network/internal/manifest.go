package internal

import (
    "context"
    "encoding/json"
    "fmt"
    "sync"
    "time"

    "github.com/libp2p/go-libp2p/core/peer"
    dht "github.com/libp2p/go-libp2p-kad-dht"
    pubsub "github.com/libp2p/go-libp2p-pubsub"
)

const (
    manifestTopic = "/filezap/manifest/1.0.0"
)

// manifestManager implements the ManifestManager interface
type manifestManager struct {
    store     map[string]*ManifestInfo
    dht       *dht.IpfsDHT
    pubsub    *pubsub.PubSub
    topic     *pubsub.Topic
    sub       *pubsub.Subscription
    localNode peer.ID
    ctx       context.Context
    mu        sync.RWMutex
}

func newManifestManager(ctx context.Context, localID peer.ID, kadDHT *dht.IpfsDHT, ps *pubsub.PubSub) (*manifestManager, error) {
    // Create manager instance
    mm := &manifestManager{
        store:     make(map[string]*ManifestInfo),
        dht:       kadDHT,
        pubsub:    ps,
        localNode: localID,
        ctx:       ctx,
    }

    // Join manifest topic
    topic, err := ps.Join(manifestTopic)
    if err != nil {
        return nil, fmt.Errorf("failed to join manifest topic: %w", err)
    }
    mm.topic = topic

    // Subscribe to manifest updates
    sub, err := topic.Subscribe()
    if err != nil {
        return nil, fmt.Errorf("failed to subscribe to manifest topic: %w", err)
    }
    mm.sub = sub

    // Start subscription handler
    go mm.handleSubscription()

    return mm, nil
}

func (m *manifestManager) handleSubscription() {
    for {
        msg, err := m.sub.Next(m.ctx)
        if err != nil {
            // Context cancelled or subscription closed
            return
        }

        // Skip messages from self
        if msg.ReceivedFrom == m.localNode {
            continue
        }

        var manifest ManifestInfo
        if err := json.Unmarshal(msg.Data, &manifest); err != nil {
            continue
        }

        // Store manifest
        m.mu.Lock()
        m.store[manifest.Name] = &manifest
        m.mu.Unlock()
    }
}

// Implement ManifestManager interface

func (m *manifestManager) AddManifest(manifest *ManifestInfo) error {
    if manifest.ReplicationGoal == 0 {
        manifest.ReplicationGoal = DefaultReplicationGoal
    }
    manifest.UpdatedAt = time.Now()

    // Store locally
    m.mu.Lock()
    m.store[manifest.Name] = manifest
    m.mu.Unlock()

    // Store in DHT
    data, err := json.Marshal(manifest)
    if err != nil {
        return fmt.Errorf("failed to marshal manifest: %w", err)
    }
    if err := m.dht.PutValue(m.ctx, "/manifest/"+manifest.Name, data); err != nil {
        return fmt.Errorf("failed to store manifest in DHT: %w", err)
    }

    // Announce to network
    if err := m.topic.Publish(m.ctx, data); err != nil {
        return fmt.Errorf("failed to publish manifest: %w", err)
    }

    return nil
}

func (m *manifestManager) GetManifest(name string) (*ManifestInfo, map[string][]byte, error) {
    // Check local store first
    m.mu.RLock()
    manifest, exists := m.store[name]
    m.mu.RUnlock()

    if !exists {
        // Try DHT
        data, err := m.dht.GetValue(m.ctx, "/manifest/"+name)
        if err != nil {
            return nil, nil, fmt.Errorf("manifest not found: %w", err)
        }

        manifest = &ManifestInfo{}
        if err := json.Unmarshal(data, manifest); err != nil {
            return nil, nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
        }

        // Cache locally
        m.mu.Lock()
        m.store[name] = manifest
        m.mu.Unlock()
    }

    // For now, return without chunks - these will be fetched separately
    return manifest, nil, nil
}

func (m *manifestManager) RemoveManifest(name string) error {
    m.mu.Lock()
    delete(m.store, name)
    m.mu.Unlock()

    // TODO: Implement DHT removal
    return nil
}
