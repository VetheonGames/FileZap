//go:build linux
package vpn

import (
    "fmt"
    "os/exec"

    "github.com/songgao/water"
)

func createTUNInterface(config TUNConfig) (*water.Interface, error) {
    ifConfig := water.Config{
        DeviceType: water.TUN,
    }

    // On Linux, we let the kernel assign a name if none specified
    if config.Name != "" {
        ifConfig.PlatformSpecificParams.Name = config.Name
    }

    iface, err := water.New(ifConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create TUN interface: %w", err)
    }

    // Set interface MTU
    if err := exec.Command("ip", "link", "set", "dev", iface.Name(), "mtu", fmt.Sprint(config.MTU)).Run(); err != nil {
        iface.Close()
        return nil, fmt.Errorf("failed to set MTU: %w", err)
    }

    // Bring interface up
    if err := exec.Command("ip", "link", "set", "dev", iface.Name(), "up").Run(); err != nil {
        iface.Close()
        return nil, fmt.Errorf("failed to bring interface up: %w", err)
    }

    // Add IP address
    addr := fmt.Sprintf("%s/%d", config.PeerIP.String(), maskBits(config.NetMask))
    if err := exec.Command("ip", "addr", "add", addr, "dev", iface.Name()).Run(); err != nil {
        iface.Close()
        return nil, fmt.Errorf("failed to set IP address: %w", err)
    }

    return iface, nil
}
