package network

import (
"bytes"
"context"
"encoding/json"
"fmt"
"time"

	"github.com/ipfs/go-cid"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	mh "github.com/multiformats/go-multihash"
)

const (
	manifestTopic            = "filezap-manifests"
	replicationCheckInterval = time.Minute * 5
)

// NewManifestManager creates a new manifest manager
func NewManifestManager(ctx context.Context, localID peer.ID, dht *dht.IpfsDHT, ps *pubsub.PubSub) *ManifestManager {
	topic, err := ps.Join(manifestTopic)
	if err != nil {
		// Log error but continue - pubsub is optional for manifest sync
		fmt.Printf("failed to join manifest topic: %v\n", err)
	}

	mm := &ManifestManager{
		dht:       dht,
		store:     make(map[string]*ManifestInfo),
		localNode: localID,
		topic:     topic,
	}

	// Create and start replicator
	mm.replicator = NewManifestReplicator(dht, mm)
	go mm.replicator.Start(ctx)

	// Subscribe to manifest updates if topic was created
	if topic != nil {
		go mm.subscribeToUpdates(ctx)
	}

	return mm
}

// AddManifest stores a manifest and ensures it meets replication goals
func (m *ManifestManager) AddManifest(manifest *ManifestInfo) error {
	// Store locally
	m.store[manifest.Name] = manifest

	// Store in DHT
	data, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := m.dht.PutValue(context.Background(), getDHTKey(manifest.Name), data); err != nil {
		return fmt.Errorf("failed to store manifest in DHT: %w", err)
	}

	// Publish update if pubsub is available
	if m.topic != nil {
		if err := m.topic.Publish(context.Background(), data); err != nil {
			// Log error but continue - pubsub is optional
			fmt.Printf("failed to publish manifest update: %v\n", err)
		}
	}

	return nil
}

// GetManifest retrieves a manifest from local store or DHT
func (m *ManifestManager) GetManifest(name string) (*ManifestInfo, error) {
	// Check local store first
	if manifest, ok := m.store[name]; ok {
		return manifest, nil
	}

	// Try to get from DHT
	data, err := m.dht.GetValue(context.Background(), getDHTKey(name))
	if err != nil {
		return nil, fmt.Errorf("manifest not found: %w", err)
	}

	var manifest ManifestInfo
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	// Cache locally
	m.store[name] = &manifest
	return &manifest, nil
}

// subscribeToUpdates subscribes to manifest updates via pubsub
func (m *ManifestManager) subscribeToUpdates(ctx context.Context) {
	sub, err := m.topic.Subscribe()
	if err != nil {
		fmt.Printf("failed to subscribe to manifest updates: %v\n", err)
		return
	}
	defer sub.Cancel()

	for {
		msg, err := sub.Next(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return // Context cancelled
			}
			continue
		}

		// Skip messages from ourselves
		if msg.ReceivedFrom == m.localNode {
			continue
		}

		var manifest ManifestInfo
		if err := json.Unmarshal(msg.Data, &manifest); err != nil {
			continue
		}

		// Update local store
		m.store[manifest.Name] = &manifest
	}
}

// NewManifestReplicator creates a new manifest replicator
func NewManifestReplicator(dht *dht.IpfsDHT, manifests *ManifestManager) *ManifestReplicator {
	return &ManifestReplicator{
		dht:       dht,
		manifests: manifests,
		interval:  int(replicationCheckInterval.Seconds()),
	}
}

// Start begins periodic replication checks
func (r *ManifestReplicator) Start(ctx context.Context) {
	ticker := time.NewTicker(replicationCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.checkReplication()
		}
	}
}

// checkReplication ensures all manifests meet their replication goals
func (r *ManifestReplicator) checkReplication() {
    ctx := context.Background()

    // Get all manifests we're responsible for storing
    for _, manifest := range r.manifests.store {
        // Get the XOR distance between our node ID and the manifest key
        manifestKey := getDHTKey(manifest.Name)
        localDist := xorDistance(r.manifests.localNode.String(), manifestKey)

        // Get closest peers to this manifest
        peers, err := r.dht.GetClosestPeers(ctx, manifestKey)
        if err != nil {
            continue
        }

        // Sort peers by XOR distance to manifest
        peerDistances := make(map[peer.ID][]byte)
        for _, p := range peers {
            dist := xorDistance(p.String(), manifestKey)
            peerDistances[p] = dist
        }

        // Check if we're one of the N closest nodes
        closerPeers := 0
        for _, p := range peers {
            if bytes.Compare(peerDistances[p], localDist) < 0 {
                closerPeers++
            }
        }

        // If we're one of the N closest nodes, ensure we have the manifest
        if closerPeers < manifest.ReplicationGoal {
            // We should store this manifest
            if _, ok := r.manifests.store[manifest.Name]; !ok {
                // Get manifest from another peer
                data, err := r.dht.GetValue(ctx, manifestKey)
                if err != nil {
                    continue
                }

                var fetchedManifest ManifestInfo
                if err := json.Unmarshal(data, &fetchedManifest); err != nil {
                    continue
                }

                r.manifests.store[manifest.Name] = &fetchedManifest
            }

            // Announce that we're providing this manifest
            mhash, _ := mh.Sum([]byte(manifestKey), mh.SHA2_256, -1)
            manifestCID := cid.NewCidV1(cid.Raw, mhash)
            r.dht.Provide(ctx, manifestCID, true)
        }

        // Health check for all replicas
        manifestHash, _ := mh.Sum([]byte(manifestKey), mh.SHA2_256, -1)
        manifestCID := cid.NewCidV1(cid.Raw, manifestHash)
        providers, err := r.dht.FindProviders(ctx, manifestCID)
        if err != nil {
            continue
        }

        // If insufficient providers found, publish manifest again
        if len(providers) < manifest.ReplicationGoal {
            data, err := json.Marshal(manifest)
            if err != nil {
                continue
            }
            r.dht.PutValue(ctx, manifestKey, data)
        }
    }
}

// xorDistance calculates the XOR distance between two strings
func xorDistance(a, b string) []byte {
    aBytes := []byte(a)
    bBytes := []byte(b)
    length := len(aBytes)
    if len(bBytes) < length {
        length = len(bBytes)
    }
    result := make([]byte, length)
    for i := 0; i < length; i++ {
        result[i] = aBytes[i] ^ bBytes[i]
    }
    return result
}

// getDHTKey returns the DHT key for a manifest name
func getDHTKey(name string) string {
	return fmt.Sprintf("/filezap/manifest/%s", name)
}
