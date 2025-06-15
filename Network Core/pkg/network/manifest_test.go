package network

import (
"context"
"encoding/json"
"testing"
"time"

"github.com/libp2p/go-libp2p"
dht "github.com/libp2p/go-libp2p-kad-dht"
pubsub "github.com/libp2p/go-libp2p-pubsub"
"github.com/libp2p/go-libp2p/core/host"
"github.com/libp2p/go-libp2p/core/peer"
"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"
)

func setupTestManifestNetwork(ctx context.Context, t *testing.T) (host.Host, *dht.IpfsDHT, *pubsub.PubSub) {
    // Create hosts
    h1, err := libp2p.New(
        libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
        libp2p.DefaultTransports,
    )
    require.NoError(t, err)

    h2, err := libp2p.New(
        libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
        libp2p.DefaultTransports,
    )
    require.NoError(t, err)

    // Create DHTs and set up validators
    d1, err := dht.New(ctx, h1,
        dht.Mode(dht.ModeServer),
        dht.ProtocolPrefix("/filezap"),
    )
    require.NoError(t, err)

    d2, err := dht.New(ctx, h2,
        dht.Mode(dht.ModeServer),
        dht.ProtocolPrefix("/filezap"),
    )
    require.NoError(t, err)
    defer d2.Close()
    defer h2.Close()

    // Create pubsub
    ps, err := pubsub.NewGossipSub(ctx, h1)
    require.NoError(t, err)

    // Connect the hosts
    require.NoError(t, h1.Connect(ctx, peer.AddrInfo{
        ID:    h2.ID(),
        Addrs: h2.Addrs(),
    }))

    // Bootstrap both DHTs
    bootstrapCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    require.NoError(t, d1.Bootstrap(bootstrapCtx))
    require.NoError(t, d2.Bootstrap(bootstrapCtx))

    // Wait for DHTs to initialize and connect
    require.Eventually(t, func() bool {
        if len(d1.RoutingTable().ListPeers()) == 0 {
            return false
        }
        if len(d2.RoutingTable().ListPeers()) == 0 {
            return false
        }
        return true
    }, 5*time.Second, 100*time.Millisecond, "DHT failed to initialize")

    // Add some random data to validate DHT is working
    testKey := "filezap/test-key"
    testManifest := &ManifestInfo{
        Name: "test-key",
        ChunkHashes: []string{"test"},
        ReplicationGoal: DefaultReplicationGoal,
        Owner: h1.ID(),
        Size: 100,
        UpdatedAt: time.Now(),
    }
    testData, err := json.Marshal(testManifest)
    require.NoError(t, err)
    require.NoError(t, d1.PutValue(ctx, testKey, testData))

    // Verify DHT can retrieve data
    require.Eventually(t, func() bool {
        val, err := d2.GetValue(ctx, testKey)
        return err == nil && len(val) > 0
    }, 5*time.Second, 100*time.Millisecond, "DHT retrieval failed")

    return h1, d1, ps
}

func TestManifestBasicOperations(t *testing.T) {
	ctx := context.Background()
    host, dht, ps := setupTestManifestNetwork(ctx, t)
	defer host.Close()
	defer dht.Close()

	mm := NewManifestManager(ctx, host.ID(), dht, ps)

	// Create test manifest
	testManifest := &ManifestInfo{
		Name:            "test.zap",
		ChunkHashes:     []string{"hash1", "hash2"},
		ReplicationGoal: DefaultReplicationGoal,
		Owner:           host.ID(),
		Size:            1024,
	}

	// Add manifest
	err := mm.AddManifest(testManifest)
	require.NoError(t, err)

	// Retrieve and verify manifest
	retrieved, err := mm.GetManifest("test.zap")
	require.NoError(t, err)
	assert.Equal(t, testManifest.Name, retrieved.Name)
	assert.Equal(t, testManifest.ChunkHashes, retrieved.ChunkHashes)
	assert.Equal(t, testManifest.ReplicationGoal, retrieved.ReplicationGoal)
	assert.Equal(t, testManifest.Size, retrieved.Size)
}

func TestManifestReplication(t *testing.T) {
	ctx := context.Background()

	// Create two nodes
    host1, dht1, ps1 := setupTestManifestNetwork(ctx, t)
	defer host1.Close()
	defer dht1.Close()

    host2, dht2, ps2 := setupTestManifestNetwork(ctx, t)
	defer host2.Close()
	defer dht2.Close()

	// Connect the nodes
	peerInfo := peer.AddrInfo{
		ID:    host2.ID(),
		Addrs: host2.Addrs(),
	}
	require.NoError(t, host1.Connect(ctx, peerInfo))
	time.Sleep(time.Millisecond * 100) // Wait for connection

	// Create manifest managers
	mm1 := NewManifestManager(ctx, host1.ID(), dht1, ps1)
	mm2 := NewManifestManager(ctx, host2.ID(), dht2, ps2)

	// Create and add manifest on first node
	testManifest := &ManifestInfo{
		Name:            "replicated.zap",
		ChunkHashes:     []string{"hash1", "hash2"},
		ReplicationGoal: 2, // Set to 2 to ensure both nodes should have it
		Owner:           host1.ID(),
		Size:            1024,
	}

	err := mm1.AddManifest(testManifest)
	require.NoError(t, err)

	// Wait for replication
	time.Sleep(time.Second * 2)

	// Verify manifest is available on second node
	retrieved, err := mm2.GetManifest("replicated.zap")
	require.NoError(t, err)
	assert.Equal(t, testManifest.Name, retrieved.Name)
	assert.Equal(t, testManifest.ChunkHashes, retrieved.ChunkHashes)
}

func TestManifestUpdates(t *testing.T) {
ctx := context.Background()
host1, dht1, ps1 := setupTestManifestNetwork(ctx, t)
	defer host1.Close()
	defer dht1.Close()

	host2, dht2, ps2 := setupTestManifestNetwork(ctx, t)
	defer host2.Close()
	defer dht2.Close()

	// Connect the nodes
	peerInfo := peer.AddrInfo{
		ID:    host2.ID(),
		Addrs: host2.Addrs(),
	}
	require.NoError(t, host1.Connect(ctx, peerInfo))

	mm1 := NewManifestManager(ctx, host1.ID(), dht1, ps1)
	mm2 := NewManifestManager(ctx, host2.ID(), dht2, ps2)

	// Initial manifest
	manifest := &ManifestInfo{
		Name:            "updated.zap",
		ChunkHashes:     []string{"hash1"},
		ReplicationGoal: 2,
		Owner:           host1.ID(),
		Size:            512,
	}

	// Add initial manifest
	err := mm1.AddManifest(manifest)
	require.NoError(t, err)

	// Update manifest with new data
	manifest.ChunkHashes = append(manifest.ChunkHashes, "hash2")
	manifest.Size = 1024
	err = mm1.AddManifest(manifest)
	require.NoError(t, err)

	// Wait for update propagation
	time.Sleep(time.Second)

	// Verify updated manifest on second node
	retrieved, err := mm2.GetManifest("updated.zap")
	require.NoError(t, err)
	assert.Equal(t, 2, len(retrieved.ChunkHashes))
	assert.Equal(t, int64(1024), retrieved.Size)
}

func TestManifestNonexistent(t *testing.T) {
	ctx := context.Background()
	host, dht, ps := setupTestManifestNetwork(ctx, t)
	defer host.Close()
	defer dht.Close()

	mm := NewManifestManager(ctx, host.ID(), dht, ps)

	// Try to retrieve nonexistent manifest
	_, err := mm.GetManifest("nonexistent.zap")
	assert.Error(t, err)
}
