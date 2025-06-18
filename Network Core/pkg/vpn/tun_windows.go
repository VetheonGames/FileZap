//go:build windows
package vpn

import (
    "fmt"
    "net"
    "os/exec"

    "github.com/songgao/water"
)

func createTUNInterface(config TUNConfig) (*water.Interface, error) {
    ifConfig := water.Config{
        DeviceType: water.TUN,
        PlatformSpecificParams: water.PlatformSpecificParams{
            ComponentID: "tap0901",
            Network:     config.Network,
        },
    }

    iface, err := water.New(ifConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create TUN interface: %w", err)
    }

    // Get interface index
    ifrName := iface.Name()

    // Set interface MTU
    if err := exec.Command("netsh", "interface", "ipv4", "set", "subinterface", 
        ifrName, fmt.Sprintf("mtu=%d", config.MTU)).Run(); err != nil {
        iface.Close()
        return nil, fmt.Errorf("failed to set MTU: %w", err)
    }

    // Configure IP address and netmask
    if err := exec.Command("netsh", "interface", "ip", "set", "address", 
        fmt.Sprintf("name=%s", ifrName), "source=static", 
        "address=" + config.PeerIP.String(),
        "mask=" + net.IP(config.NetMask).String()).Run(); err != nil {
        iface.Close()
        return nil, fmt.Errorf("failed to set IP address: %w", err)
    }

    // Enable the interface
    if err := exec.Command("netsh", "interface", "set", "interface", 
        ifrName, "admin=enabled").Run(); err != nil {
        iface.Close()
        return nil, fmt.Errorf("failed to enable interface: %w", err)
    }

    return iface, nil
}
