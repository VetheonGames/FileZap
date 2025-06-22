package vpn

import (
    "net"
)

// TUNConfig holds configuration for the TUN device
type TUNConfig struct {
    Name     string
    MTU      int
    Network  string
    BaseIP   net.IP
    PeerIP   net.IP
    NetMask  net.IPMask
}

// TUNDevice represents a virtual network interface
type TUNDevice struct {
    config    TUNConfig
    handle    tunHandle
    onPacket  func([]byte, string) error
}

// NewTUNDevice creates a new TUN device with the given configuration
func NewTUNDevice(cfg TUNConfig) (*TUNDevice, error) {
    handle, err := createTunDevice(cfg)
    if err != nil {
        return nil, err
    }

    return &TUNDevice{
        config: cfg,
        handle: handle,
    }, nil
}

// Start begins reading packets from the TUN device
func (t *TUNDevice) Start(handler func([]byte, string) error) error {
    t.onPacket = handler
    return t.handle.start(t.config.MTU, t.onPacket)
}

// Stop shuts down the TUN device
func (t *TUNDevice) Stop() error {
    return t.handle.close()
}

// WritePacket writes a packet to the TUN device
func (t *TUNDevice) WritePacket(packet []byte) error {
    return t.handle.write(packet)
}

// UpdateRoute adds or updates a route for a peer
func (t *TUNDevice) UpdateRoute(ip string, peerID string) error {
    return t.handle.updateRoute(ip, peerID)
}

// RemoveRoute removes a route for a peer
func (t *TUNDevice) RemoveRoute(ip string) error {
    return t.handle.removeRoute(ip)
}
