package peer

import (
	"log"
	"sync"
	"time"
)

// Manager handles peer connections and their states
type Manager struct {
	peers   map[string]*Peer
	timeout time.Duration
	mu      sync.RWMutex
}

// Peer represents a connected peer
type Peer struct {
	ID            string
	LastSeen      time.Time
	AvailableZaps []string
	Address       string
}

// NewManager creates a new peer manager with the specified timeout
func NewManager(timeout time.Duration) *Manager {
	pm := &Manager{
		peers:   make(map[string]*Peer),
		timeout: timeout,
	}
	go pm.CleanupInactivePeers()
	return pm
}

// CleanupInactivePeers periodically removes inactive peers
func (m *Manager) CleanupInactivePeers() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for id, peer := range m.peers {
			if now.Sub(peer.LastSeen) > m.timeout {
				log.Printf("Removing inactive peer: %s", id)
				delete(m.peers, id)
			}
		}
		m.mu.Unlock()
	}
}

// AddPeer adds or updates a peer in the manager
func (m *Manager) AddPeer(id, address string, availableZaps []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.peers[id] = &Peer{
		ID:            id,
		LastSeen:      time.Now(),
		AvailableZaps: availableZaps,
		Address:       address,
	}
}

// GetPeer retrieves a peer by ID
func (m *Manager) GetPeer(id string) (*Peer, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	peer, exists := m.peers[id]
	return peer, exists
}

// UpdatePeerLastSeen updates the last seen time for a peer
func (m *Manager) UpdatePeerLastSeen(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if peer, exists := m.peers[id]; exists {
		peer.LastSeen = time.Now()
	}
}

// UpdatePeerZaps updates the available zaps for a peer
func (m *Manager) UpdatePeerZaps(id string, zaps []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if peer, exists := m.peers[id]; exists {
		peer.AvailableZaps = zaps
	}
}

// GetAllPeers returns a copy of all active peers
func (m *Manager) GetAllPeers() map[string]Peer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]Peer, len(m.peers))
	for id, peer := range m.peers {
		result[id] = *peer
	}
	return result
}
