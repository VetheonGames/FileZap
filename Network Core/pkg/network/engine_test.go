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
    engine, err := NewNetworkEngine(ctx)
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
    // Create two engines
    engine1, err := NewNetworkEngine(ctx)
    require.NoError(t, err)

    engine2, err := NewNetworkEngine(ctx)
    require.NoError(t, err)

    // Engine1 will act as the bootstrap node for Engine2
    transportAddr := engine1.GetTransportHost().Addrs()[0]
    metadataAddr := engine1.GetMetadataHost().Addrs()[0]

    // Connect transport networks
    transportInfo := peer.AddrInfo{
        ID:    engine1.GetTransportHost().ID(),
        Addrs: []ma.Multiaddr{transportAddr},
    }
    err = engine2.GetTransportHost().Connect(ctx, transportInfo)
    require.NoError(t, err)

    // Connect metadata networks
    metadataInfo := peer.AddrInfo{
        ID:    engine1.GetMetadataHost().ID(),
        Addrs: []ma.Multiaddr{metadataAddr},
    }
    err = engine2.GetMetadataHost().Connect(ctx, metadataInfo)
    require.NoError(t, err)

    // Wait for DHT routing tables to be updated
    require.Eventually(t, func() bool {
        transportPeers := len(engine1.transportNode.dht.RoutingTable().ListPeers())
        metadataPeers := len(engine1.metadataNode.dht.RoutingTable().ListPeers())
        return transportPeers > 0 && metadataPeers > 0
    }, 5*time.Second, 100*time.Millisecond)

    // Wait for DHT to be ready and connections to be established
    time.Sleep(2 * time.Second)

    // Verify connections on both networks
    require.Eventually(t, func() bool {
        return engine1.transportNode.host.Network().Connectedness(engine2.GetTransportHost().ID()) == network.Connected &&
            engine1.metadataNode.host.Network().Connectedness(engine2.GetMetadataHost().ID()) == network.Connected
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
    Owner:           engine1.GetTransportHost().ID(),
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
engine1, err := NewNetworkEngine(ctx)
require.NoError(t, err)
defer engine1.Close()

// Create bootstrap nodes
bootstrapNodes := make([]*NetworkEngine, 3)
bootstrapAddrs := make([]ma.Multiaddr, 3)
for i := range bootstrapNodes {
node, err := NewNetworkEngine(ctx)
require.NoError(t, err)
defer node.Close()
bootstrapNodes[i] = node

addr := node.GetTransportHost().Addrs()[0]
peerID := node.GetTransportHost().ID()
p2pComponent, err := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", peerID.String()))
require.NoError(t, err)
bootstrapAddrs[i] = addr.Encapsulate(p2pComponent)
}

// Test bootstrapping
err = engine1.Bootstrap(bootstrapAddrs)
require.NoError(t, err)

// Verify connections
for _, node := range bootstrapNodes {
require.Eventually(t, func() bool {
return engine1.transportNode.host.Network().Connectedness(node.GetTransportHost().ID()) == network.Connected
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

invalidAddr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/1234/p2p/%s", id))
require.NoError(t, err)
err = engine1.Bootstrap([]ma.Multiaddr{invalidAddr})
assert.Error(t, err)
assert.Contains(t, err.Error(), "failed to connect to bootstrap peer")
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
    Owner:           engine1.GetTransportHost().ID(),
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

func TestNetworkFailures(t *testing.T) {
ctx := context.Background()
engine1, engine2 := setupTestNetwork(t, ctx)

manifest := &ManifestInfo{
    Name:            "test.txt",
    Size:            1024,
    ChunkHashes:     []string{"hash1"},
    Owner:           engine1.GetTransportHost().ID(),
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
engine3, err := NewNetworkEngine(ctx)
require.NoError(t, err)
defer engine3.Close()

_, _, err = engine3.GetZapFile(manifest.Name)
assert.Error(t, err, "should fail without connection to owner")

// Test cleanup
engine1.Close()
_, _, err = engine3.GetZapFile(manifest.Name)
assert.Error(t, err, "should fail after owner shutdown")
}

func TestNetworkEngineConnect(t *testing.T) {
    ctx := context.Background()
    engine1, engine2 := setupTestNetwork(t, ctx)
    defer engine1.Close()
    defer engine2.Close()

    addr := engine2.GetTransportHost().Addrs()[0]
    peerInfo := peer.AddrInfo{
        ID:    engine2.GetTransportHost().ID(),
        Addrs: []ma.Multiaddr{addr},
    }
    p2pComponent := fmt.Sprintf("/p2p/%s", peerInfo.ID.String())
    p2pAddr, err := ma.NewMultiaddr(p2pComponent)
    require.NoError(t, err)
    fullAddr := addr.Encapsulate(p2pAddr)

    err = engine1.Connect(fullAddr)
    require.NoError(t, err)

    require.Eventually(t, func() bool {
        return engine1.transportNode.host.Network().Connectedness(engine2.GetTransportHost().ID()) == network.Connected
    }, 5*time.Second, 100*time.Millisecond)

    require.Eventually(t, func() bool {
        return engine1.metadataNode.host.Network().Connectedness(engine2.GetMetadataHost().ID()) == network.Connected
    }, 5*time.Second, 100*time.Millisecond)

    t.Run("Invalid Address", func(t *testing.T) {
        invalidAddr, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/0")
        require.NoError(t, err)
        err = engine1.Connect(invalidAddr)
        assert.Error(t, err, "should fail when address doesn't include peer ID")
    })

    t.Run("Nonexistent Peer", func(t *testing.T) {
        randBytes := make([]byte, 32)
        _, err := rand.Read(randBytes)
        require.NoError(t, err)
        
        pubKey := &fakeKey{randBytes}
        id, err := peer.IDFromPublicKey(pubKey)
        require.NoError(t, err)
        
        invalidAddr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/1234/p2p/%s", id.String()))
        require.NoError(t, err)
        err = engine1.Connect(invalidAddr)
        assert.Error(t, err, "should fail when peer doesn't exist")
    })
}
