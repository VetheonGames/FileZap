package network

import (
    "context"
    "crypto/rand"
    "fmt"
    "testing"
    "time"

    "github.com/libp2p/go-libp2p/core/crypto"
    "github.com/libp2p/go-libp2p/core/crypto/pb"
    "github.com/libp2p/go-libp2p/core/network"
    "github.com/libp2p/go-libp2p/core/peer"
    ma "github.com/multiformats/go-multiaddr"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// fakeKey implements crypto.PubKey for testing
type fakeKey struct {
    data []byte
}

func (f *fakeKey) Verify(_ []byte, _ []byte) (bool, error) { return true, nil }
func (f *fakeKey) Type() pb.KeyType                        { return pb.KeyType_RSA }
func (f *fakeKey) Equals(_ crypto.Key) bool                { return false }
func (f *fakeKey) Raw() ([]byte, error)                   { return f.data, nil }
func (f *fakeKey) Bytes() ([]byte, error)                 { return f.data, nil }
func (f *fakeKey) String() string                         { return string(f.data) }

func TestNewNetworkEngine(t *testing.T) {
    ctx := context.Background()
    cfg := DefaultNetworkConfig()
    engine, err := NewNetworkEngine(ctx, cfg)
    require.NoError(t, err)
    defer engine.Close()

    require.NotNil(t, engine.transportNode)
    require.NotNil(t, engine.metadataNode)
    require.NotNil(t, engine.manifests)
    require.NotNil(t, engine.chunkStore)

    tAddrs := engine.transportNode.host.Addrs()
    mAddrs := engine.metadataNode.host.Addrs()
    require.NotEmpty(t, tAddrs)
    require.NotEmpty(t, mAddrs)
    require.NotEqual(t, tAddrs[0], mAddrs[0])
}

// setupTestNetwork creates a test network with two connected engines and bootstrapping
func setupTestNetwork(t *testing.T, ctx context.Context) (*NetworkEngine, *NetworkEngine) {
    cfg := DefaultNetworkConfig()

    // Create two engines
    engine1, err := NewNetworkEngine(ctx, cfg)
    require.NoError(t, err)

    engine2, err := NewNetworkEngine(ctx, cfg)
    require.NoError(t, err)

    // Connect engine2 to engine1
    tAddr := engine1.transportNode.host.Addrs()[0]
    tID := engine1.transportNode.host.ID()
    info := peer.AddrInfo{
        ID:    tID,
        Addrs: []ma.Multiaddr{tAddr},
    }
    err = engine2.Bootstrap([]peer.AddrInfo{info})
    require.NoError(t, err)

    // Wait for connections to be established
    require.Eventually(t, func() bool {
        return engine1.transportNode.host.Network().Connectedness(engine2.transportNode.host.ID()) == network.Connected &&
            engine1.metadataNode.host.Network().Connectedness(engine2.metadataNode.host.ID()) == network.Connected
    }, 5*time.Second, 100*time.Millisecond)

    return engine1, engine2
}

func TestZapFileOperations(t *testing.T) {
    ctx := context.Background()
    engine1, engine2 := setupTestNetwork(t, ctx)
    defer engine1.Close()
    defer engine2.Close()

    now := time.Now()
    manifest := &ManifestInfo{
        Name:            "test.txt",
        Size:            1024,
        ChunkHashes:     []string{"hash1", "hash2"},
        Owner:           engine1.transportNode.host.ID(),
        UpdatedAt:       now,
        ReplicationGoal: DefaultReplicationGoal,
    }

    chunks := map[string][]byte{
        "hash1": []byte("chunk1 data"),
        "hash2": []byte("chunk2 data"),
    }

    t.Run("Add and Get ZapFile", func(t *testing.T) {
        // Test adding file
        err := engine1.AddZapFile(manifest, chunks)
        require.NoError(t, err)

        // Test retrieval from the same node
        retrievedManifest, retrievedChunks, err := engine1.GetZapFile(manifest.Name)
        require.NoError(t, err)

        // Compare all fields except UpdatedAt
        assert.Equal(t, manifest.Name, retrievedManifest.Name)
        assert.Equal(t, manifest.Size, retrievedManifest.Size)
        assert.Equal(t, manifest.ChunkHashes, retrievedManifest.ChunkHashes)
        assert.Equal(t, manifest.Owner, retrievedManifest.Owner)
        assert.Equal(t, manifest.ReplicationGoal, retrievedManifest.ReplicationGoal)

        // Compare UpdatedAt with tolerance
        timeDiff := retrievedManifest.UpdatedAt.Sub(manifest.UpdatedAt)
        assert.True(t, timeDiff.Abs() < time.Second, "UpdatedAt times should be within 1 second")

        assert.Equal(t, chunks, retrievedChunks)

        // Test retrieval from a different node
        retrievedManifest, retrievedChunks, err = engine2.GetZapFile(manifest.Name)
        require.NoError(t, err)

        // Compare all fields except UpdatedAt
        assert.Equal(t, manifest.Name, retrievedManifest.Name)
        assert.Equal(t, manifest.Size, retrievedManifest.Size)
        assert.Equal(t, manifest.ChunkHashes, retrievedManifest.ChunkHashes)
        assert.Equal(t, manifest.Owner, retrievedManifest.Owner)
        assert.Equal(t, manifest.ReplicationGoal, retrievedManifest.ReplicationGoal)

        // Compare UpdatedAt with tolerance
        timeDiff = retrievedManifest.UpdatedAt.Sub(manifest.UpdatedAt)
        assert.True(t, timeDiff.Abs() < time.Second, "UpdatedAt times should be within 1 second")

        assert.Equal(t, chunks, retrievedChunks)
    })

    t.Run("Nonexistent File", func(t *testing.T) {
        _, _, err := engine1.GetZapFile("nonexistent.txt")
        assert.Error(t, err)
    })
}

func TestBootstrapping(t *testing.T) {
    ctx := context.Background()
    cfg := DefaultNetworkConfig()

    engine1, err := NewNetworkEngine(ctx, cfg)
    require.NoError(t, err)
    defer engine1.Close()

    // Create bootstrap nodes
    bootstrapNodes := make([]*NetworkEngine, 3)
    bootstrapAddrs := make([]peer.AddrInfo, 3)

    for i := range bootstrapNodes {
        node, err := NewNetworkEngine(ctx, cfg)
        require.NoError(t, err)
        defer node.Close()
        bootstrapNodes[i] = node

        addr := node.transportNode.host.Addrs()[0]
        peerID := node.transportNode.host.ID()
        bootstrapAddrs[i] = peer.AddrInfo{
            ID:    peerID,
            Addrs: []ma.Multiaddr{addr},
        }
    }

    // Test bootstrapping
    err = engine1.Bootstrap(bootstrapAddrs)
    require.NoError(t, err)

    // Verify connections
    for _, node := range bootstrapNodes {
        require.Eventually(t, func() bool {
            return engine1.transportNode.host.Network().Connectedness(node.transportNode.host.ID()) == network.Connected
        }, 5*time.Second, 100*time.Millisecond)
    }

    // Test failed bootstrap with non-existent peer
    randBytes := make([]byte, 32)
    _, err = rand.Read(randBytes)
    require.NoError(t, err)

    privKey, _, err := crypto.GenerateECDSAKeyPair(rand.Reader)
    require.NoError(t, err)

    id, err := peer.IDFromPrivateKey(privKey)
    require.NoError(t, err)

    invalidAddr := peer.AddrInfo{
        ID:    id,
        Addrs: []ma.Multiaddr{ma.StringCast("/ip4/127.0.0.1/tcp/1234")},
    }
    err = engine1.Bootstrap([]peer.AddrInfo{invalidAddr})
    assert.Error(t, err)
}

func TestNetworkFailures(t *testing.T) {
    ctx := context.Background()
    engine1, engine2 := setupTestNetwork(t, ctx)

    manifest := &ManifestInfo{
        Name:            "test.txt",
        Size:            1024,
        ChunkHashes:     []string{"hash1"},
        Owner:           engine1.transportNode.host.ID(),
        ReplicationGoal: DefaultReplicationGoal,
    }
    chunks := map[string][]byte{"hash1": []byte("data")}

    // Add file before simulating failures
    err := engine1.AddZapFile(manifest, chunks)
    require.NoError(t, err)

    // Test network partition
    engine2.Close()
    _, _, err = engine1.GetZapFile(manifest.Name)
    assert.NoError(t, err, "should work from cache")

    // Test new node after partition
    cfg := DefaultNetworkConfig()
    engine3, err := NewNetworkEngine(ctx, cfg)
    require.NoError(t, err)
    defer engine3.Close()

    _, _, err = engine3.GetZapFile(manifest.Name)
    assert.Error(t, err, "should fail without connection to owner")

    // Test cleanup
    engine1.Close()
    _, _, err = engine3.GetZapFile(manifest.Name)
    assert.Error(t, err, "should fail after owner shutdown")
}

func TestConcurrentOperations(t *testing.T) {
    ctx := context.Background()
    engine1, engine2 := setupTestNetwork(t, ctx)
    defer engine1.Close()
    defer engine2.Close()

    const numOperations = 10
    errors := make(chan error, numOperations*2)
    done := make(chan bool, numOperations*2)

    // Concurrent file operations
    for i := 0; i < numOperations; i++ {
        go func(i int) {
            manifest := &ManifestInfo{
                Name:            fmt.Sprintf("test%d.txt", i),
                Size:            1024,
                ChunkHashes:     []string{fmt.Sprintf("hash%d", i)},
                Owner:           engine1.transportNode.host.ID(),
                ReplicationGoal: DefaultReplicationGoal,
            }
            chunks := map[string][]byte{
                fmt.Sprintf("hash%d", i): []byte(fmt.Sprintf("data%d", i)),
            }

            if err := engine1.AddZapFile(manifest, chunks); err != nil {
                errors <- fmt.Errorf("add error: %v", err)
                return
            }

            _, _, err := engine2.GetZapFile(manifest.Name)
            if err != nil {
                errors <- fmt.Errorf("get error: %v", err)
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
