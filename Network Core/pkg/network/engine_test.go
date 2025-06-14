    package network

import (
"context"
"testing"
"time"

"github.com/libp2p/go-libp2p"
"github.com/libp2p/go-libp2p/core/network"
"github.com/libp2p/go-libp2p/core/peer"
"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNetworkEngine(t *testing.T) {
ctx := context.Background()

// Create a bootstrap node for testing
bootstrapNode, err := libp2p.New(
    libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0", "/ip4/127.0.0.1/udp/0/quic"),
)
require.NoError(t, err)
defer bootstrapNode.Close()

engine, err := NewNetworkEngine(ctx)
require.NoError(t, err)
	defer engine.Close()

	// Verify both nodes were created
	require.NotNil(t, engine.transportNode)
	require.NotNil(t, engine.metadataNode)
	
	// Verify components are initialized
	require.NotNil(t, engine.manifests)
	require.NotNil(t, engine.chunkStore)

	// Verify nodes have different addresses
	tAddrs := engine.transportNode.host.Addrs()
	mAddrs := engine.metadataNode.host.Addrs()
	require.NotEmpty(t, tAddrs)
	require.NotEmpty(t, mAddrs)
	require.NotEqual(t, tAddrs[0], mAddrs[0])
}

func TestNetworkEngineConnect(t *testing.T) {
	ctx := context.Background()
	
	// Create two engines
// Create bootstrap node
bootstrapNode, err := libp2p.New(
    libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0", "/ip4/127.0.0.1/udp/0/quic"),
)
require.NoError(t, err)
defer bootstrapNode.Close()

engine1, err := NewNetworkEngine(ctx)
require.NoError(t, err)
defer engine1.Close()

engine2, err := NewNetworkEngine(ctx)
	require.NoError(t, err)
	defer engine2.Close()

	// Get multiaddr from engine2
	addr := engine2.GetTransportHost().Addrs()[0]
	peerInfo := peer.AddrInfo{
		ID:    engine2.GetTransportHost().ID(),
		Addrs: []multiaddr.Multiaddr{addr},
	}
	multiaddr, err := multiaddr.NewMultiaddr("/p2p/" + peerInfo.ID.String())
	require.NoError(t, err)
	fullAddr := addr.Encapsulate(multiaddr)

	// Connect engine1 to engine2
	err = engine1.Connect(fullAddr)
	require.NoError(t, err)

	// Verify connection on both networks
require.Eventually(t, func() bool {
return engine1.transportNode.host.Network().Connectedness(engine2.GetTransportHost().ID()) == network.Connected
}, 5*time.Second, 100*time.Millisecond)

require.Eventually(t, func() bool {
return engine1.metadataNode.host.Network().Connectedness(engine2.GetMetadataHost().ID()) == network.Connected
}, 5*time.Second, 100*time.Millisecond)
}

func TestZapFileOperations(t *testing.T) {
ctx := context.Background()

// Create bootstrap node
bootstrapNode, err := libp2p.New(
    libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0", "/ip4/127.0.0.1/udp/0/quic"),
)
require.NoError(t, err)
defer bootstrapNode.Close()

// Create two engines
engine1, err := NewNetworkEngine(ctx)
	require.NoError(t, err)
	defer engine1.Close()

	engine2, err := NewNetworkEngine(ctx)
	require.NoError(t, err)
	defer engine2.Close()

	// Connect the engines
	addr := engine2.GetTransportHost().Addrs()[0]
	peerInfo := peer.AddrInfo{
		ID:    engine2.GetTransportHost().ID(),
		Addrs: []multiaddr.Multiaddr{addr},
	}
	multiaddr, err := multiaddr.NewMultiaddr("/p2p/" + peerInfo.ID.String())
	require.NoError(t, err)
	fullAddr := addr.Encapsulate(multiaddr)
	err = engine1.Connect(fullAddr)
	require.NoError(t, err)

	// Create test data
	testManifest := &ManifestInfo{
		Name:           "test.zap",
		ChunkHashes:    []string{"hash1", "hash2"},
		ReplicationGoal: DefaultReplicationGoal,
		Owner:          engine1.GetTransportHost().ID(),
		Size:           1024,
	}

	testChunks := map[string][]byte{
		"hash1": []byte("test data 1"),
		"hash2": []byte("test data 2"),
	}

	// Add file to network through engine1
	err = engine1.AddZapFile(testManifest, testChunks)
	require.NoError(t, err)

	// Try to retrieve file through engine2
	retrievedManifest, retrievedChunks, err := engine2.GetZapFile("test.zap")
	require.NoError(t, err)
	require.NotNil(t, retrievedManifest)
	require.NotNil(t, retrievedChunks)

	// Verify retrieved data
	assert.Equal(t, testManifest.Name, retrievedManifest.Name)
	assert.Equal(t, testManifest.ChunkHashes, retrievedManifest.ChunkHashes)
	assert.Equal(t, testManifest.Size, retrievedManifest.Size)
	assert.Equal(t, len(testChunks), len(retrievedChunks))
	
	for hash, data := range testChunks {
		retrievedData, exists := retrievedChunks[hash]
		assert.True(t, exists)
		assert.Equal(t, data, retrievedData)
	}
}
