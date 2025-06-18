package vpn

import (
    "context"
    "crypto/sha256"
    "encoding/json"
    "fmt"
    "sync"
    "time"

    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/peer"
    dht "github.com/libp2p/go-libp2p-kad-dht"
    pubsub "github.com/libp2p/go-libp2p-pubsub"
)

const (
    // Topic name for peer announcements
    announcementTopic = "vpn-peers"
    
    // How often to republish peer info
    announcementInterval = 1 * time.Minute
    
    // How long to consider peer info valid
    peerTTL = 2 * time.Minute
)

// PeerInfo represents the information shared about a peer
type PeerInfo struct {
    PeerID    peer.ID `json:"peer_id"`
    VirtualIP string  `json:"virtual_ip"`
    Timestamp int64   `json:"timestamp"`
}

// Discovery manages peer discovery using DHT and pubsub
type Discovery struct {
    ctx       context.Context
    host      host.Host
    dht       *dht.IpfsDHT
    pubsub    *pubsub.PubSub
    topic     *pubsub.Topic
    sub       *pubsub.Subscription
    vpn       *VPNManager
    peerInfo  PeerInfo
    mu        sync.RWMutex
    cancel    context.CancelFunc
}

// NewDiscovery creates a new peer discovery service
func NewDiscovery(ctx context.Context, h host.Host, dht *dht.IpfsDHT, ps *pubsub.PubSub, vpn *VPNManager) (*Discovery, error) {
    // Create discovery context
    dctx, cancel := context.WithCancel(ctx)
    
    // Join pubsub topic
    topic, err := ps.Join(announcementTopic)
    if err != nil {
        cancel()
        return nil, fmt.Errorf("failed to join topic: %w", err)
    }
    
    // Subscribe to topic
    sub, err := topic.Subscribe()
    if err != nil {
        cancel()
        topic.Close()
        return nil, fmt.Errorf("failed to subscribe to topic: %w", err)
    }

    d := &Discovery{
        ctx:     dctx,
        host:    h,
        dht:     dht,
        pubsub:  ps,
        topic:   topic,
        sub:     sub,
        vpn:     vpn,
        cancel:  cancel,
    }

    // Start message handler
    go d.handleMessages()
    
    // Start periodic announcements
    go d.announceRoutine()

    return d, nil
}

// Close shuts down the discovery service
func (d *Discovery) Close() error {
    d.cancel()
    d.sub.Cancel()
    return d.topic.Close()
}

// handleMessages processes incoming peer announcements
func (d *Discovery) handleMessages() {
    for {
        msg, err := d.sub.Next(d.ctx)
        if err != nil {
            if d.ctx.Err() != nil {
                return // Context cancelled
            }
            continue
        }

        // Skip messages from self
        if msg.ReceivedFrom == d.host.ID() {
            continue
        }

        // Parse peer info
        var info PeerInfo
        if err := json.Unmarshal(msg.Data, &info); err != nil {
            continue
        }

        // Validate timestamp
        if time.Since(time.Unix(info.Timestamp, 0)) > peerTTL {
            continue
        }

        // Update VPN routing
        d.vpn.handlePeerAnnouncement(info)
    }
}

// announceRoutine periodically announces this peer's presence
func (d *Discovery) announceRoutine() {
    ticker := time.NewTicker(announcementInterval)
    defer ticker.Stop()

    for {
        select {
        case <-d.ctx.Done():
            return
        case <-ticker.C:
            if err := d.announce(); err != nil {
                // Log error but continue
                continue
            }
        }
    }
}

// announce publishes this peer's information
func (d *Discovery) announce() error {
    d.mu.RLock()
    info := d.peerInfo
    d.mu.RUnlock()

    // Update timestamp
    info.Timestamp = time.Now().Unix()

    // Marshal peer info
    data, err := json.Marshal(info)
    if err != nil {
        return fmt.Errorf("failed to marshal peer info: %w", err)
    }

    // Publish to topic
    return d.topic.Publish(d.ctx, data)
}

// generateRendezvousKey generates a deterministic key for DHT rendezvous
func generateRendezvousKey(seed string) []byte {
    hash := sha256.Sum256([]byte(seed))
    return hash[:]
}
