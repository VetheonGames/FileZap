package vpn

// tunHandle represents the platform-specific TUN implementation
type tunHandle interface {
    // start begins reading packets from the TUN device
    start(mtu int, handler func([]byte, string) error) error
    
    // close shuts down the TUN device
    close() error
    
    // write sends a packet to the TUN device
    write(packet []byte) error
    
    // updateRoute adds or updates a route for a peer
    updateRoute(ip string, peerID string) error
    
    // removeRoute removes a route for a peer
    removeRoute(ip string) error
}

// createTunDevice creates a platform-specific TUN device
func createTunDevice(cfg TUNConfig) (tunHandle, error) {
    // Implementation provided by platform-specific files
    return newTunDevice(cfg)
}
