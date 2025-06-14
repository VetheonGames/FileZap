package network

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestHosts(t *testing.T) (host.Host, host.Host) {
	// Create two libp2p hosts for testing with TCP transport
	host1, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
	)
	require.NoError(t, err)

	host2, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
	)
	require.NoError(t, err)

	// Connect the hosts
	peerInfo := host2.Peerstore().PeerInfo(host2.ID())
	err = host1.Connect(context.Background(), peerInfo)
	require.NoError(t, err)

	// Wait for connection
	time.Sleep(time.Millisecond * 100)

	return host1, host2
}

func TestChunkStoreBasicOperations(t *testing.T) {
	// Create test hosts
	host1, host2 := setupTestHosts(t)
	defer host1.Close()
	defer host2.Close()

	// Create chunk stores
	store1 := NewChunkStore(host1)
	store2 := NewChunkStore(host2)

	// Test data
	testHash := "testhash"
	testData := []byte("test chunk data")

	// Store chunk in store1
	store1.Store(testHash, testData)

	// Verify local storage
	data, exists := store1.Get(testHash)
	require.True(t, exists)
	assert.Equal(t, testData, data)

	// Download chunk from store1 to store2
	downloadedData, err := store2.transfers.Download(host1.ID(), testHash)
	require.NoError(t, err)
	assert.Equal(t, testData, downloadedData)
}

func TestChunkStoreMultipleTransfers(t *testing.T) {
	host1, host2 := setupTestHosts(t)
	defer host1.Close()
	defer host2.Close()

	store1 := NewChunkStore(host1)
	store2 := NewChunkStore(host2)

	// Test multiple chunks
	chunks := map[string][]byte{
		"hash1": []byte("chunk data 1"),
		"hash2": []byte("chunk data 2"),
		"hash3": []byte("chunk data 3"),
	}

	// Store all chunks in store1
	for hash, data := range chunks {
		store1.Store(hash, data)
	}

	// Download and verify all chunks in store2
	for hash, expectedData := range chunks {
		downloadedData, err := store2.transfers.Download(host1.ID(), hash)
		require.NoError(t, err)
		assert.Equal(t, expectedData, downloadedData)
	}
}

func TestChunkStoreNonexistentChunk(t *testing.T) {
	host1, host2 := setupTestHosts(t)
	defer host1.Close()
	defer host2.Close()

	_ = NewChunkStore(host1)
	store2 := NewChunkStore(host2)

	// Try to download nonexistent chunk
	_, err := store2.transfers.Download(host1.ID(), "nonexistent")
	assert.Error(t, err)
}

func TestChunkStoreConcurrentTransfers(t *testing.T) {
	host1, host2 := setupTestHosts(t)
	defer host1.Close()
	defer host2.Close()

	store1 := NewChunkStore(host1)
	store2 := NewChunkStore(host2)

	// Create large test chunks
	chunks := make(map[string][]byte)
	for i := 0; i < 10; i++ {
		hash := fmt.Sprintf("hash%d", i)
		data := make([]byte, 1024*1024) // 1MB chunks
        if _, err := rand.Read(data); err != nil {
            t.Fatal(err)
        }
		chunks[hash] = data
		store1.Store(hash, data)
	}

	// Download chunks concurrently
	var wg sync.WaitGroup
	errors := make(chan error, len(chunks))

	for hash, expectedData := range chunks {
		wg.Add(1)
		go func(h string, expected []byte) {
			defer wg.Done()
			downloadedData, err := store2.transfers.Download(host1.ID(), h)
			if err != nil {
				errors <- err
				return
			}
			if !bytes.Equal(expected, downloadedData) {
				errors <- fmt.Errorf("data mismatch for chunk %s", h)
			}
		}(hash, expectedData)
	}

	// Wait for all transfers and check for errors
	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}
