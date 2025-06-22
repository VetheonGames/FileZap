package vpn

import (
    "context"
    "encoding/json"
    "fmt"
    "sync"
    "time"

    dht "github.com/libp2p/go-libp2p-kad-dht"
    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/peer"
    pubsub "github.com/libp2p/go-libp2p-pubsub"
)

const (
    // Topic name for VPN peer discovery
    discoveryTopic = "filezap-vpn-discovery"
    
    // Announcement interval
    announceInterval = 30 * time.Second
    
    // Peer timeout
    peerTimeout = 2 * time.Minute
)

// Discovery handles VPN peer discovery and announcements
type Discovery struct {
    host      host.Host
    dht       *dht.IpfsDHT
    ps        *pubsub.PubSub
    topic     *pubsub.Topic
    sub       *pubsub.Subscription
    vpn       *VPNManager
    peerInfo  PeerInfo
    peers     sync.Map
    ctx       context.Context
    cancel    context.CancelFunc
}

// PeerInfo contains information about a VPN peer
type PeerInfo struct {
    PeerID    peer.ID `json:"peer_id"`
    VirtualIP string  `json:"virtual_ip"`
    Timestamp int64   `json:"timestamp"`
}

// NewDiscovery creates a new peer discovery service
func NewDiscovery(ctx context.Context, h host.Host, d *dht.IpfsDHT, ps *pubsub.PubSub, vpn *VPNManager) (*Discovery, error) {
    // Create discovery context
    ctx, cancel := context.WithCancel(ctx)

    // Join pubsub topic
    topic, err := ps.Join(discoveryTopic)
    if err != nil {
        cancel()
        return nil, fmt.Errorf("failed to join topic: %w", err)
    }

    // Subscribe to topic
    sub, err := topic.Subscribe()
    if err != nil {
        cancel()
        topic.Close()
        return nil, fmt.Errorf("failed to subscribe: %w", err)
    }

    disc := &Discovery{
        host:    h,
        dht:     d,
        ps:      ps,
        topic:   topic,
        sub:     sub,
        vpn:     vpn,
        ctx:     ctx,
        cancel:  cancel,
        peerInfo: PeerInfo{
            PeerID:    h.ID(),
            VirtualIP: vpn.GetLocalIP(),
        },
    }

    // Start announcement and message handling
    go disc.announcePeriodically()
    go disc.handleMessages()

    return disc, nil
}

// Close shuts down the discovery service
func (d *Discovery) Close() error {
    d.cancel()
    d.sub.Cancel()
    return d.topic.Close()
}

func (d *Discovery) announcePeriodically() {
    ticker := time.NewTicker(announceInterval)
    defer ticker.Stop()

    for {
        select {
        case <-d.ctx.Done():
            return
        case <-ticker.C:
            d.announce()
        }
    }
}

func (d *Discovery) announce() {
    // Update timestamp
    d.peerInfo.Timestamp = time.Now().Unix()

    // Marshal peer info
    data, err := json.Marshal(d.peerInfo)
    if err != nil {
        return
    }

    // Publish to topic
    if err := d.topic.Publish(d.ctx, data); err != nil {
        fmt.Printf("Failed to publish announcement: %v\n", err)
    }
}

func (d *Discovery) handleMessages() {
    for {
        msg, err := d.sub.Next(d.ctx)
        if err != nil {
            if d.ctx.Err() != nil {
                return
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

        // Check timestamp
        if time.Since(time.Unix(info.Timestamp, 0)) > peerTimeout {
            continue
        }

        // Store/update peer
        d.peers.Store(info.PeerID, info)

        // Update VPN routing
        d.vpn.handlePeerAnnouncement(info)
    }
}

// GetPeers returns a list of known peers
func (d *Discovery) GetPeers() []PeerInfo {
    var peers []PeerInfo
    d.peers.Range(func(key, value interface{}) bool {
        if info, ok := value.(PeerInfo); ok {
            // Filter out old peers
            if time.Since(time.Unix(info.Timestamp, 0)) <= peerTimeout {
                peers = append(peers, info)
            } else {
                d.peers.Delete(key)
            }
        }
        return true
    })
    return peers
}
