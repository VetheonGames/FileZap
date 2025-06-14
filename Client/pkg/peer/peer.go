package peer

import (
	"context"
	"sync"
	"time"
)

// Peer represents a node in the network
type Peer struct {
	ID            string    // Unique identifier for the peer
	Address       string    // Network address
	AvailableZaps []string  // List of zap files this peer has
	LastSeen      time.Time // Last time this peer was seen
}

// Manager handles peer connections and state
type Manager struct {
	peers   map[string]*Peer // map[peerID]Peer
	timeout time.Duration
	mu      sync.RWMutex
}

// NewManager creates a new peer manager
func NewManager(timeoutSecs int64) *Manager {
	return &Manager{
		peers:   make(map[string]*Peer),
		timeout: time.Duration(timeoutSecs) * time.Second,
	}
}

// UpdatePeer updates or adds a peer's status
func (m *Manager) UpdatePeer(id string, address string, availableZaps []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	peer, exists := m.peers[id]
	if !exists {
		peer = &Peer{
			ID:      id,
			Address: address,
		}
		m.peers[id] = peer
	}

	peer.AvailableZaps = availableZaps
	peer.LastSeen = time.Now()
	if address != "" {
		peer.Address = address
	}
}

// GetPeer retrieves a peer by ID
func (m *Manager) GetPeer(id string) (*Peer, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	peer, exists := m.peers[id]
	return peer, exists
}

// GetAllPeers returns all known peers
func (m *Manager) GetAllPeers() []*Peer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	peers := make([]*Peer, 0, len(m.peers))
	for _, p := range m.peers {
		if time.Since(p.LastSeen) < m.timeout {
			peers = append(peers, p)
		}
	}
	return peers
}

// RemovePeer removes a peer from the manager
func (m *Manager) RemovePeer(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.peers, id)
}

// StartHealthChecks begins periodic peer health checks
func (m *Manager) StartHealthChecks(ctx context.Context) {
	ticker := time.NewTicker(m.timeout / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.cleanupStale()
		}
	}
}

// cleanupStale removes peers that haven't been seen recently
func (m *Manager) cleanupStale() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for id, peer := range m.peers {
		if now.Sub(peer.LastSeen) > m.timeout {
			delete(m.peers, id)
		}
	}
}

// GetPeersWithZap returns all peers that have a specific zap file
func (m *Manager) GetPeersWithZap(zapID string) []*Peer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var peers []*Peer
	for _, peer := range m.peers {
		if time.Since(peer.LastSeen) < m.timeout {
			for _, id := range peer.AvailableZaps {
				if id == zapID {
					peers = append(peers, peer)
					break
				}
			}
		}
	}
	return peers
}
