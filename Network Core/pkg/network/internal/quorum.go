package internal

import (
    "context"
    "encoding/json"
    "fmt"
    "sync"
    "time"

    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/peer"
    pubsub "github.com/libp2p/go-libp2p-pubsub"
)

const (
    quorumTopic = "/filezap/quorum/1.0.0"
    votingWindow = time.Minute * 5
    voteThreshold = 0.66 // 66% of peers must agree for a vote to pass
)

// Vote represents a peer vote
type Vote struct {
    Type      VoteType
    Name      string
    Reason    string
    Data      []byte
    Timestamp time.Time
    VoterID   peer.ID
}

// quorumManager implements consensus and voting
type quorumManager struct {
    ctx       context.Context
    host      host.Host
    pubsub    *pubsub.PubSub
    topic     *pubsub.Topic
    sub       *pubsub.Subscription
    peers     map[peer.ID]int // reputation scores
    votes     map[string]map[peer.ID]Vote
    voteTimes map[string]time.Time
    mu        sync.RWMutex
}

func newQuorumManager(ctx context.Context, h host.Host, ps *pubsub.PubSub) (*quorumManager, error) {
    // Create manager
    qm := &quorumManager{
        ctx:       ctx,
        host:      h,
        pubsub:    ps,
        peers:     make(map[peer.ID]int),
        votes:     make(map[string]map[peer.ID]Vote),
        voteTimes: make(map[string]time.Time),
    }

    // Join quorum topic
    topic, err := ps.Join(quorumTopic)
    if err != nil {
        return nil, fmt.Errorf("failed to join quorum topic: %w", err)
    }
    qm.topic = topic

    // Subscribe to votes
    sub, err := topic.Subscribe()
    if err != nil {
        return nil, fmt.Errorf("failed to subscribe to quorum topic: %w", err)
    }
    qm.sub = sub

    // Start vote handler
    go qm.handleVotes()

    // Start vote cleanup
    go qm.cleanupVotes()

    return qm, nil
}

func (qm *quorumManager) handleVotes() {
    for {
        msg, err := qm.sub.Next(qm.ctx)
        if err != nil {
            return
        }

        // Skip messages from self
        if msg.ReceivedFrom == qm.host.ID() {
            continue
        }

        // Parse vote
        var vote Vote
        if err := json.Unmarshal(msg.Data, &vote); err != nil {
            continue
        }

        // Process vote
        qm.processVote(vote)
    }
}

func (qm *quorumManager) processVote(vote Vote) {
    qm.mu.Lock()
    defer qm.mu.Unlock()

    // Create vote map if it doesn't exist
    voteKey := fmt.Sprintf("%s:%s", vote.Type, vote.Name)
    if _, exists := qm.votes[voteKey]; !exists {
        qm.votes[voteKey] = make(map[peer.ID]Vote)
        qm.voteTimes[voteKey] = time.Now()
    }

    // Record vote
    qm.votes[voteKey][vote.VoterID] = vote

    // Check if vote should be tallied
    qm.checkVoteThreshold(voteKey)
}

func (qm *quorumManager) checkVoteThreshold(voteKey string) {
    // Get total peers
    totalPeers := len(qm.peers)
    if totalPeers == 0 {
        return
    }

    // Get vote count
    votes := len(qm.votes[voteKey])

    // Check threshold
    if float64(votes)/float64(totalPeers) >= voteThreshold {
        // Vote passed - execute action
        qm.executeVote(voteKey)

        // Clean up vote
        delete(qm.votes, voteKey)
        delete(qm.voteTimes, voteKey)
    }
}

func (qm *quorumManager) executeVote(voteKey string) {
    // Get a vote to determine type/name
    var vote Vote
    for _, v := range qm.votes[voteKey] {
        vote = v
        break
    }

    switch vote.Type {
    case VoteRemovePeer:
        // Remove peer from network
        peerID := peer.ID(vote.Name)
        delete(qm.peers, peerID)
        // TODO: Implement peer removal actions

    case VoteRemoveFile:
        // Remove file from network
        // TODO: Implement file removal actions
    }
}

func (qm *quorumManager) cleanupVotes() {
    ticker := time.NewTicker(time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-qm.ctx.Done():
            return
        case <-ticker.C:
            qm.mu.Lock()
            now := time.Now()
            for key, timestamp := range qm.voteTimes {
                if now.Sub(timestamp) > votingWindow {
                    delete(qm.votes, key)
                    delete(qm.voteTimes, key)
                }
            }
            qm.mu.Unlock()
        }
    }
}

// ProposeVote implements the QuorumManager interface
func (qm *quorumManager) ProposeVote(voteType VoteType, name string, reason string, data []byte) error {
    vote := Vote{
        Type:      voteType,
        Name:      name,
        Reason:    reason,
        Data:      data,
        Timestamp: time.Now(),
        VoterID:   qm.host.ID(),
    }

    // Marshal vote
    voteData, err := json.Marshal(vote)
    if err != nil {
        return fmt.Errorf("failed to marshal vote: %w", err)
    }

    // Publish vote
    if err := qm.topic.Publish(qm.ctx, voteData); err != nil {
        return fmt.Errorf("failed to publish vote: %w", err)
    }

    return nil
}

// UpdatePeerReputation implements the QuorumManager interface
func (qm *quorumManager) UpdatePeerReputation(id peer.ID, change int) {
    qm.mu.Lock()
    defer qm.mu.Unlock()

    qm.peers[id] += change

    // If reputation drops too low, propose removal
    if qm.peers[id] < -10 {
        go qm.ProposeVote(VoteRemovePeer, string(id), "reputation too low", nil)
    }
}
