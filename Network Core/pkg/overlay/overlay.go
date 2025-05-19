package overlay

import (
"context"
"crypto/rand"
"encoding/hex"
"encoding/json"
"fmt"
"io"
"net"
"sync"
"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/multiformats/go-multiaddr"
)

const (
	ProtocolID          = "/filezap/1.0.0"
	DHTPingInterval     = 30 * time.Second
	LANDiscoveryPort    = 6666
	BootstrapTimeout    = 60 * time.Second
)

// OverlayNode represents a node in the overlay network
type OverlayNode struct {
	host       host.Host
	dht        *dht.IpfsDHT
	ctx        context.Context
	cancel     context.CancelFunc
	nodeID     string
	peers      sync.Map // peer.ID -> PeerInfo
	lanPeers   sync.Map // string -> PeerInfo
	msgHandler MessageHandler
}

// PeerInfo stores information about a peer
type PeerInfo struct {
	ID        peer.ID
	Addresses []multiaddr.Multiaddr
	IsLAN    bool
	LastSeen time.Time
}

// Message represents an overlay network message
type Message struct {
	FromID   string `json:"from_id"`
	ToID     string `json:"to_id"`
	Type     string `json:"msg_type"`
	Payload  []byte `json:"payload"`
	IsLAN    bool   `json:"is_lan"`
}

// MessageHandler handles incoming messages
type MessageHandler interface {
	HandleMessage(msg *Message) error
}

// NewOverlayNode creates a new overlay network node
func NewOverlayNode(ctx context.Context) (*OverlayNode, error) {
	// Generate node private key
	priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate node key: %v", err)
	}

// Configure network transports
listenAddrs := []multiaddr.Multiaddr{
mustMultiaddr("/ip4/0.0.0.0/tcp/0"),
mustMultiaddr("/ip4/0.0.0.0/udp/0/quic"),
}

// Create libp2p host
h, err := libp2p.New(
libp2p.Identity(priv),
libp2p.ListenAddrs(listenAddrs...),
libp2p.EnableRelay(),
libp2p.EnableAutoRelay(),
libp2p.NATPortMap(),
libp2p.EnableHolePunching(),
)
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %v", err)
	}

	// Create context
	ctx, cancel := context.WithCancel(ctx)

	// Create DHT
	kdht, err := dht.New(ctx, h)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create DHT: %v", err)
	}

	// Create node
	node := &OverlayNode{
		host:    h,
		dht:     kdht,
		ctx:     ctx,
		cancel:  cancel,
		nodeID:  hex.EncodeToString([]byte(h.ID())),
	}

// Set up stream handler
node.setupStreamHandler()

// Start discovery
go node.startDiscovery()
go node.startLANDiscovery()

return node, nil
}

// Helper function to create multiaddr
func mustMultiaddr(s string) multiaddr.Multiaddr {
addr, err := multiaddr.NewMultiaddr(s)
if err != nil {
panic(fmt.Sprintf("Failed to create multiaddr: %v", err))
}
return addr
}

// Close shuts down the overlay node
func (n *OverlayNode) Close() error {
	n.cancel()
	if err := n.dht.Close(); err != nil {
		return err
	}
	return n.host.Close()
}

// SendMessage sends a message to a specific node
func (n *OverlayNode) SendMessage(toID string, msgType string, payload []byte) error {
	// Check if peer is on LAN first
	if lanPeer, ok := n.lanPeers.Load(toID); ok {
		peerInfo := lanPeer.(PeerInfo)
		return n.sendDirectMessage(peerInfo.ID, &Message{
			FromID:  n.nodeID,
			ToID:    toID,
			Type:    msgType,
			Payload: payload,
			IsLAN:   true,
		})
	}

	// Otherwise route through overlay
	return n.sendOverlayMessage(&Message{
		FromID:  n.nodeID,
		ToID:    toID,
		Type:    msgType,
		Payload: payload,
		IsLAN:   false,
	})
}

// SetMessageHandler sets the handler for incoming messages
func (n *OverlayNode) SetMessageHandler(handler MessageHandler) {
	n.msgHandler = handler
}

// Internal methods

func (n *OverlayNode) startDiscovery() {
	// Bootstrap DHT
	ctx, cancel := context.WithTimeout(n.ctx, BootstrapTimeout)
	defer cancel()

	if err := n.dht.Bootstrap(ctx); err != nil {
		fmt.Printf("DHT bootstrap error: %v\n", err)
		return
	}

	// Start periodic ping to maintain DHT
	ticker := time.NewTicker(DHTPingInterval)
	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			n.pingDHTPeers()
		}
	}
}

