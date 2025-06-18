package internal

import (
    "github.com/libp2p/go-libp2p/core/peer"
)

// VPNStatus represents the current state of the VPN connection
type VPNStatus struct {
    Connected   bool
    LocalIP     string
    PeerCount   int
    ActivePeers []string
}

// VPNPeer represents a VPN peer node
type VPNPeer struct {
    ID         peer.ID
    IP         string
    LastActive int64
    Status     VPNPeerStatus
}

// VPNPeerStatus represents the status of a VPN peer
type VPNPeerStatus int

const (
    VPNPeerConnecting VPNPeerStatus = iota
    VPNPeerConnected
    VPNPeerDisconnected
    VPNPeerFailed
)

// VPNPeerStatusString converts a VPNPeerStatus to string
func (s VPNPeerStatus) String() string {
    switch s {
    case VPNPeerConnecting:
        return "Connecting"
    case VPNPeerConnected:
        return "Connected"
    case VPNPeerDisconnected:
        return "Disconnected"
    case VPNPeerFailed:
        return "Failed"
    default:
        return "Unknown"
    }
}

// VPNStats holds VPN statistics
type VPNStats struct {
    BytesReceived uint64
    BytesSent     uint64
    Uptime        int64
    MTU           int
    ErrorCount    int
}
