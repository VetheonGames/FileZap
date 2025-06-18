package tun

import (
	"context"
	"fmt"
"net"
"os/exec"

"github.com/songgao/water"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

const (
	// VirtualNetworkProtocol is the protocol ID for virtual network streams
	VirtualNetworkProtocol protocol.ID = "/vpn/1.0.0"
	
	// TUNDeviceName is the name prefix for our TUN device
	TUNDeviceName = "mesh0"
	
	// DefaultMTU is the default MTU for the TUN device
	DefaultMTU = 1420
)

// MeshNetwork manages the virtual network overlay
type MeshNetwork struct {
	device    *water.Interface
	peerConns map[peer.ID]*PeerConnection
	ipNet     *net.IPNet
	selfIP    net.IP
}

// PeerConnection represents a connection to a remote peer
type PeerConnection struct {
	PeerID     peer.ID
	VirtualIP  net.IP
	StreamChan chan []byte
}

// NewMeshNetwork creates a new mesh network instance
func NewMeshNetwork(networkKey []byte) (*MeshNetwork, error) {
	// Create TUN device
cfg := water.Config{
DeviceType: water.TUN,
}
	
	dev, err := water.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create TUN device: %w", err)
	}

	// Generate deterministic IP based on network key
	ip := generateDeterministicIP(networkKey)
	ipNet := &net.IPNet{
		IP:   ip.Mask(net.CIDRMask(24, 32)), // Use /24 subnet
		Mask: net.CIDRMask(24, 32),
	}

	mn := &MeshNetwork{
		device:    dev,
		peerConns: make(map[peer.ID]*PeerConnection),
		ipNet:     ipNet,
		selfIP:    ip,
	}

	return mn, nil
}

// ConfigureInterface configures the TUN interface with IP address and routing
func (mn *MeshNetwork) ConfigureInterface() error {
	// Set interface up and configure IP
	ipStr := fmt.Sprintf("%s/24", mn.selfIP.String())
	
	// Linux-specific commands to configure the interface
	cmds := [][]string{
		{"ip", "link", "set", "dev", mn.device.Name(), "up"},
		{"ip", "addr", "add", ipStr, "dev", mn.device.Name()},
	}

	for _, cmd := range cmds {
		if err := execCommand(cmd[0], cmd[1:]...); err != nil {
			return fmt.Errorf("failed to configure interface: %w", err)
		}
	}

	return nil
}

// StartForwarding begins forwarding packets between TUN and libp2p streams
func (mn *MeshNetwork) StartForwarding(ctx context.Context) {
	go mn.readFromTUN(ctx)
}

// AddPeer adds a new peer to the mesh network
func (mn *MeshNetwork) AddPeer(id peer.ID, virtualIP net.IP) {
	mn.peerConns[id] = &PeerConnection{
		PeerID:     id,
		VirtualIP:  virtualIP,
		StreamChan: make(chan []byte, 100),
	}
}

// RemovePeer removes a peer from the mesh network
func (mn *MeshNetwork) RemovePeer(id peer.ID) {
	if conn, exists := mn.peerConns[id]; exists {
		close(conn.StreamChan)
		delete(mn.peerConns, id)
	}
}

// readFromTUN reads packets from the TUN device and forwards them
func (mn *MeshNetwork) readFromTUN(ctx context.Context) {
	buffer := make([]byte, DefaultMTU)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := mn.device.Read(buffer)
			if err != nil {
				continue
			}

			packet := make([]byte, n)
			copy(packet, buffer[:n])

			// Forward packet to appropriate peer based on destination IP
			dst := net.IP(packet[16:20]) // IPv4 destination address
			mn.forwardPacket(dst, packet)
		}
	}
}

// forwardPacket forwards a packet to the appropriate peer
func (mn *MeshNetwork) forwardPacket(dstIP net.IP, packet []byte) {
	for _, conn := range mn.peerConns {
		if conn.VirtualIP.Equal(dstIP) {
			select {
			case conn.StreamChan <- packet:
				// Successfully forwarded
			default:
				// Channel full, drop packet
			}
			return
		}
	}
}

// WriteToTUN writes a packet to the TUN interface
func (mn *MeshNetwork) WriteToTUN(packet []byte) error {
	_, err := mn.device.Write(packet)
	return err
}

// generateDeterministicIP generates a deterministic IP address from a key
func generateDeterministicIP(key []byte) net.IP {
	// Use first 4 bytes of key hash to generate IP in 10.42.0.0/16 range
	hash := make([]byte, 32)
	copy(hash, key)
	
	ip := net.IPv4(10, 42, 0, hash[0])
	return ip
}

// execCommand executes a system command
func execCommand(command string, args ...string) error {
cmd := exec.Command(command, args...)
return cmd.Run()
}
