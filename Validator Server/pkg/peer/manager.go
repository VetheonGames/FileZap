package peer

import (
	"sync"
	"time"
)

// Peer represents a connected peer in the network
type Peer struct {
	ID            string    `json:"id"`
	Address       string    `json:"address"`
	LastSeen      time.Time `json:"last_seen"`
	AvailableZaps []string  `json:"available_zaps"`
}

// Manager handles the state of all connected peers
type Manager struct {
	peers   map[string]*Peer
	timeout time.Duration
	mu      sync.RWMutex
}

// NewManager creates a new peer manager
func NewManager(timeout time.Duration) *Manager {
	pm := &Manager{
		peers:   make(map[string]*Peer),
		timeout: timeout,
	}
	go pm.cleanupInactivePeers()
	return pm
}

// cleanupInactivePeers periodically removes inactive peers
func (m *Manager) cleanupInactivePeers() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for id, peer := range m.peers {
			if now.Sub(peer.LastSeen) > m.timeout {
				delete(m.peers, id)
			}
		}
		m.mu.Unlock()
	}
}

// UpdatePeer adds or updates a peer's status
func (m *Manager) UpdatePeer(id, address string, availableZaps []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.peers[id] = &Peer{
		ID:            id,
		Address:       address,
		LastSeen:      time.Now(),
		AvailableZaps: availableZaps,
	}
}

// GetPeer retrieves a peer by ID
func (m *Manager) GetPeer(id string) (*Peer, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	peer, exists := m.peers[id]
	return peer, exists
}

// GetPeersWithZap returns all peers that have the specified .zap file
func (m *Manager) GetPeersWithZap(zapID string) []*Peer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var peersWithZap []*Peer
	for _, peer := range m.peers {
		for _, peerZap := range peer.AvailableZaps {
			if peerZap == zapID {
				peersWithZap = append(peersWithZap, peer)
				break
			}
		}
	}
	return peersWithZap
}

// GetAllPeers returns all active peers
func (m *Manager) GetAllPeers() map[string]*Peer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	peers := make(map[string]*Peer, len(m.peers))
	for id, peer := range m.peers {
		peers[id] = peer
	}
	return peers
}
