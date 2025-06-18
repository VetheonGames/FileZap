package vpn

import (
    "fmt"
    "net"
    "sync"

    "github.com/songgao/water"
    "github.com/songgao/water/waterutil"
)

// TUNConfig holds configuration for the TUN interface
type TUNConfig struct {
    Name      string    // Interface name (platform specific)
    MTU       int       // Maximum transmission unit
    Network   string    // Network CIDR (e.g. "10.42.0.0/16")
    BaseIP    net.IP    // Network base IP (e.g. 10.42.0.0)
    PeerIP    net.IP    // This peer's IP address
    NetMask   net.IPMask
}

// TUNDevice manages a platform-specific TUN interface
type TUNDevice struct {
    iface     *water.Interface
    config    TUNConfig
    isRunning bool
    mu        sync.RWMutex
    routes    map[string]string // Map of IP to peer ID
}

// NewTUNDevice creates a new TUN interface with the given configuration
func NewTUNDevice(config TUNConfig) (*TUNDevice, error) {
    // Create TUN interface with platform-specific settings
    iface, err := createTUNInterface(config)
    if err != nil {
        return nil, fmt.Errorf("failed to create TUN interface: %w", err)
    }

    return &TUNDevice{
        iface:     iface,
        config:    config,
        isRunning: false,
        routes:    make(map[string]string),
    }, nil
}

// Start begins processing packets on the TUN interface
func (t *TUNDevice) Start(packetHandler func([]byte, string) error) error {
    t.mu.Lock()
    if t.isRunning {
        t.mu.Unlock()
        return fmt.Errorf("TUN device is already running")
    }
    t.isRunning = true
    t.mu.Unlock()

    go t.readPackets(packetHandler)
    return nil
}

// Stop halts packet processing and closes the TUN interface
func (t *TUNDevice) Stop() error {
    t.mu.Lock()
    defer t.mu.Unlock()

    if !t.isRunning {
        return nil
    }

    t.isRunning = false
    return t.iface.Close()
}

// WritePacket writes a packet to the TUN interface
func (t *TUNDevice) WritePacket(packet []byte) error {
    t.mu.RLock()
    if !t.isRunning {
        t.mu.RUnlock()
        return fmt.Errorf("TUN device is not running")
    }
    t.mu.RUnlock()

    _, err := t.iface.Write(packet)
    return err
}

// UpdateRoute adds or updates a route for a peer
func (t *TUNDevice) UpdateRoute(ip string, peerID string) {
    t.mu.Lock()
    defer t.mu.Unlock()
    t.routes[ip] = peerID
}

// RemoveRoute removes a route for a peer
func (t *TUNDevice) RemoveRoute(ip string) {
    t.mu.Lock()
    defer t.mu.Unlock()
    delete(t.routes, ip)
}

// GetPeerIDByIP returns the peer ID associated with an IP address
func (t *TUNDevice) GetPeerIDByIP(ip string) (string, bool) {
    t.mu.RLock()
    defer t.mu.RUnlock()
    peerID, exists := t.routes[ip]
    return peerID, exists
}

// readPackets continuously reads packets from the TUN interface
func (t *TUNDevice) readPackets(packetHandler func([]byte, string) error) {
    buffer := make([]byte, t.config.MTU)

    for {
        t.mu.RLock()
        if !t.isRunning {
            t.mu.RUnlock()
            return
        }
        t.mu.RUnlock()

        n, err := t.iface.Read(buffer)
        if err != nil {
            continue
        }

        packet := buffer[:n]
        if !waterutil.IsIPv4(packet) {
            continue // Skip non-IPv4 packets for now
        }

        dstIP := waterutil.IPv4Destination(packet).String()
        if peerID, exists := t.GetPeerIDByIP(dstIP); exists {
            if err := packetHandler(packet, peerID); err != nil {
                // Log error but continue processing
                continue
            }
        }
    }
}
