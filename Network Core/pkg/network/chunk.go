package network

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/quic-go/quic-go"
)

const chunkProtocol = "/filezap/chunk/1.0.0"

// NewChunkStore creates a new chunk store
func NewChunkStore(host host.Host) *ChunkStore {
	cs := &ChunkStore{
		chunks:    make(map[string][]byte),
		transfers: NewTransferManager(host),
	}

	// Register chunk protocol handler
	host.SetStreamHandler(protocol.ID(chunkProtocol), cs.handleChunkStream)

	return cs
}

// Store stores a chunk in the local store
func (cs *ChunkStore) Store(hash string, data []byte) {
	cs.chunks[hash] = data
}

// Get retrieves a chunk from the local store
func (cs *ChunkStore) Get(hash string) ([]byte, bool) {
	data, ok := cs.chunks[hash]
	return data, ok
}

// handleChunkStream handles incoming chunk requests
func (cs *ChunkStore) handleChunkStream(stream network.Stream) {
    defer stream.Close()

    // Read chunk hash
    buf := make([]byte, 64)
    n, err := stream.Read(buf)
    if err != nil {
        stream.Reset()
        return
    }
    hash := string(buf[:n])

    // Get chunk data
    data, ok := cs.Get(hash)
    if !ok {
        // Send error response (0 byte followed by error message)
        stream.Write([]byte{0})
        stream.Write([]byte("chunk not found"))
        return
    }

    // Send success response (1 byte followed by data)
    stream.Write([]byte{1})
    _, err = stream.Write(data)
    if err != nil {
        stream.Reset()
        return
    }
}

// NewTransferManager creates a new transfer manager
func NewTransferManager(host host.Host) *TransferManager {
	return &TransferManager{
		host:     host,
		sessions: make(map[peer.ID]*quic.Connection),
		mu:       sync.RWMutex{},
	}
}

// Download downloads a chunk from a peer
func (tm *TransferManager) Download(from peer.ID, hash string) ([]byte, error) {
	// Open stream to peer
	stream, err := tm.host.NewStream(context.Background(), from, protocol.ID(chunkProtocol))
	if err != nil {
		return nil, fmt.Errorf("failed to open stream: %w", err)
	}
	defer stream.Close()

	// Send chunk hash
	_, err = stream.Write([]byte(hash))
	if err != nil {
		return nil, fmt.Errorf("failed to send hash: %w", err)
	}

// Read response status
status := make([]byte, 1)
_, err = stream.Read(status)
if err != nil {
    return nil, fmt.Errorf("failed to read status: %w", err)
}

// Check status
if status[0] == 0 {
    // Error response
    errMsg, _ := io.ReadAll(stream)
    return nil, fmt.Errorf("chunk retrieval failed: %s", string(errMsg))
}

// Read chunk data
data, err := io.ReadAll(stream)
if err != nil {
    return nil, fmt.Errorf("failed to read chunk: %w", err)
}

return data, nil
}
