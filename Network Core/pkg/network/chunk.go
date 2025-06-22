package network

import (
    "bytes"
    "context"
    "crypto/sha256"
    "fmt"
    "io"
    "sync"
    "time"

    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/network"
    "github.com/libp2p/go-libp2p/core/peer"
    "github.com/libp2p/go-libp2p/core/protocol"
    quic "github.com/quic-go/quic-go"
)

// Protocol identifiers
const (
    chunkProtocol = "/filezap/chunk/1.0.0"
)

// ChunkStore manages chunk storage
type ChunkStore struct {
    host      host.Host
    chunks    map[string][]byte
    totalSize uint64
    transfers *TransferManager
    requests  chan *StorageRequest
    mu        sync.RWMutex
}

// TransferManager handles QUIC-based chunk transfers
type TransferManager struct {
    host     host.Host
    sessions map[peer.ID]*quic.Connection
    mu       sync.RWMutex
}

// NewTransferManager creates a new transfer manager
func NewTransferManager(host host.Host) *TransferManager {
    return &TransferManager{
        host:     host,
        sessions: make(map[peer.ID]*quic.Connection),
        mu:       sync.RWMutex{},
    }
}

// NewChunkStore creates a new chunk store
func NewChunkStore(host host.Host) *ChunkStore {
    cs := &ChunkStore{
        host:      host,
        chunks:    make(map[string][]byte),
        transfers: NewTransferManager(host),
        requests:  make(chan *StorageRequest, 100),
    }

    // Set up chunk protocol handler
    host.SetStreamHandler(protocol.ID(chunkProtocol), cs.handleChunkStream)
    return cs
}

// GetPendingRequest gets the next pending storage request
func (cs *ChunkStore) GetPendingRequest() (*StorageRequest, error) {
    select {
    case req := <-cs.requests:
        return req, nil
    default:
        return nil, ErrNoRequestsPending
    }
}

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

    // Don't try to connect to self
    if from == tm.host.ID() {
        return nil, fmt.Errorf("cannot download from self")
    }

    // Create stream
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

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

// ChunkValidationEvidence contains proof of chunk validation failure
type ChunkValidationEvidence struct {
    ChunkHash    string            `json:"chunk_hash"`
    Provider     peer.ID           `json:"provider"`
    FailureType  ValidationResult  `json:"failure_type"`
}

// Marshal converts evidence to bytes for network transmission
func (e *ChunkValidationEvidence) Marshal() ([]byte, error) {
    buf := bytes.NewBuffer(nil)
    
    // Write chunk hash
    if _, err := buf.WriteString(e.ChunkHash); err != nil {
        return nil, err
    }
    
    // Write provider ID
    if _, err := buf.Write([]byte(e.Provider)); err != nil {
        return nil, err
    }
    
    // Write failure type
    if err := buf.WriteByte(byte(e.FailureType)); err != nil {
        return nil, err
    }
    
    return buf.Bytes(), nil
}

// Unmarshal parses evidence from bytes
func (e *ChunkValidationEvidence) Unmarshal(data []byte) error {
    if len(data) < 65 { // Minimum length for hash(32) + peerID(32) + failureType(1)
        return fmt.Errorf("evidence data too short")
    }

    e.ChunkHash = string(data[:32])
    e.Provider = peer.ID(data[32:64])
    e.FailureType = ValidationResult(data[64])

    return nil
}

// ChunkValidator handles validation of chunk data
type ChunkValidator struct {
    ctx        context.Context
    quorum     QuorumManager
    store      *ChunkStore
    
    // Cache of recently validated chunks to prevent duplicate work
    cache      map[string]ValidationResult
    cacheSize  int
    mu         sync.RWMutex
}

// NewChunkValidator creates a new chunk validation system
func NewChunkValidator(ctx context.Context, quorum QuorumManager, store *ChunkStore) *ChunkValidator {
    return &ChunkValidator{
        ctx:       ctx,
        quorum:    quorum,
        store:     store,
        cache:     make(map[string]ValidationResult),
        cacheSize: 1000, // Cache size limit
    }
}

