package network

import (
    "context"
    "fmt"
    "time"

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

// Size constants
const (
    maxChunkSize    = 100 * 1024 * 1024   // 100MB max chunk size
    maxTotalSize    = 1024 * 1024 * 1024  // 1GB total storage limit
    maxStorageSize  = 10 * 1024 * 1024 * 1024 // 10GB default max storage
)

// Vote related constants
const (
    QuorumTopic          = "filezap-quorum"
    VotingTimeout        = 30 * time.Second
    MinQuorumSize        = 5   // Minimum peers needed for valid quorum
    MinVotingPercentage  = 67  // Min percentage needed for vote to pass (2/3 majority)
    ReputationThreshold  = -50 // Reputation threshold for peer removal
    MaxReputation        = 100 // Maximum reputation score
    BaseVoteWeight       = 1   // Base voting weight for regular nodes
    StorerVoteWeight     = 3   // Higher voting weight for storage nodes
)

// VoteType represents different types of votes
type VoteType int

const (
    // VoteRemovePeer indicates a vote to remove a peer
    VoteRemovePeer VoteType = iota
    // VoteRemoveFile indicates a vote to remove a file
    VoteRemoveFile
    // VoteUpdateRules indicates a vote to update network rules
    VoteUpdateRules
)

// Vote represents a network decision to be made
type Vote struct {
    ID        string    `json:"id"`
    Type      VoteType  `json:"type"`
    Target    string    `json:"target"`   // Peer ID or file hash
    Reason    string    `json:"reason"`
    Evidence  []byte    `json:"evidence"` // Optional evidence (e.g., invalid chunk data)
    Timestamp time.Time `json:"timestamp"`
    Proposer  peer.ID   `json:"proposer"`
}

// VoteResponse represents a peer's vote
type VoteResponse struct {
    VoteID    string    `json:"vote_id"`
    Voter     peer.ID   `json:"voter"`
    Approve   bool      `json:"approve"`
    Timestamp time.Time `json:"timestamp"`
    IsStorer  bool      `json:"is_storer"` // Whether voter is a storage node
    Weight    int       `json:"weight"`     // Voting weight (higher for storage nodes)
}

// ManifestInfo contains metadata about a stored file
type ManifestInfo struct {
    Name            string
    Owner           string
    ChunkHashes     []string
    Size            int64
    Created         time.Time
    Modified        time.Time
    ReplicationGoal int
    UpdatedAt       time.Time
}

// StorageRequest represents a request to store data
type StorageRequest struct {
    ChunkHash string
    Data      []byte
    Size      int64
    Owner     string
}

// StorageNodeInfo contains information about a storage node
type StorageNodeInfo struct {
    ID             string
    AvailableSpace int64
    TotalSpace     int64
    Uptime         float64
    Version        string
    Location       string
}

// Error definitions
var (
    ErrNoRequestsPending = fmt.Errorf("no pending requests")
    ErrStorageFull      = fmt.Errorf("storage full")
    ErrInvalidChunk     = fmt.Errorf("invalid chunk")
)

// Interface definitions

// DHT defines the interface for DHT operations
type DHT interface {
    Bootstrap(ctx context.Context) error
}

// PubSub defines the interface for publish/subscribe operations
type PubSub interface {
    Publish(topic string, data []byte) error
    Subscribe(topic string) (Subscription, error)
}

// Subscription defines the interface for pubsub subscriptions
type Subscription interface {
    Next() (Message, error)
    Cancel()
}

// Message defines the interface for pubsub messages
type Message interface {
    Data() []byte
    From() peer.ID
    Topics() []string
}

// QuorumManager handles peer voting and consensus
type QuorumManager interface {
    Start() error
    Stop() error
    ProposeVote(voteType VoteType, target string, reason string, evidence []byte) error
    StartVote(voteType VoteType, target string, proposer peer.ID) error
    UpdatePeerReputation(p peer.ID, delta int) error
}
