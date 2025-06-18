package internal

import (
    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/peer"
    ma "github.com/multiformats/go-multiaddr"

    "github.com/VetheonGames/FileZap/NetworkCore/pkg/vpn"
)

// NetworkEngine defines the core network functionality
type NetworkEngine interface {
    // Core operations
    Connect(addr ma.Multiaddr) error
    Close() error
    GetNodeID() string
    GetPeers() []peer.ID
    GetTransportHost() host.Host
    GetMetadataHost() host.Host
    Bootstrap(addrs []peer.AddrInfo) error

    // VPN operations
    GetVPNManager() *vpn.VPNManager
    GetVPNStatus() *VPNStatus

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

// Manager interfaces for internal components

type GossipManager interface {
    AnnounceStorageNode(info *StorageNodeInfo) error
    RemoveStorageNode(id string) error
    GetPeers() []*PeerGossipInfo
    NotifyStorageSuccess(req *StorageRequest) error
    NotifyStorageRejection(req *StorageRequest, reason string) error
}

type QuorumManager interface {
    ProposeVote(voteType VoteType, name string, reason string, data []byte) error
    UpdatePeerReputation(id peer.ID, change int)
}

type ManifestManager interface {
    AddManifest(manifest *ManifestInfo) error
    GetManifest(name string) (*ManifestInfo, map[string][]byte, error)
    RemoveManifest(name string) error
}

type ChunkStore interface {
    Store(hash string, data []byte) bool
    Get(hash string) ([]byte, bool)
    Remove(hash string)
    GetPendingRequest() (*StorageRequest, error)
}

type ChunkValidator interface {
    ValidateChunk(data []byte, hash string, owner peer.ID) ValidationResult
}
