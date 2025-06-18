package api

import (
    "time"

    "github.com/libp2p/go-libp2p/core/peer"
)

// Config types

// NetworkConfig holds configuration for the network manager
type NetworkConfig struct {
    Transport     TransportConfig
    MetadataStore string
    ChunkCacheDir string
    VPNConfig    *VPNConfig
}

// TransportConfig holds configuration for network transport
type TransportConfig struct {
    ListenAddrs     []string
    ListenPort      int
    EnableQUIC      bool
    EnableTCP       bool
    EnableRelay     bool
    EnableAutoRelay bool
    EnableHolePunch bool
    QUICOpts        QUICOptions
}

// QUICOptions configures QUIC transport behavior
type QUICOptions struct {
    MaxStreams       uint32
    KeepAlivePeriod  time.Duration
    HandshakeTimeout time.Duration
    IdleTimeout      time.Duration
}

// VPNConfig holds configuration for VPN functionality
type VPNConfig struct {
    Enabled       bool
    NetworkCIDR   string
    InterfaceName string
    NetworkKey    string
}

// Data types

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

// StorageNodeInfo contains information about a storage node
type StorageNodeInfo struct {
    ID             string
    AvailableSpace int64
    Uptime         float64
    LastSeen       time.Time
    ChunksStored   []string
}

// PeerGossipInfo contains gossip information about a peer
type PeerGossipInfo struct {
    ID            peer.ID
    LastSeen      time.Time
    Uptime        float64
    ResponseTime  int64
}

// Enums and constants

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
