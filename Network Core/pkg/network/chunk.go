package network

import (
"context"
"fmt"
"io"
"time"

"github.com/libp2p/go-libp2p/core/network"
"github.com/libp2p/go-libp2p/core/peer"
"github.com/libp2p/go-libp2p/core/protocol"
)

const chunkProtocol = "/filezap/chunk/1.0.0"


// isValidChunk validates chunk metadata
func isValidChunk(hash string, data []byte) bool {
    // Check for empty hash
    if len(hash) == 0 {
        return false
    }

    // Check for nil data
    if data == nil {
        return false
    }

    // Check if hash contains invalid UTF-8
    for _, r := range hash {
        if r == 0xFFFD { // Unicode replacement character
            return false
        }
    }

    return true
}

// Store stores a chunk in the local store
func (cs *ChunkStore) Store(hash string, data []byte) bool {
    if !isValidChunk(hash, data) {
        return false
    }

    cs.mu.Lock()
    defer cs.mu.Unlock()

    // Check chunk size limit
    if len(data) > maxChunkSize {
        return false
    }

    // Check if we need to evict chunks to make space
    for cs.totalSize+uint64(len(data)) > maxTotalSize && len(cs.chunks) > 0 {
        // Remove oldest chunk (first one we find)
        for oldHash, oldData := range cs.chunks {
            delete(cs.chunks, oldHash)
            cs.totalSize -= uint64(len(oldData))
            break
        }
    }

    // Store new chunk if we have space
    if cs.totalSize+uint64(len(data)) <= maxTotalSize {
        cs.chunks[hash] = data
        cs.totalSize += uint64(len(data))
        return true
    }

    return false
}

// Get retrieves a chunk from the local store
func (cs *ChunkStore) Get(hash string) ([]byte, bool) {
    cs.mu.RLock()
    defer cs.mu.RUnlock()
    data, ok := cs.chunks[hash]
    return data, ok
}

// Remove deletes a chunk from the store
func (cs *ChunkStore) Remove(hash string) {
    cs.mu.Lock()
    defer cs.mu.Unlock()

    if data, exists := cs.chunks[hash]; exists {
        cs.totalSize -= uint64(len(data))
        delete(cs.chunks, hash)
    }
}

// handleChunkStream handles incoming chunk requests
func (cs *ChunkStore) handleChunkStream(stream network.Stream) {
    defer func() {
        // Ensure stream is properly closed or reset on error
        if err := stream.Close(); err != nil {
            stream.Reset()
        }
    }()

    // Read chunk hash with timeout
    stream.SetDeadline(time.Now().Add(10 * time.Second))
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
        if _, err := stream.Write([]byte{0}); err != nil {
            stream.Reset()
            return
        }
        if _, err := stream.Write([]byte("chunk not found")); err != nil {
            stream.Reset()
            return
        }
        return
    }

    // Send success response (1 byte followed by data)
    if _, err := stream.Write([]byte{1}); err != nil {
        stream.Reset()
        return
    }

    // Send data in chunks to handle large files
    const chunkSize = 1024 * 1024 // 1MB chunks
    for i := 0; i < len(data); i += chunkSize {
        end := i + chunkSize
        if end > len(data) {
            end = len(data)
        }
        if _, err := stream.Write(data[i:end]); err != nil {
            stream.Reset()
            return
        }
    }
}


// Download downloads a chunk from a peer
func (tm *TransferManager) Download(from peer.ID, hash string) ([]byte, error) {
    if tm.host == nil {
        return nil, fmt.Errorf("transfer manager not initialized")
    }
    // Open stream to peer with timeout context
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Don't try to connect to self
    if from == tm.host.ID() {
        return nil, fmt.Errorf("cannot download from self")
    }

    // Create stream
    stream, err := tm.host.NewStream(ctx, from, protocol.ID(chunkProtocol))
    if err != nil {
        return nil, fmt.Errorf("failed to open stream: %w", err)
    }

    // Ensure stream cleanup
    defer func() {
        stream.Reset()
        stream.Close()
    }()

    // Set a short deadline for initial operations
    stream.SetDeadline(time.Now().Add(5 * time.Second))

    // Send chunk hash
    _, err = stream.Write([]byte(hash))
    if err != nil {
        return nil, fmt.Errorf("failed to send hash: %w", err)
    }

    // Read response status with timeout
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

    // Read chunk data with shorter timeouts to detect disconnections faster
    var data []byte
    buf := make([]byte, 1024*1024) // 1MB buffer
    for {
        // Set a shorter deadline for each read operation
        stream.SetDeadline(time.Now().Add(2 * time.Second))
        
        n, err := stream.Read(buf)
        if err == io.EOF {
            break
        }
        if err != nil {
            // Check for connection/stream errors
            if err.Error() == "stream reset" || 
               err.Error() == "connection reset" ||
               err.Error() == "deadline exceeded" ||
               err.Error() == "protocol not supported" ||
               tm.host.Network().Connectedness(from) != network.Connected {
                return nil, fmt.Errorf("connection closed during transfer")
            }
            return nil, fmt.Errorf("failed to read chunk: %w", err)
        }
        data = append(data, buf[:n]...)
    }

    return data, nil
}
