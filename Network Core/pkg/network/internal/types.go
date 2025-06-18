package internal

import (
    "time"

    "github.com/libp2p/go-libp2p/core/peer"
)

// ManifestInfo holds information about a manifest
type ManifestInfo struct {
    Name            string
    Owner           peer.ID
    ChunkHashes     []string
    Size            int64
    Created         time.Time
    Modified        time.Time
    ReplicationGoal int
    UpdatedAt       time.Time
}

// StorageRequest represents a chunk storage request
type StorageRequest struct {
    ChunkHash string
    Data      []byte
    Size      int64
    Owner     peer.ID
}

// PeerGossipInfo contains information about a peer
type PeerGossipInfo struct {
    ID            peer.ID
    LastSeen      time.Time
    Uptime        float64
    ResponseTime  int64
}

// StorageNodeInfo contains information about a storage node
type StorageNodeInfo struct {
    ID             string
    AvailableSpace int64
    Uptime         float64
    LastSeen       time.Time
    ChunksStored   []string
}

// VoteType represents the type of vote
type VoteType int
const (
    VoteRemovePeer VoteType = iota
    VoteRemoveFile
)

// ValidationResult represents chunk validation result
type ValidationResult int
const (
    ValidationSuccess ValidationResult = iota
    ValidationInvalidHash
    ValidationInvalidSignature
    ValidationInvalidSize
)

// Default storage values
const (
    DefaultMaxChunks      = 10000
    DefaultMaxChunkSize   = 1024 * 1024 // 1MB
    DefaultTotalSize      = 1024 * 1024 * 1024 * 10 // 10GB
    DefaultReplicationGoal = 3
)
