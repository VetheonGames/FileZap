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