// ValidateChunk checks if a chunk is valid and reports bad actors
func (cv *ChunkValidator) ValidateChunk(chunk []byte, expectedHash string, provider peer.ID) ValidationResult {
    // Check cache first
    if result, ok := cv.getCachedResult(expectedHash); ok {
        return result
    }

    // Validate chunk hash
    actualHash := cv.calculateHash(chunk)
    if actualHash != expectedHash {
        cv.reportBadChunk(provider, expectedHash, ValidationHashMismatch)
        cv.cacheResult(expectedHash, ValidationHashMismatch)
        return ValidationHashMismatch
    }

    // Validate chunk size
    if !cv.validateChunkSize(chunk) {
        cv.reportBadChunk(provider, expectedHash, ValidationSizeMismatch)
        cv.cacheResult(expectedHash, ValidationSizeMismatch)
        return ValidationSizeMismatch
    }

    // Validate chunk content format
    if !cv.validateChunkFormat(chunk) {
        cv.reportBadChunk(provider, expectedHash, ValidationContentMalformed)
        cv.cacheResult(expectedHash, ValidationContentMalformed)
        return ValidationContentMalformed
    }

    // Cache successful validation
    cv.cacheResult(expectedHash, ValidationSuccess)
    return ValidationSuccess
}

// calculateHash generates the SHA-256 hash of chunk data
func (cv *ChunkValidator) calculateHash(chunk []byte) string {
    hash := sha256.Sum256(chunk)
    return fmt.Sprintf("%x", hash[:])
}

// validateChunkSize checks if the chunk size is within acceptable limits
func (cv *ChunkValidator) validateChunkSize(chunk []byte) bool {
    // Ensure chunk is not empty and not too large (100MB max)
    return len(chunk) > 0 && int64(len(chunk)) <= maxChunkSize
}

// validateChunkFormat verifies the chunk content structure
func (cv *ChunkValidator) validateChunkFormat(chunk []byte) bool {
    // Basic format validation
    // - First byte should be version number (currently 1)
    // - Next 4 bytes should be sequence number
    // - Remaining bytes are payload
    if len(chunk) < 5 {
        return false
    }

    version := chunk[0]
    if version != 1 {
        return false
    }

    // Additional format checks can be added here
    return true
}

// reportBadChunk notifies the quorum of a bad chunk provider
func (cv *ChunkValidator) reportBadChunk(provider peer.ID, hash string, reason ValidationResult) {
    evidence := &ChunkValidationEvidence{
        ChunkHash:    hash,
        Provider:     provider,
        FailureType:  reason,
    }

    evidenceBytes, err := evidence.Marshal()
    if err != nil {
        return
    }

    // Propose vote to remove peer if they provide bad chunks
    reasonStr := fmt.Sprintf("Provided invalid chunk: %s (Reason: %d)", hash, reason)
    if err := cv.quorum.ProposeVote(VoteRemovePeer, string(provider), reasonStr, evidenceBytes); err != nil {
        return
    }

    // Update peer reputation
    cv.quorum.UpdatePeerReputation(provider, -10) // Significant reputation penalty
}

// getCachedResult retrieves a cached validation result
func (cv *ChunkValidator) getCachedResult(hash string) (ValidationResult, bool) {
    cv.mu.RLock()
    defer cv.mu.RUnlock()
    result, ok := cv.cache[hash]
    return result, ok
}

// cacheResult stores a validation result
func (cv *ChunkValidator) cacheResult(hash string, result ValidationResult) {
    cv.mu.Lock()
    defer cv.mu.Unlock()

    // Remove oldest entry if cache is full
    if len(cv.cache) >= cv.cacheSize {
        // Simple approach: clear half the cache
        newSize := cv.cacheSize / 2
        newCache := make(map[string]ValidationResult, cv.cacheSize)
        count := 0
        for k, v := range cv.cache {
            if count >= newSize {
                break
            }
            newCache[k] = v
            count++
        }
        cv.cache = newCache
    }

    cv.cache[hash] = result
}
