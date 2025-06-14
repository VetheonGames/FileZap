package quorum

import (
	"fmt"
	"sync"
	"time"
)

// Vote represents a validator's vote on a key request
type Vote struct {
	ValidatorID string
	Approved    bool
	Timestamp   int64
}

// VoteSession represents an active voting session for a key request
type VoteSession struct {
	FileID        string
	ClientID      string
	Votes         map[string]Vote // map[validatorID]Vote
	StartTime     int64
	TimeoutSecs   int64
	RequiredVotes int
	mu            sync.RWMutex
	pending       bool
}

// QuorumManager handles voting sessions for key distribution
type QuorumManager struct {
	sessions      map[string]*VoteSession // map[fileID+clientID]VoteSession
	validators    map[string]bool         // map[validatorID]isActive
	voteTimeout   int64                   // seconds
	requiredVotes int
	mu            sync.RWMutex
}

// NewQuorumManager creates a new quorum manager
func NewQuorumManager(voteTimeoutSecs, requiredVotes int64) *QuorumManager {
	qm := &QuorumManager{
		sessions:      make(map[string]*VoteSession),
		validators:    make(map[string]bool),
		voteTimeout:   voteTimeoutSecs,
		requiredVotes: int(requiredVotes),
	}
	go qm.cleanupExpiredSessions()
	return qm
}

// RegisterValidator adds a validator to the quorum
func (qm *QuorumManager) RegisterValidator(validatorID string) {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	qm.validators[validatorID] = true
}

// RemoveValidator removes a validator from the quorum
func (qm *QuorumManager) RemoveValidator(validatorID string) {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	delete(qm.validators, validatorID)
}

// CreateVoteSession starts a new voting session for a key request
func (qm *QuorumManager) CreateVoteSession(fileID, clientID string) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	sessionKey := fmt.Sprintf("%s:%s", fileID, clientID)
	if _, exists := qm.sessions[sessionKey]; exists {
		return fmt.Errorf("vote session already exists")
	}

	session := &VoteSession{
		FileID:        fileID,
		ClientID:      clientID,
		Votes:         make(map[string]Vote),
		StartTime:     time.Now().Unix(),
		TimeoutSecs:   qm.voteTimeout,
		RequiredVotes: qm.requiredVotes,
		pending:       true,
	}

	qm.sessions[sessionKey] = session
	return nil
}

// SubmitVote adds a validator's vote to a session
func (qm *QuorumManager) SubmitVote(fileID, clientID, validatorID string, approved bool) error {
	qm.mu.RLock()
	if !qm.validators[validatorID] {
		qm.mu.RUnlock()
		return fmt.Errorf("invalid validator")
	}
	qm.mu.RUnlock()

	sessionKey := fmt.Sprintf("%s:%s", fileID, clientID)

	qm.mu.Lock()
	session, exists := qm.sessions[sessionKey]
	qm.mu.Unlock()

	if !exists {
		return fmt.Errorf("vote session not found")
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	// Check if session has expired
	if time.Now().Unix() > session.StartTime+session.TimeoutSecs {
		return fmt.Errorf("vote session has expired")
	}

	// Record the vote
	session.Votes[validatorID] = Vote{
		ValidatorID: validatorID,
		Approved:    approved,
		Timestamp:   time.Now().Unix(),
	}

	return nil
}

// CheckQuorum checks if a voting session has reached consensus
func (qm *QuorumManager) CheckQuorum(fileID, clientID string) (bool, error) {
	sessionKey := fmt.Sprintf("%s:%s", fileID, clientID)

	qm.mu.RLock()
	session, exists := qm.sessions[sessionKey]
	qm.mu.RUnlock()

	if !exists {
		return false, fmt.Errorf("vote session not found")
	}

	session.mu.RLock()
	defer session.mu.RUnlock()

	// Check if session has expired
	if time.Now().Unix() > session.StartTime+session.TimeoutSecs {
		return false, fmt.Errorf("vote session has expired")
	}

	// Count approved votes
	approvedCount := 0
	for _, vote := range session.Votes {
		if vote.Approved {
			approvedCount++
		}
	}

	// Check if we have enough votes for quorum
	approved := approvedCount >= session.RequiredVotes
	if approved {
		session.pending = false
	}
	return approved, nil
}

// cleanupExpiredSessions periodically removes expired voting sessions
func (qm *QuorumManager) cleanupExpiredSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		qm.mu.Lock()
		now := time.Now().Unix()

		for key, session := range qm.sessions {
			session.mu.RLock()
			if now > session.StartTime+session.TimeoutSecs {
				delete(qm.sessions, key)
			}
			session.mu.RUnlock()
		}

		qm.mu.Unlock()
	}
}

// GetVoteSession retrieves the current state of a voting session
func (qm *QuorumManager) GetVoteSession(fileID, clientID string) (*VoteSession, error) {
	sessionKey := fmt.Sprintf("%s:%s", fileID, clientID)

	qm.mu.RLock()
	defer qm.mu.RUnlock()

	session, exists := qm.sessions[sessionKey]
	if !exists {
		return nil, fmt.Errorf("vote session not found")
	}

	return session, nil
}

// GetPendingSessions returns all active vote sessions that haven't reached a decision
func (qm *QuorumManager) GetPendingSessions() []*VoteSession {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	var pending []*VoteSession
	now := time.Now().Unix()

	for _, session := range qm.sessions {
		session.mu.RLock()
		// Check if session is still active and hasn't expired
		if session.pending && now <= session.StartTime+session.TimeoutSecs {
			pending = append(pending, session)
		}
		session.mu.RUnlock()
	}

	return pending
}
