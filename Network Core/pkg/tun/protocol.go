package tun

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	// rendezvousKey is used for peer discovery
	rendezvousKey = "MeshNetGenesis"

	// announceInterval is how often to re-announce on DHT
	announceInterval = 1 * time.Minute

	// peerTimeout is how long to wait before considering a peer offline
	peerTimeout = 2 * time.Minute
)

// MeshProtocol handles the virtual network protocol
type MeshProtocol struct {
	host       host.Host
	dht        *dht.IpfsDHT
	mesh       *MeshNetwork
	peerMap    sync.Map // peer.ID -> virtualIP
	networkKey []byte
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewMeshProtocol creates a new mesh protocol handler
func NewMeshProtocol(h host.Host, d *dht.IpfsDHT, networkKey []byte) (*MeshProtocol, error) {
	ctx, cancel := context.WithCancel(context.Background())

	mp := &MeshProtocol{
		host:       h,
		dht:        d,
		networkKey: networkKey,
		ctx:        ctx,
		cancel:     cancel,
	}

	// Create mesh network
	mesh, err := NewMeshNetwork(networkKey)
	if err != nil {
		cancel()
		return nil, err
	}
	mp.mesh = mesh

	// Set up protocol handler
	h.SetStreamHandler(VirtualNetworkProtocol, mp.handleStream)

	return mp, nil
}

// Start begins protocol operations
func (mp *MeshProtocol) Start() error {
	// Configure TUN interface
	if err := mp.mesh.ConfigureInterface(); err != nil {
		return fmt.Errorf("failed to configure TUN: %w", err)
	}

	// Start packet forwarding
	mp.mesh.StartForwarding(mp.ctx)

	// Start discovery
	go mp.discoveryLoop()

	// Start peer management
	go mp.peerManagementLoop()

	return nil
}

// Stop gracefully shuts down the protocol
func (mp *MeshProtocol) Stop() error {
	mp.cancel()
	// Cleanup code here
	return nil
}

// discoveryLoop periodically announces and looks for peers
func (mp *MeshProtocol) discoveryLoop() {
	// Generate rendezvous point from shared key
	rendezvousHash := sha256.Sum256([]byte(rendezvousKey))
	rendezvousPoint := fmt.Sprintf("/filezap/mesh/%x", rendezvousHash[:8])

	ticker := time.NewTicker(announceInterval)
	defer ticker.Stop()

	for {
		select {
		case <-mp.ctx.Done():
			return
		case <-ticker.C:
			// Announce self
			if err := mp.dht.Provide(mp.ctx, rendezvousPoint, true); err != nil {
				continue
			}

			// Find other peers
			peers, err := mp.dht.FindProviders(mp.ctx, rendezvousPoint)
			if err != nil {
				continue
			}

			// Connect to new peers
			for _, p := range peers {
				if p.ID == mp.host.ID() {
					continue // Skip self
				}
				mp.connectToPeer(p.ID)
			}
		}
	}
}

// peerManagementLoop handles peer timeouts and cleanup
func (mp *MeshProtocol) peerManagementLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-mp.ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			mp.peerMap.Range(func(key, value interface{}) bool {
				peerID := key.(peer.ID)
				lastSeen := value.(time.Time)
				if now.Sub(lastSeen) > peerTimeout {
					mp.mesh.RemovePeer(peerID)
					mp.peerMap.Delete(peerID)
				}
				return true
			})
		}
	}
}

// connectToPeer attempts to establish a virtual network connection with a peer
func (mp *MeshProtocol) connectToPeer(p peer.ID) error {
	// Skip if already connected
	if _, ok := mp.peerMap.Load(p); ok {
		return nil
	}

	// Open stream
	s, err := mp.host.NewStream(mp.ctx, p, VirtualNetworkProtocol)
	if err != nil {
		return err
	}

	// Handle stream in goroutine
	go mp.handleStream(s)

	return nil
}

// handleStream manages a virtual network stream with a peer
func (mp *MeshProtocol) handleStream(s network.Stream) {
	defer s.Close()

	// Get peer info
	p := s.Conn().RemotePeer()

	// Generate deterministic IP for peer
	peerKey := append(mp.networkKey, []byte(p.String())...)
	peerIP := generateDeterministicIP(peerKey)

	// Add to mesh
	mp.mesh.AddPeer(p, peerIP)
	mp.peerMap.Store(p, time.Now())

	// Start packet forwarding
	go mp.forwardPackets(s, p)

	// Keep stream alive and update last seen time
	for {
		select {
		case <-mp.ctx.Done():
			return
		default:
			// Simple keepalive - read and discard
			_, err := s.Read(make([]byte, 1))
			if err != nil {
				if err != io.EOF {
					mp.mesh.RemovePeer(p)
					mp.peerMap.Delete(p)
				}
				return
			}
			mp.peerMap.Store(p, time.Now())
		}
	}
}

// forwardPackets handles packet forwarding between TUN and libp2p stream
func (mp *MeshProtocol) forwardPackets(s network.Stream, p peer.ID) {
	conn := mp.mesh.peerConns[p]

	// Stream to TUN
	go func() {
		buf := make([]byte, DefaultMTU+2) // +2 for length prefix
		for {
			n, err := s.Read(buf)
			if err != nil {
				return
			}
			if n < 2 {
				continue
			}

			length := binary.BigEndian.Uint16(buf[:2])
			if int(length)+2 > n {
				continue
			}

			packet := buf[2 : length+2]
			mp.mesh.WriteToTUN(packet)
		}
	}()

	// TUN to Stream
	for packet := range conn.StreamChan {
		if len(packet) > DefaultMTU {
			continue
		}

		// Add length prefix
		buf := make([]byte, len(packet)+2)
		binary.BigEndian.PutUint16(buf, uint16(len(packet)))
		copy(buf[2:], packet)

		if _, err := s.Write(buf); err != nil {
			return
		}
	}
}
