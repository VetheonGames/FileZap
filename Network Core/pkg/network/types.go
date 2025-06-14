package network

import (
    "context"
    "sync"
    "time"

    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/peer"
    dht "github.com/libp2p/go-libp2p-kad-dht"
    pubsub "github.com/libp2p/go-libp2p-pubsub"
    "github.com/quic-go/quic-go"
)

// ReplicationGoal represents the number of nodes that should store a manifest
const DefaultReplicationGoal = 5

// ManifestInfo represents a .zap file manifest with its metadata
type ManifestInfo struct {
	Name           string
	ChunkHashes    []string
	ReplicationGoal int
	Owner          peer.ID
	Size           int64
}

// NetworkEngine manages the P2P communication for FileZap
type NetworkEngine struct {
    ctx           context.Context
    transportNode *NetworkNode // For chunk transfer via QUIC/UDP
    metadataNode  *NetworkNode // For manifest sharing and health checks
    manifests     *ManifestManager
    chunkStore    *ChunkStore
}

// NetworkNode represents a libp2p node for either transport or metadata
type NetworkNode struct {
    host    host.Host
    dht     *dht.IpfsDHT
    pubsub  *pubsub.PubSub
    overlay *OverlayNetwork
}

// OverlayNetwork manages the logical network topology
type OverlayNetwork struct {
    node      *NetworkNode
    neighbors map[peer.ID]time.Time
    maxPeers  int
    mu        sync.RWMutex
}

// ManifestManager handles the storage and retrieval of .zap manifests
type ManifestManager struct {
	dht        *dht.IpfsDHT
	store      map[string]*ManifestInfo
	localNode  peer.ID
	topic      *pubsub.Topic
	replicator *ManifestReplicator
}

// ChunkStore handles the storage and transfer of file chunks using QUIC
type ChunkStore struct {
	chunks    map[string][]byte
	transfers *TransferManager
}

// TransferManager handles QUIC-based chunk transfers
type TransferManager struct {
    host     host.Host
    sessions map[peer.ID]*quic.Connection
    mu       sync.RWMutex
}

// ManifestReplicator ensures manifests meet their replication goals
type ManifestReplicator struct {
	dht       *dht.IpfsDHT
	manifests *ManifestManager
	interval  int
}
