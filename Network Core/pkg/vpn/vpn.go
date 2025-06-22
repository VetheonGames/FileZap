package vpn

import (
    "context"
    "crypto/sha256"
    "fmt"
    "net"
    "sync"
    "time"

    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/network"
    "github.com/libp2p/go-libp2p/core/peer"
    "github.com/libp2p/go-libp2p/core/protocol"
)

const (
    // Protocol ID for VPN streams
    VPNProtocolID = protocol.ID("/vpn/1.0.0")
    
    // Default network settings
    DefaultMTU        = 1420 // Slightly lower than standard 1500 to account for overhead
    DefaultNetworkCIDR = "10.42.0.0/16"
)

// VPNManager handles the virtual network overlay
type VPNManager struct {
    host     host.Host
    tun      *TUNDevice
    peers    map[peer.ID]*VPNPeer
    streams  map[peer.ID]network.Stream
    baseIP   net.IP
    netmask  net.IPMask
    ctx      context.Context
    cancel   context.CancelFunc
    mu       sync.RWMutex
}

// VPNPeer represents a connected peer in the VPN
type VPNPeer struct {
    ID       peer.ID
    IP       net.IP
    Stream   network.Stream
    Active   bool
}

// Config holds VPN configuration
type Config struct {
    NetworkCIDR string  // Network CIDR (e.g. "10.42.0.0/16")
    InterfaceName string // TUN interface name
    MTU          int    // Maximum transmission unit
}

// DefaultConfig returns default VPN configuration
func DefaultConfig() *Config {
    return &Config{
        NetworkCIDR:   DefaultNetworkCIDR,
        InterfaceName: "tun0",
        MTU:          DefaultMTU,
    }
}

// NewVPNManager creates a new VPN manager
func NewVPNManager(ctx context.Context, h host.Host, cfg *Config) (*VPNManager, error) {
    // Parse network CIDR
    _, ipNet, err := net.ParseCIDR(cfg.NetworkCIDR)
    if err != nil {
        return nil, fmt.Errorf("invalid network CIDR: %w", err)
    }

    // Create VPN manager
    ctx, cancel := context.WithCancel(ctx)
    vpn := &VPNManager{
        host:     h,
        peers:    make(map[peer.ID]*VPNPeer),
        streams:  make(map[peer.ID]network.Stream),
        baseIP:   ipNet.IP,
        netmask:  ipNet.Mask,
        ctx:      ctx,
        cancel:   cancel,
    }

    // Calculate this peer's IP based on peer ID
    peerIP, err := vpn.calculatePeerIP(h.ID())
    if err != nil {
        cancel()
        return nil, err
    }

    // Create TUN device
    tunCfg := TUNConfig{
        Name:    cfg.InterfaceName,
        MTU:     cfg.MTU,
        Network: cfg.NetworkCIDR,
        BaseIP:  ipNet.IP,
        PeerIP:  peerIP,
        NetMask: ipNet.Mask,
    }

    tun, err := NewTUNDevice(tunCfg)
    if err != nil {
        cancel()
        return nil, fmt.Errorf("failed to create TUN device: %w", err)
    }
    vpn.tun = tun

    // Set up stream handler
    h.SetStreamHandler(VPNProtocolID, vpn.handleStream)

    // Start packet handling
    if err := tun.Start(vpn.handlePacket); err != nil {
        cancel()
        return nil, fmt.Errorf("failed to start TUN device: %w", err)
    }

    return vpn, nil
}

// Close shuts down the VPN manager
func (v *VPNManager) Close() error {
    v.cancel()
    v.mu.Lock()
    defer v.mu.Unlock()

    // Close all streams
    for _, stream := range v.streams {
        stream.Close()
    }

    // Stop TUN device
    if v.tun != nil {
        return v.tun.Stop()
    }
    return nil
}

// GetLocalIP returns this peer's virtual IP address
func (v *VPNManager) GetLocalIP() string {
    return v.tun.config.PeerIP.String()
}

// GetPeers returns a list of all known peer IDs
func (v *VPNManager) GetPeers() []peer.ID {
    v.mu.RLock()
    defer v.mu.RUnlock()
    
    peerList := make([]peer.ID, 0, len(v.peers))
    for id := range v.peers {
        peerList = append(peerList, id)
    }
    return peerList
}

// GetActivePeers returns information about currently active peers
func (v *VPNManager) GetActivePeers() []VPNPeerInfo {
    v.mu.RLock()
    defer v.mu.RUnlock()
    
    activePeers := make([]VPNPeerInfo, 0)
    for id, peer := range v.peers {
        if peer.Active {
            activePeers = append(activePeers, VPNPeerInfo{
                ID: id.String(),
                IP: peer.IP.String(),
            })
        }
    }
    return activePeers
}

