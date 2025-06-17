package network

import (
"context"
"encoding/json"
"fmt"
"testing"
"time"

"github.com/libp2p/go-libp2p"
dht "github.com/libp2p/go-libp2p-kad-dht"
pubsub "github.com/libp2p/go-libp2p-pubsub"
record "github.com/libp2p/go-libp2p-record"
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

    // Create DHTs with Server mode and validator
    nsval := record.NamespacedValidator{
        "pk":      record.PublicKeyValidator{},
        "ipns":    record.PublicKeyValidator{},
        "filezap": &validator{},
        "/filezap": &validator{}, // Add with leading slash too
    }

    d1, err := dht.New(ctx, h1,
        dht.Mode(dht.ModeServer),
        dht.ProtocolPrefix("/filezap"),
        dht.Validator(nsval),
    )
    require.NoError(t, err)

    d2, err := dht.New(ctx, h2,
        dht.Mode(dht.ModeServer),
        dht.ProtocolPrefix("/filezap"),
        dht.Validator(nsval),
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

    // Connect all peers first
    connectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    // Ensure both hosts are connected to each other
    require.NoError(t, h1.Connect(connectCtx, peer.AddrInfo{
        ID:    h2.ID(),
        Addrs: h2.Addrs(),
    }))
    
    // Wait for connection to be established
    time.Sleep(time.Second)

    // Bootstrap both DHTs
    bootstrapCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    require.NoError(t, d1.Bootstrap(bootstrapCtx))
    require.NoError(t, d2.Bootstrap(bootstrapCtx))

    // Wait for both DHTs to connect and initialize
    require.Eventually(t, func() bool {
        peers1 := d1.RoutingTable().ListPeers()
        peers2 := d2.RoutingTable().ListPeers()
        return len(peers1) > 0 && len(peers2) > 0
    }, 10*time.Second, 100*time.Millisecond, "DHT failed to initialize")

    // Additional wait for DHT to stabilize
    time.Sleep(2 * time.Second)

    // Add some random data to validate DHT is working
    testKey := getDHTKey("test-key")
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

func TestManifestErrorCases(t *testing.T) {
ctx := context.Background()
host, dht, ps := setupTestManifestNetwork(ctx, t)
defer host.Close()
defer dht.Close()

mm := NewManifestManager(ctx, host.ID(), dht, ps)

t.Run("Invalid Manifest", func(t *testing.T) {
// Test empty name
manifest := &ManifestInfo{
ChunkHashes:     []string{"hash1"},
ReplicationGoal: DefaultReplicationGoal,
Owner:           host.ID(),
Size:            1024,
}
err := mm.AddManifest(manifest)
assert.Error(t, err)

// Test empty chunk hashes
manifest.Name = "test.zap"
manifest.ChunkHashes = nil
err = mm.AddManifest(manifest)
assert.Error(t, err)

// Test invalid replication goal
manifest.ChunkHashes = []string{"hash1"}
manifest.ReplicationGoal = 0
err = mm.AddManifest(manifest)
assert.Error(t, err)

// Test missing owner
manifest.ReplicationGoal = DefaultReplicationGoal
manifest.Owner = ""
err = mm.AddManifest(manifest)
assert.Error(t, err)
})

t.Run("Network Disruption", func(t *testing.T) {
    // Create valid manifest
    manifest := &ManifestInfo{
        Name:            "disrupted.zap",
        ChunkHashes:     []string{"hash1"},
        ReplicationGoal: DefaultReplicationGoal,
        Owner:           host.ID(),
        Size:            1024,
    }

    // Add manifest
    err := mm.AddManifest(manifest)
    require.NoError(t, err)

    // Simulate network disruption by disconnecting from all peers
    for _, peer := range host.Network().Peers() {
        host.Network().ClosePeer(peer)
    }
    time.Sleep(time.Second) // Allow time for disconnections

    // Test DHT operations during network partition
    _, err = mm.GetManifest("nonexistent.zap")
    assert.Error(t, err, "should fail to get non-cached manifest during disruption")

    // Verify we can still access locally stored manifest
    cached, err := mm.GetManifest(manifest.Name)
    assert.NoError(t, err, "should be able to get cached manifest")
    assert.Equal(t, manifest.Name, cached.Name)

    // Adding new manifest should work (stored locally)
    newManifest := &ManifestInfo{
        Name:            "local-only.zap",
        ChunkHashes:     []string{"hash2"},
        ReplicationGoal: DefaultReplicationGoal,
        Owner:           host.ID(),
        Size:            1024,
    }
    err = mm.AddManifest(newManifest)
    assert.NoError(t, err, "should store locally during disruption")
})
}

func TestConcurrentManifestOperations(t *testing.T) {
ctx := context.Background()
host, dht, ps := setupTestManifestNetwork(ctx, t)
defer host.Close()
defer dht.Close()

mm := NewManifestManager(ctx, host.ID(), dht, ps)

const numOperations = 10
errors := make(chan error, numOperations*2)
done := make(chan bool, numOperations*2)

// Concurrent manifest operations
for i := 0; i < numOperations; i++ {
go func(i int) {
// Create and add manifest
manifest := &ManifestInfo{
Name:            fmt.Sprintf("concurrent%d.zap", i),
ChunkHashes:     []string{fmt.Sprintf("hash%d", i)},
ReplicationGoal: DefaultReplicationGoal,
Owner:           host.ID(),
Size:            1024,
}

if err := mm.AddManifest(manifest); err != nil {
errors <- fmt.Errorf("add error: %v", err)
return
}

// Retrieve manifest
_, err := mm.GetManifest(manifest.Name)
if err != nil {
errors <- fmt.Errorf("get error: %v", err)
return
}

// Update manifest
manifest.ChunkHashes = append(manifest.ChunkHashes, fmt.Sprintf("hash%d-2", i))
if err := mm.AddManifest(manifest); err != nil {
errors <- fmt.Errorf("update error: %v", err)
return
}

done <- true
}(i)
}

// Wait for all operations
for i := 0; i < numOperations; i++ {
select {
case err := <-errors:
t.Errorf("Concurrent operation failed: %v", err)
case <-done:
// Operation succeeded
}
}
}

func TestManifestEdgeCases(t *testing.T) {
ctx := context.Background()
host, dht, ps := setupTestManifestNetwork(ctx, t)
defer host.Close()
defer dht.Close()

mm := NewManifestManager(ctx, host.ID(), dht, ps)

t.Run("Update After Network Partition", func(t *testing.T) {
    // Create initial manifest
    manifest := &ManifestInfo{
        Name:            "partition.zap",
        ChunkHashes:     []string{"hash1"},
        ReplicationGoal: DefaultReplicationGoal,
        Owner:           host.ID(),
        Size:            1024,
    }

    err := mm.AddManifest(manifest)
    require.NoError(t, err)

    // Simulate network partition by disconnecting from all peers
    for _, peer := range host.Network().Peers() {
        host.Network().ClosePeer(peer)
    }
    time.Sleep(time.Second) // Allow time for disconnections

    // Attempt update during partition - should work locally
    manifest.ChunkHashes = append(manifest.ChunkHashes, "hash2")
    err = mm.AddManifest(manifest)
    assert.NoError(t, err, "should store locally during network partition")

    // Verify we can read the updated manifest locally
    retrieved, err := mm.GetManifest(manifest.Name)
    assert.NoError(t, err, "should retrieve from local store")
    assert.Equal(t, 2, len(retrieved.ChunkHashes), "should have updated chunk hashes")
    assert.Equal(t, manifest.ChunkHashes, retrieved.ChunkHashes)
})

t.Run("Large Manifest", func(t *testing.T) {
// Create manifest with many chunks
chunks := make([]string, 1000)
for i := range chunks {
chunks[i] = fmt.Sprintf("hash%d", i)
}

manifest := &ManifestInfo{
Name:            "large.zap",
ChunkHashes:     chunks,
ReplicationGoal: DefaultReplicationGoal,
Owner:           host.ID(),
Size:            1024 * 1024,
}

err := mm.AddManifest(manifest)
require.NoError(t, err)

// Verify retrieval works
retrieved, err := mm.GetManifest(manifest.Name)
require.NoError(t, err)
assert.Equal(t, len(chunks), len(retrieved.ChunkHashes))
})
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
