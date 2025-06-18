package internal

import (
    "context"
    "fmt"
    "sync"

    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/peer"
    "github.com/libp2p/go-libp2p/core/protocol"
    "github.com/libp2p/go-libp2p/core/network"
)

const (
    chunkProtocol   = protocol.ID("/filezap/chunk/1.0.0")
    maxRequestSize  = 1024 * 1024 // 1MB
)

// chunkStore implements the ChunkStore interface
type chunkStore struct {
    chunks    map[string][]byte
    requests  map[string]*StorageRequest
    host      host.Host
    totalSize uint64
    maxSize   uint64
    ctx       context.Context
    mu        sync.RWMutex
}

func newChunkStore(ctx context.Context, h host.Host) *chunkStore {
    cs := &chunkStore{
        chunks:    make(map[string][]byte),
        requests:  make(map[string]*StorageRequest),
        host:      h,
        maxSize:   DefaultTotalSize,
        ctx:       ctx,
    }

    // Set up chunk request handler
    h.SetStreamHandler(chunkProtocol, cs.handleChunkRequest)

    return cs
}

func (cs *chunkStore) handleChunkRequest(stream network.Stream) {
    // Process incoming chunk request
    defer stream.Close()

    // Read request
    buf := make([]byte, maxRequestSize)
    _, err := stream.Read(buf)
    if err != nil {
        return
    }

    // TODO: Unmarshal request
    // var req ChunkRequest
    // if err := json.Unmarshal(buf[:n], &req); err != nil {
    //     return
    // }

    // TODO: Process chunk request (implement actual protocol)
    // 1. Validate request
    // 2. Fetch chunk
    // 3. Send response
}

// ChunkRequest represents a request for a chunk
type ChunkRequest struct {
    Hash string
    Size int64
}

// Implement ChunkStore interface

func (cs *chunkStore) Store(hash string, data []byte) bool {
    cs.mu.Lock()
    defer cs.mu.Unlock()

    // Check size limit
    if uint64(len(data))+cs.totalSize > cs.maxSize {
        return false
    }

    // Store chunk
    cs.chunks[hash] = data
    cs.totalSize += uint64(len(data))
    return true
}

func (cs *chunkStore) Get(hash string) ([]byte, bool) {
    cs.mu.RLock()
    data, exists := cs.chunks[hash]
    cs.mu.RUnlock()
    return data, exists
}

func (cs *chunkStore) Remove(hash string) {
    cs.mu.Lock()
    if data, exists := cs.chunks[hash]; exists {
        cs.totalSize -= uint64(len(data))
        delete(cs.chunks, hash)
    }
    cs.mu.Unlock()
}

func (cs *chunkStore) GetPendingRequest() (*StorageRequest, error) {
    cs.mu.Lock()
    defer cs.mu.Unlock()

    // Return first pending request
    for hash, req := range cs.requests {
        delete(cs.requests, hash)
        return req, nil
    }

    return nil, nil
}

// Additional helper methods

func (cs *chunkStore) downloadChunk(owner peer.ID, hash string) ([]byte, error) {
    // Open stream to chunk owner
    stream, err := cs.host.NewStream(cs.ctx, owner, chunkProtocol)
    if err != nil {
        return nil, fmt.Errorf("failed to open stream: %w", err)
    }
    defer stream.Close()

    // Send chunk request
    // TODO: Implement chunk download protocol

    return nil, fmt.Errorf("not implemented")
}

func (cs *chunkStore) verifyChunk(data []byte, hash string) bool {
    // TODO: Implement chunk verification
    return true
}

func (cs *chunkStore) isSpaceAvailable(size int64) bool {
    cs.mu.RLock()
    defer cs.mu.RUnlock()
    return uint64(size)+cs.totalSize <= cs.maxSize
}

func (cs *chunkStore) getTotalSize() uint64 {
    cs.mu.RLock()
    defer cs.mu.RUnlock()
    return cs.totalSize
}

func (cs *chunkStore) getStoredHashes() []string {
    cs.mu.RLock()
    defer cs.mu.RUnlock()

    hashes := make([]string, 0, len(cs.chunks))
    for hash := range cs.chunks {
        hashes = append(hashes, hash)
    }
    return hashes
}
