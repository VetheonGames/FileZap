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
    "github.com/libp2p/go-libp2p/core/peer"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func setupTestHosts(t *testing.T) (host.Host, host.Host) {
// Create two libp2p hosts for testing with TCP transport
host1, err := libp2p.New(
libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
libp2p.DefaultTransports,
)
if err != nil {
t.Fatal(err)
}

host2, err := libp2p.New(
libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
libp2p.DefaultTransports,
)
if err != nil {
t.Fatal(err)
}

// Get host information
h1Info := peer.AddrInfo{
ID:    host1.ID(),
Addrs: host1.Addrs(),
}
h2Info := peer.AddrInfo{
ID:    host2.ID(),
Addrs: host2.Addrs(),
}

// Add peers to peerstore
host1.Peerstore().AddAddrs(h2Info.ID, h2Info.Addrs, time.Hour)
host2.Peerstore().AddAddrs(h1Info.ID, h1Info.Addrs, time.Hour)

// Connect the hosts
err = host1.Connect(context.Background(), h2Info)
if err != nil {
t.Fatal(err)
}

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

store1 := NewChunkStore(host1)
store2 := NewChunkStore(host2)

// Test data to ensure connectivity works
testHash := "testhash"
testData := []byte("test data")
store1.Store(testHash, testData)

// Try to download nonexistent chunk
_, err := store2.transfers.Download(host1.ID(), "nonexistent")
assert.Error(t, err, "should fail when chunk does not exist")

// Verify the existing chunk can still be downloaded
data, err := store2.transfers.Download(host1.ID(), testHash)
require.NoError(t, err)
assert.Equal(t, testData, data)
}

func TestChunkStoreNetworkFailures(t *testing.T) {
host1, host2 := setupTestHosts(t)
defer host1.Close()
defer host2.Close()

store1 := NewChunkStore(host1)
store2 := NewChunkStore(host2)

// Store test chunk
testHash := "testhash"
testData := []byte("test data")
store1.Store(testHash, testData)

// Test disconnection during transfer
bigData := make([]byte, 10*1024*1024) // 10MB chunk
rand.Read(bigData)
store1.Store("bighash", bigData)

// Start download and immediately close connection
go func() {
time.Sleep(100 * time.Millisecond)
host1.Network().ClosePeer(host2.ID())
}()

_, err := store2.transfers.Download(host1.ID(), "bighash")
assert.Error(t, err, "should fail when connection is closed")

// Reconnect hosts
err = host1.Connect(context.Background(), peer.AddrInfo{
ID:    host2.ID(),
Addrs: host2.Addrs(),
})
require.NoError(t, err)
time.Sleep(100 * time.Millisecond)

// Verify normal transfer still works
data, err := store2.transfers.Download(host1.ID(), testHash)
require.NoError(t, err)
assert.Equal(t, testData, data)
}

func TestChunkStoreLimits(t *testing.T) {
host1, _ := setupTestHosts(t)
defer host1.Close()

store := NewChunkStore(host1)

// Test memory limits
bigChunk := make([]byte, 100*1024*1024) // 100MB chunk
rand.Read(bigChunk)
store.Store("bighash", bigChunk)

// Try to store more chunks than the memory limit
for i := 0; i < 20; i++ { // Try to store 2GB total
data := make([]byte, 100*1024*1024)
rand.Read(data)
store.Store(fmt.Sprintf("hash%d", i), data)
}

// Verify older chunks are evicted
_, exists := store.Get("bighash")
assert.False(t, exists, "first chunk should be evicted")

// Test chunk size limit
hugeChunk := make([]byte, 1024*1024*1024) // 1GB chunk
rand.Read(hugeChunk)
store.Store("hugehash", hugeChunk)
_, exists = store.Get("hugehash")
assert.False(t, exists, "oversized chunk should not be stored")
}

func TestChunkTransferInterruption(t *testing.T) {
host1, host2 := setupTestHosts(t)
defer host1.Close()
defer host2.Close()

store1 := NewChunkStore(host1)
store2 := NewChunkStore(host2)

// Create large chunk
data := make([]byte, 10*1024*1024) // 10MB
rand.Read(data)
store1.Store("largehash", data)

// Start multiple concurrent downloads and interrupt them
var wg sync.WaitGroup
errors := make(chan error, 5)

// Start downloads
for i := 0; i < 5; i++ {
wg.Add(1)
go func() {
defer wg.Done()

// Start download and interrupt by closing connection
go func() {
time.Sleep(50 * time.Millisecond)
host1.Network().ClosePeer(host2.ID())
}()

_, err := store2.transfers.Download(host1.ID(), "largehash")
if err == nil {
errors <- fmt.Errorf("expected error on interrupted transfer")
}
}()
}

// Wait for all attempts
wg.Wait()
close(errors)

// Check for expected failures
for err := range errors {
t.Error(err)
}

// Reconnect hosts for final test
connectErr := host1.Connect(context.Background(), peer.AddrInfo{
ID:    host2.ID(),
Addrs: host2.Addrs(),
})
require.NoError(t, connectErr)
time.Sleep(100 * time.Millisecond)

// Verify chunk can still be downloaded normally
downloadedData, err := store2.transfers.Download(host1.ID(), "largehash")
require.NoError(t, err)
assert.Equal(t, data, downloadedData)
}

func TestInvalidChunkMetadata(t *testing.T) {
host1, host2 := setupTestHosts(t)
defer host1.Close()
defer host2.Close()

store1 := NewChunkStore(host1)
store2 := NewChunkStore(host2)

// Test with invalid hash format
invalidHash := string([]byte{0xFF, 0xFE, 0xFD}) // Invalid UTF-8
store1.Store(invalidHash, []byte("test"))
_, err := store2.transfers.Download(host1.ID(), invalidHash)
assert.Error(t, err)

// Test with empty hash
store1.Store("", []byte("test"))
_, err = store2.transfers.Download(host1.ID(), "")
assert.Error(t, err)

// Test with nil data
store1.Store("nilhash", nil)
_, err = store2.transfers.Download(host1.ID(), "nilhash")
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
