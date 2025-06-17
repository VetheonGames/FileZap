package network

import (
    "bytes"
    "context"
    "crypto/sha256"
    "fmt"
    "sync"

    "github.com/libp2p/go-libp2p/core/peer"
)

// ValidationResult represents the outcome of chunk validation
type ValidationResult int

const (
    // ValidationSuccess indicates the chunk is valid
    ValidationSuccess ValidationResult = iota
    // ValidationHashMismatch indicates the chunk hash doesn't match expected
    ValidationHashMismatch
    // ValidationSizeMismatch indicates the chunk size is incorrect
    ValidationSizeMismatch
    // ValidationContentMalformed indicates the chunk content is malformed
    ValidationContentMalformed
)

// ChunkValidator handles validation of chunk data
type ChunkValidator struct {
    ctx        context.Context
    quorum     *QuorumManager
    store      *ChunkStore
    
    // Cache of recently validated chunks to prevent duplicate work
    cache      map[string]ValidationResult
    cacheSize  int
    mu         sync.RWMutex
}

// NewChunkValidator creates a new chunk validation system
func NewChunkValidator(ctx context.Context, quorum *QuorumManager, store *ChunkStore) *ChunkValidator {
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
    maxSize := int64(100 * 1024 * 1024)
    return len(chunk) > 0 && int64(len(chunk)) <= maxSize
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
    cv.quorum.ProposeVote(VoteRemovePeer, string(provider), reasonStr, evidenceBytes)

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