// VPNPeerInfo contains information about a VPN peer
type VPNPeerInfo struct {
    ID string
    IP string
}

func (v *VPNManager) calculatePeerIP(id peer.ID) (net.IP, error) {
    // Hash the peer ID to get a deterministic value
    hash := sha256.Sum256([]byte(id))
    
    // Use the first 2 bytes of the hash to generate the last two octets
    // This gives us up to 65536 possible peer IPs
    ip := make(net.IP, len(v.baseIP))
    copy(ip, v.baseIP)
    ip[2] = hash[0]
    ip[3] = hash[1]

    if ip[2] == 0 && ip[3] == 0 {
        return nil, fmt.Errorf("invalid IP generated for peer %s", id)
    }

    return ip, nil
}

// handleStream processes incoming VPN streams
func (v *VPNManager) handleStream(s network.Stream) {
    peer := s.Conn().RemotePeer()
    
    v.mu.Lock()
    // Close existing stream if any
    if oldStream, exists := v.streams[peer]; exists {
        oldStream.Close()
    }
    v.streams[peer] = s
    
    // Calculate peer's IP
    peerIP, err := v.calculatePeerIP(peer)
    if err != nil {
        v.mu.Unlock()
        s.Close()
        return
    }
    
    // Create or update peer info
    v.peers[peer] = &VPNPeer{
        ID:     peer,
        IP:     peerIP,
        Stream: s,
        Active: true,
    }
    
    // Update TUN routing
    v.tun.UpdateRoute(peerIP.String(), peer.String())
    v.mu.Unlock()

    // Handle stream data
    go v.streamReader(s, peer)
}

// streamReader reads packets from a peer stream
func (v *VPNManager) streamReader(s network.Stream, peer peer.ID) {
    defer func() {
        v.mu.Lock()
        if p, exists := v.peers[peer]; exists {
            p.Active = false
        }
        delete(v.streams, peer)
        v.mu.Unlock()
        s.Close()
    }()

    buf := make([]byte, v.tun.config.MTU)
    for {
        n, err := s.Read(buf)
        if err != nil {
            return
        }

        if err := v.tun.WritePacket(buf[:n]); err != nil {
            return
        }
    }
}

// handlePacket processes packets from the TUN interface
func (v *VPNManager) handlePacket(packet []byte, peerID string) error {
    v.mu.RLock()
    defer v.mu.RUnlock()

    for id, peer := range v.peers {
        if id.String() == peerID && peer.Active {
            _, err := peer.Stream.Write(packet)
            return err
        }
    }
    return fmt.Errorf("no active stream for peer %s", peerID)
}

// handlePeerAnnouncement processes peer announcements from discovery
func (v *VPNManager) handlePeerAnnouncement(info PeerInfo) {
    v.mu.Lock()
    defer v.mu.Unlock()

    // Skip announcements from self
    if info.PeerID == v.host.ID() {
        return
    }

    // Create or update peer info
    peer, exists := v.peers[info.PeerID]
    if !exists {
        // New peer - create entry
        peer = &VPNPeer{
            ID:     info.PeerID,
            IP:     net.ParseIP(info.VirtualIP),
            Active: false,
        }
        v.peers[info.PeerID] = peer
        
        // Open stream to new peer
        go v.connectToPeer(info.PeerID)
    }

    // Update routing if IP changed
    if peer.IP.String() != info.VirtualIP {
        if peer.Active {
            v.tun.RemoveRoute(peer.IP.String())
        }
        peer.IP = net.ParseIP(info.VirtualIP)
        if peer.Active {
            v.tun.UpdateRoute(peer.IP.String(), info.PeerID.String())
        }
    }
}

// connectToPeer attempts to establish a VPN stream with a peer
func (v *VPNManager) connectToPeer(id peer.ID) {
    ctx, cancel := context.WithTimeout(v.ctx, 30*time.Second)
    defer cancel()

    // Open stream
    stream, err := v.host.NewStream(ctx, id, VPNProtocolID)
    if err != nil {
        return
    }

    // Update peer info
    v.mu.Lock()
    if peer, exists := v.peers[id]; exists {
        // Close existing stream if any
        if oldStream := v.streams[id]; oldStream != nil {
            oldStream.Close()
        }
        
        v.streams[id] = stream
        peer.Stream = stream
        peer.Active = true
        
        // Update routing
        v.tun.UpdateRoute(peer.IP.String(), id.String())
    }
    v.mu.Unlock()

    // Start reading from stream
    go v.streamReader(stream, id)
}
