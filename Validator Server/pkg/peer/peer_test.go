package peer

import (
    "fmt"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestPeerManager(t *testing.T) {
    t.Run("new peer manager", func(t *testing.T) {
        pm := NewManager(5 * time.Minute)
        assert.NotNil(t, pm)
    })

    t.Run("invalid expiry duration", func(t *testing.T) {
        pm := NewManager(0)
        assert.Nil(t, pm)

        pm = NewManager(-time.Minute)
        assert.Nil(t, pm)
    })
}

func TestPeerOperations(t *testing.T) {
    pm := NewManager(5 * time.Minute)
    require.NotNil(t, pm)

    t.Run("add and get peer", func(t *testing.T) {
        pm.UpdatePeer("peer1", "localhost:8081", []string{"zap1", "zap2"})

        // Verify peer was added
        peer, exists := pm.GetPeer("peer1")
        assert.True(t, exists)
        assert.Equal(t, "peer1", peer.ID)
        assert.Equal(t, "localhost:8081", peer.Address)
        assert.ElementsMatch(t, []string{"zap1", "zap2"}, peer.AvailableZaps)
        assert.False(t, peer.LastSeen.IsZero())
    })

    t.Run("update existing peer", func(t *testing.T) {
        // Update with new info
        pm.UpdatePeer("peer1", "localhost:8082", []string{"zap2", "zap3"})

        peer, exists := pm.GetPeer("peer1")
        assert.True(t, exists)
        assert.Equal(t, "localhost:8082", peer.Address)
        assert.ElementsMatch(t, []string{"zap2", "zap3"}, peer.AvailableZaps)
    })

    t.Run("nonexistent peer", func(t *testing.T) {
        _, exists := pm.GetPeer("nonexistent")
        assert.False(t, exists)
    })

    t.Run("get all peers", func(t *testing.T) {
        // Add another peer
        pm.UpdatePeer("peer2", "localhost:8083", []string{"zap4"})

        peers := pm.GetAllPeers()
        assert.Equal(t, 2, len(peers))

        // Verify both peers are present
        var peerIDs []string
        for _, p := range peers {
            peerIDs = append(peerIDs, p.ID)
        }
        assert.ElementsMatch(t, []string{"peer1", "peer2"}, peerIDs)
    })
}

func TestPeerExpiry(t *testing.T) {
    // Use short expiry for testing
    pm := NewManager(100 * time.Millisecond)
    require.NotNil(t, pm)

    t.Run("peer expiration", func(t *testing.T) {
        // Add a peer
        pm.UpdatePeer("peer1", "localhost:8081", []string{"zap1"})

        // Initially peer should exist
        _, exists := pm.GetPeer("peer1")
        assert.True(t, exists)

        // Wait longer than expiry duration
        time.Sleep(200 * time.Millisecond)

        // Peer should be automatically removed after expiry
        _, exists = pm.GetPeer("peer1")
        assert.False(t, exists)
    })

    t.Run("peer refresh", func(t *testing.T) {
        // Add a peer
        pm.UpdatePeer("peer2", "localhost:8082", []string{"zap2"})

        // Update it just before expiry
        time.Sleep(50 * time.Millisecond)
        pm.UpdatePeer("peer2", "localhost:8082", []string{"zap2"})

        // Wait some more, but not enough for the refresh to expire
        time.Sleep(75 * time.Millisecond)

        // Peer should still exist
        _, exists := pm.GetPeer("peer2")
        assert.True(t, exists)
    })
}

func TestZapAvailability(t *testing.T) {
    pm := NewManager(5 * time.Minute)
    require.NotNil(t, pm)

    t.Run("find peers for zap", func(t *testing.T) {
        // Add peers with different zap availability
        pm.UpdatePeer("peer1", "localhost:8081", []string{"zap1", "zap2"})
        pm.UpdatePeer("peer2", "localhost:8082", []string{"zap2", "zap3"})
        pm.UpdatePeer("peer3", "localhost:8083", []string{"zap3", "zap4"})

        // Check zap1 (only peer1 has it)
        peers := pm.GetPeersWithZap("zap1")
        assert.Equal(t, 1, len(peers))
        assert.Equal(t, "peer1", peers[0].ID)

        // Check zap2 (peer1 and peer2 have it)
        peers = pm.GetPeersWithZap("zap2")
        assert.Equal(t, 2, len(peers))
        var peerIDs []string
        for _, p := range peers {
            peerIDs = append(peerIDs, p.ID)
        }
        assert.ElementsMatch(t, []string{"peer1", "peer2"}, peerIDs)

        // Check nonexistent zap
        peers = pm.GetPeersWithZap("nonexistent")
        assert.Empty(t, peers)
    })

    t.Run("update zap availability", func(t *testing.T) {
        // Update peer1's available zaps
        pm.UpdatePeer("peer1", "localhost:8081", []string{"zap2", "zap3"})

        // Should no longer have zap1
        peers := pm.GetPeersWithZap("zap1")
        assert.Empty(t, peers)

        // Should still have zap2
        peers = pm.GetPeersWithZap("zap2")
        assert.Equal(t, 2, len(peers))
    })
}

func TestConcurrentPeerOperations(t *testing.T) {
    pm := NewManager(5 * time.Minute)
    require.NotNil(t, pm)

    t.Run("concurrent updates", func(t *testing.T) {
        done := make(chan bool)
        const numGoroutines = 10

        // Perform concurrent updates
        for i := 0; i < numGoroutines; i++ {
            go func(id int) {
                peerID := fmt.Sprintf("peer%d", id)
                address := fmt.Sprintf("localhost:%d", 8080+id)
                zaps := []string{fmt.Sprintf("zap%d", id)}
                
                pm.UpdatePeer(peerID, address, zaps)
                done <- true
            }(i)
        }

        // Wait for all updates
        for i := 0; i < numGoroutines; i++ {
            <-done
        }

        // Verify all peers were added
        peers := pm.GetAllPeers()
        assert.Equal(t, numGoroutines, len(peers))

        // Verify each peer's data
        for i := 0; i < numGoroutines; i++ {
            peerID := fmt.Sprintf("peer%d", i)
            peer, exists := pm.GetPeer(peerID)
            assert.True(t, exists)
            assert.Equal(t, fmt.Sprintf("localhost:%d", 8080+i), peer.Address)
            assert.ElementsMatch(t, []string{fmt.Sprintf("zap%d", i)}, peer.AvailableZaps)
        }
    })

    t.Run("concurrent reads and updates", func(t *testing.T) {
        done := make(chan bool)
        const numReaders = 5
        const numWriters = 5

        // Start readers
        for i := 0; i < numReaders; i++ {
            go func() {
                for j := 0; j < 10; j++ {
                    peers := pm.GetAllPeers()
                    assert.NotEmpty(t, peers)
                    time.Sleep(time.Millisecond)
                }
                done <- true
            }()
        }

        // Start writers
        for i := 0; i < numWriters; i++ {
            go func(id int) {
                for j := 0; j < 10; j++ {
                    peerID := fmt.Sprintf("peer%d", id)
                    zaps := []string{fmt.Sprintf("zap%d_%d", id, j)}
                    pm.UpdatePeer(peerID, fmt.Sprintf("localhost:%d", 8080+id), zaps)
                    time.Sleep(time.Millisecond)
                }
                done <- true
            }(i)
        }

        // Wait for all goroutines
        for i := 0; i < numReaders+numWriters; i++ {
            <-done
        }

        // Verify final state
        peers := pm.GetAllPeers()
        assert.NotEmpty(t, peers)
    })
}
