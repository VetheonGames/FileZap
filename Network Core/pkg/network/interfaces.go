package network

import (
    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/peer"
    ma "github.com/multiformats/go-multiaddr"

    "github.com/VetheonGames/FileZap/NetworkCore/pkg/vpn"
)

// NetworkEngine defines the main network engine interface
type NetworkEngine interface {
    Connect(addr ma.Multiaddr) error
    Close() error
    GetNodeID() string
    GetPeers() []peer.ID
    GetTransportHost() host.Host
    GetMetadataHost() host.Host
    GetVPNManager() *vpn.VPNManager
    Bootstrap(addrs []peer.AddrInfo) error

    // File operations
    AddZapFile(manifest *ManifestInfo, chunks map[string][]byte) error
    GetZapFile(name string) (*ManifestInfo, map[string][]byte, error)
    ReportBadFile(name string, reason string) error

    // Storage operations
    RegisterStorageNode() error
    UnregisterStorageNode() error
    GetStorageRequest() (*StorageRequest, error)
    ValidateChunkRequest(req *StorageRequest) error
    StoreChunk(req *StorageRequest) error
    RejectStorageRequest(req *StorageRequest, reason string) error
    AcknowledgeStorage(req *StorageRequest) error
}

// NetworkNode represents a network node with transport capabilities
type NetworkNode interface {
    Close() error
    GetHost() host.Host
    GetDHT() DHT
    GetPubSub() PubSub
}

// DHT represents a distributed hash table
type DHT interface {
    Bootstrap(ctx Context) error
    Close() error
}

// PubSub represents a publish-subscribe system
type PubSub interface {
    Publish(topic string, data []byte) error
    Subscribe(topic string) (Subscription, error)
}

// Subscription represents a pubsub subscription
type Subscription interface {
    Next() (Message, error)
    Cancel()
}

// Message represents a pubsub message
type Message interface {
    Data() []byte
    From() peer.ID
    Topics() []string
}

// Validator represents a chunk validator
type Validator interface {
    ValidateChunk(data []byte, hash string, owner peer.ID) ValidationResult
}

// ChunkStore represents a chunk storage interface
type ChunkStore interface {
    Store(hash string, data []byte) bool
    Get(hash string) ([]byte, bool)
    Remove(hash string)
    GetPendingRequest() (*StorageRequest, error)
}

// ManifestManager represents a manifest manager interface
type ManifestManager interface {
    AddManifest(manifest *ManifestInfo) error
    GetManifest(name string) (*ManifestInfo, error)
    RemoveManifest(name string) error
}

// GossipManager represents a gossip manager interface
type GossipManager interface {
    AnnounceStorageNode(info *StorageNodeInfo) error
    RemoveStorageNode(id string) error
    GetPeers() []*PeerGossipInfo
    NotifyStorageSuccess(req *StorageRequest) error
    NotifyStorageRejection(req *StorageRequest, reason string) error
}