func (n *OverlayNode) startLANDiscovery() {
// Nothing to do with port already in UDPAddr
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{Port: LANDiscoveryPort})
	if err != nil {
		fmt.Printf("LAN discovery listen error: %v\n", err)
		return
	}
	defer conn.Close()

	// Broadcast presence
	go n.broadcastPresence(conn)

	// Listen for other peers
	buffer := make([]byte, 1024)
	for {
		select {
case <-n.ctx.Done():
return
default:
nBytes, _, err := conn.ReadFromUDP(buffer)
if err != nil {
continue
}
// Process peer announcement
n.handlePeerAnnouncement(buffer[:nBytes])
		}
	}
}

func (n *OverlayNode) broadcastPresence(conn *net.UDPConn) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	announcement := []byte(n.nodeID)
	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			bcast := &net.UDPAddr{
				IP:   net.IPv4(255, 255, 255, 255),
				Port: LANDiscoveryPort,
			}
			conn.WriteToUDP(announcement, bcast)
		}
	}
}

func (n *OverlayNode) handlePeerAnnouncement(data []byte) {
	peerID := string(data)
	if peerID == n.nodeID {
		return // Ignore self
	}

	// Store as LAN peer
	n.lanPeers.Store(peerID, PeerInfo{
		ID:       peer.ID(peerID),
		IsLAN:    true,
		LastSeen: time.Now(),
	})
}

func (n *OverlayNode) pingDHTPeers() {
for _, p := range n.host.Network().Peers() {
resultChan := ping.Ping(n.ctx, n.host, p)
select {
case result, ok := <-resultChan:
if !ok || result.Error != nil {
n.host.Network().ClosePeer(p)
}
case <-n.ctx.Done():
return
}
}
}

func (n *OverlayNode) sendDirectMessage(peerID peer.ID, msg *Message) error {
	stream, err := n.host.NewStream(n.ctx, peerID, protocol.ID(ProtocolID))
	if err != nil {
		return fmt.Errorf("failed to open stream: %v", err)
	}
	defer stream.Close()

	if err := WriteMessage(stream, msg); err != nil {
		return fmt.Errorf("failed to write message: %v", err)
	}

	return nil
}

func (n *OverlayNode) sendOverlayMessage(msg *Message) error {
	// Find peer in DHT
	ctx, cancel := context.WithTimeout(n.ctx, 30*time.Second)
	defer cancel()

	peerID, err := n.dht.FindPeer(ctx, peer.ID(msg.ToID))
	if err != nil {
		return fmt.Errorf("failed to find peer: %v", err)
	}

	return n.sendDirectMessage(peerID.ID, msg)
}

func (n *OverlayNode) handleIncomingStream(stream network.Stream) {
	defer stream.Close()

	msg, err := ReadMessage(stream)
	if err != nil {
		fmt.Printf("Failed to read message: %v\n", err)
		return
	}

	if n.msgHandler != nil {
		if err := n.msgHandler.HandleMessage(msg); err != nil {
			fmt.Printf("Failed to handle message: %v\n", err)
		}
	}
}

// setupStreamHandler sets up the handler for incoming streams
func (n *OverlayNode) setupStreamHandler() {
n.host.SetStreamHandler(protocol.ID(ProtocolID), n.handleIncomingStream)
}

// Utility functions for message serialization
func WriteMessage(stream network.Stream, msg *Message) error {
data, err := json.Marshal(msg)
if err != nil {
return fmt.Errorf("failed to marshal message: %v", err)
}

// Write length prefix
length := uint64(len(data))
if err := writeUint64(stream, length); err != nil {
return fmt.Errorf("failed to write message length: %v", err)
}

// Write message data
_, err = stream.Write(data)
if err != nil {
return fmt.Errorf("failed to write message data: %v", err)
}

return nil
}

func ReadMessage(stream network.Stream) (*Message, error) {
// Read length prefix
length, err := readUint64(stream)
if err != nil {
return nil, fmt.Errorf("failed to read message length: %v", err)
}

// Read message data
data := make([]byte, length)
_, err = io.ReadFull(stream, data)
if err != nil {
return nil, fmt.Errorf("failed to read message data: %v", err)
}

var msg Message
if err := json.Unmarshal(data, &msg); err != nil {
return nil, fmt.Errorf("failed to unmarshal message: %v", err)
}

return &msg, nil
}

func writeUint64(w io.Writer, n uint64) error {
var buf [8]byte
for i := range buf {
buf[i] = byte(n >> (8 * (7 - i)))
}
_, err := w.Write(buf[:])
return err
}

func readUint64(r io.Reader) (uint64, error) {
var buf [8]byte
_, err := io.ReadFull(r, buf[:])
if err != nil {
return 0, err
}
var n uint64
for i := range buf {
n |= uint64(buf[i]) << (8 * (7 - i))
}
return n, nil
}
