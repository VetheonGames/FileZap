package network

import (
    "sync"
    "time"

    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/peer"
    dht "github.com/libp2p/go-libp2p-kad-dht"
    pubsub "github.com/libp2p/go-libp2p-pubsub"

    "github.com/VetheonGames/FileZap/NetworkCore/pkg/vpn"
)

// Basic configuration types

type NetworkManagerConfig struct {
    Transport     TransportConfig
    MetadataStore string
    ChunkCacheDir string
    VPN          *VPNConfig
}

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

type QUICOptions struct {
    MaxStreams       uint32
    KeepAlivePeriod  time.Duration
    HandshakeTimeout time.Duration
    IdleTimeout      time.Duration
}

// Implementation types

type networkNode struct {
    host     host.Host
    dht      *dht.IpfsDHT
    pubsub   *pubsub.PubSub
    overlay  *overlayNetwork
}

type overlayNetwork struct {
    neighbors map[peer.ID]time.Time
    maxPeers int
    mu       sync.RWMutex
}

type networkEngine struct {
    ctx           context.Context
    cancel        context.CancelFunc
    transportNode *networkNode
    metadataNode  *networkNode
    gossipMgr     GossipManager
    quorum        *quorumManager
    validator     Validator
    manifests     ManifestManager
    chunkStore    ChunkStore
    vpnManager    *vpn.VPNManager
    vpnDiscovery  *vpn.Discovery
    mu            sync.RWMutex
}

// Data types

type StorageRequest struct {
    ChunkHash string
    Data      []byte
    Size      int64
    Owner     peer.ID
}

type StorageNodeInfo struct {
    ID             string
    AvailableSpace int64
    Uptime         float64
    LastSeen       time.Time
    ChunksStored   []string
}

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

type PeerGossipInfo struct {
    ID            peer.ID
    LastSeen      time.Time
    Uptime        float64
    ResponseTime  int64
}

// Enums and constants

type VoteType int
const (
    VoteRemovePeer VoteType = iota
    VoteRemoveFile
)

type ValidationResult int
const (
    ValidationSuccess ValidationResult = iota
    ValidationInvalidHash
    ValidationInvalidSignature
    ValidationInvalidSize
)
