package network

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "strings"
    "time"

    "github.com/ipfs/go-cid"
    dht "github.com/libp2p/go-libp2p-kad-dht"
    record "github.com/libp2p/go-libp2p-record"
    pubsub "github.com/libp2p/go-libp2p-pubsub"
    "github.com/libp2p/go-libp2p/core/peer"
    mh "github.com/multiformats/go-multihash"
)

// Custom validator for DHT records
type validator struct{}

func (v *validator) Validate(key string, value []byte) error {
    parts := strings.Split(key, "/")
    
    // The key should be in the format: /filezap/key or filezap/key
    if !strings.Contains(key, "filezap") {
        return fmt.Errorf("key does not contain filezap namespace: %s", key)
    }
    
    // Extract namespace
    var namespace string
    if strings.HasPrefix(key, "/") {
        if len(parts) < 3 {
            return fmt.Errorf("invalid key format with leading slash: %s", key)
        }
        namespace = parts[1]
    } else {
        if len(parts) < 2 {
            return fmt.Errorf("invalid key format without leading slash: %s", key)
        }
        namespace = parts[0]
    }
    
    if namespace != "filezap" {
        return fmt.Errorf("invalid record keytype: expected filezap namespace, got %s", namespace)
    }
    
    // Try to unmarshal to verify it's a valid manifest
    var manifest ManifestInfo
    if err := json.Unmarshal(value, &manifest); err != nil {
        return fmt.Errorf("invalid manifest data: %w", err)
    }
    
    return nil
}

func (v *validator) Select(key string, values [][]byte) (int, error) {
    if len(values) == 0 {
        return 0, fmt.Errorf("no values to select from")
    }
    
    // Select the most recent valid manifest
    var latest time.Time
    selected := 0
    
    for i, value := range values {
        var manifest ManifestInfo
        if err := json.Unmarshal(value, &manifest); err != nil {
            continue
        }
        
        // Compare timestamps if available, otherwise keep first valid one
        if latest.IsZero() || manifest.UpdatedAt.After(latest) {
            latest = manifest.UpdatedAt
            selected = i
        }
    }
    
    return selected, nil
}

const (
	manifestTopic            = "filezap-manifests"
	replicationCheckInterval = time.Minute * 5
)

// NewManifestManager creates a new manifest manager
func NewManifestManager(ctx context.Context, localID peer.ID, kdht *dht.IpfsDHT, ps *pubsub.PubSub) *ManifestManager {
    // Set up validator
    nsval := record.NamespacedValidator{
        "pk":     record.PublicKeyValidator{},
        "ipns":   record.PublicKeyValidator{},
        "filezap": &validator{},
    }
    kdht.Validator = nsval

    // Wait for DHT initialization
    timeout := time.After(30 * time.Second)
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-timeout:
            // Continue even if timeout occurs, but log warning
            fmt.Printf("warning: timeout waiting for DHT initialization\n")
            goto init
        case <-ticker.C:
            if len(kdht.RoutingTable().ListPeers()) > 0 {
                goto init
            }
        case <-ctx.Done():
            return nil
        }
    }

init:
    topic, err := ps.Join(manifestTopic)
    if err != nil {
        // Log error but continue - pubsub is optional for manifest sync
        fmt.Printf("failed to join manifest topic: %v\n", err)
    }

    mm := &ManifestManager{
        dht:       kdht,
        store:     make(map[string]*ManifestInfo),
        localNode: localID,
        topic:     topic,
    }

// Create and start replicator
mm.replicator = NewManifestReplicator(kdht, mm)
	go mm.replicator.Start(ctx)

	// Subscribe to manifest updates if topic was created
	if topic != nil {
		go mm.subscribeToUpdates(ctx)
	}

	return mm
}

// AddManifest stores a manifest and ensures it meets replication goals
func (m *ManifestManager) AddManifest(manifest *ManifestInfo) error {
    // Validate manifest
    if manifest == nil {
        return fmt.Errorf("manifest cannot be nil")
    }
    if manifest.Name == "" {
        return fmt.Errorf("manifest name cannot be empty")
    }
    if len(manifest.ChunkHashes) == 0 {
        return fmt.Errorf("manifest must have at least one chunk hash")
    }
    if manifest.ReplicationGoal <= 0 {
        return fmt.Errorf("replication goal must be greater than 0")
    }
    if manifest.Owner == "" {
        return fmt.Errorf("manifest must have an owner")
    }

    // Set update timestamp
    manifest.UpdatedAt = time.Now()

    // Store locally
    m.store[manifest.Name] = manifest

// Store in DHT
data, err := json.Marshal(manifest)
if err != nil {
    return fmt.Errorf("failed to marshal manifest: %w", err)
}

// Ensure DHT has peers before storing
peers := m.dht.RoutingTable().ListPeers()
if len(peers) == 0 {
    // Store locally and retry DHT storage later via replicator
    return nil
}

// Use a timeout context for DHT operations
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := m.dht.PutValue(ctx, getDHTKey(manifest.Name), data); err != nil {
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
dhtKey := getDHTKey(name)
data, err := m.dht.GetValue(context.Background(), dhtKey)
if err != nil {
    // Pass through the DHT error directly so it can be properly checked
    return nil, err
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
			mhash, err := mh.Sum([]byte(manifestKey), mh.SHA2_256, -1)
			if err != nil {
				continue
			}
			manifestCID := cid.NewCidV1(cid.Raw, mhash)
			if err := r.dht.Provide(ctx, manifestCID, true); err != nil {
				continue
			}
		}

		// Health check for all replicas
		manifestHash, err := mh.Sum([]byte(manifestKey), mh.SHA2_256, -1)
		if err != nil {
			continue
		}
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
			if err := r.dht.PutValue(ctx, manifestKey, data); err != nil {
				continue
			}
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
    // Following libp2p-record key format
    key := strings.TrimPrefix(name, "/")
    key = strings.TrimPrefix(key, "filezap/")
    result := "/filezap/" + key  // Leading slash is required for DHT keys
    return result
}
