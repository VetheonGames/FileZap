package network

import (
    "context"
    "sync"
    "time"

    dht "github.com/libp2p/go-libp2p-kad-dht"
    pubsub "github.com/libp2p/go-libp2p-pubsub"
    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/peer"
)

// StorageRequest represents a request to store a chunk
type StorageRequest struct {
    ChunkHash string    // Hash of the chunk
    Data      []byte    // Chunk data
    Timestamp time.Time // When the request was made
    Requester string    // ID of requesting node
    Size      int64     // Size of chunk
}

// StorageNodeInfo tracks a node's storage capabilities
type StorageNodeInfo struct {
    ID            string
    AvailableSpace int64
    Uptime        float64
    LastSeen      time.Time
    ChunksStored  []string
}

// ReplicationGoal represents the number of nodes that should store a manifest
const DefaultReplicationGoal = 5

// ManifestInfo represents a .zap file manifest with its metadata
type ManifestInfo struct {
Name            string
ChunkHashes     []string
ReplicationGoal int
Owner           peer.ID
Size            int64
UpdatedAt       time.Time
}

// NetworkEngine manages the P2P communication for FileZap
type NetworkEngine struct {
    ctx          context.Context
    cancel       context.CancelFunc
    mu           sync.RWMutex

    // Network components
    transportNode *NetworkNode // For chunk transfer via QUIC/UDP
    metadataNode  *NetworkNode // For manifest sharing and health checks
    gossipMgr     *GossipManager
    quorum        *QuorumManager
    validator     *ChunkValidator
    
    // Data management
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
    host       host.Host
    chunks     map[string][]byte
    transfers  *TransferManager
    totalSize  uint64
    mu         sync.RWMutex
    
    // Storage request handling
    requests   chan *StorageRequest
}

// ManifestReplicator ensures manifests meet their replication goals
type ManifestReplicator struct {
	dht       *dht.IpfsDHT
	manifests *ManifestManager
	interval  int
}
