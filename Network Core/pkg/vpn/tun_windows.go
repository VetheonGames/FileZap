//go:build windows

package vpn

import (
    "fmt"
    "net"
    "sync"
    "syscall"

    "golang.org/x/sys/windows"
)

type winTun struct {
    handle    windows.Handle
    device    string
    network   string
    adapter   string
    mtu       int
    routes    sync.Map
    readChan  chan []byte
    stopChan  chan struct{}
}

func newTunDevice(cfg TUNConfig) (tunHandle, error) {
    // Create Windows TUN adapter using WinTun driver
    device := fmt.Sprintf("FileZap-%s", cfg.Name)
    handle, err := createWinTunAdapter(device)
    if err != nil {
        return nil, fmt.Errorf("failed to create WinTun adapter: %w", err)
    }

    tun := &winTun{
        handle:   handle,
        device:   device,
        network:  cfg.Network,
        adapter:  cfg.Name,
        mtu:      cfg.MTU,
        readChan: make(chan []byte, 100),
        stopChan: make(chan struct{}),
    }

    // Configure adapter
    if err := configureAdapter(tun.adapter, cfg.PeerIP, cfg.NetMask); err != nil {
        tun.close()
        return nil, fmt.Errorf("failed to configure adapter: %w", err)
    }

    return tun, nil
}

func (t *winTun) start(mtu int, handler func([]byte, string) error) error {
    go t.readPackets(handler)
    return nil
}

func (t *winTun) close() error {
    close(t.stopChan)
    if t.handle != 0 {
        windows.CloseHandle(t.handle)
        t.handle = 0
    }
    return nil
}

func (t *winTun) write(packet []byte) error {
    var written uint32
    err := windows.WriteFile(t.handle, packet, &written, nil)
    if err != nil {
        return fmt.Errorf("write failed: %w", err)
    }
    return nil
}

func (t *winTun) updateRoute(ip string, peerID string) error {
    parsedIP := net.ParseIP(ip)
    if parsedIP == nil {
        return fmt.Errorf("invalid IP address: %s", ip)
    }

    // Add route to Windows routing table
    args := []string{
        "route", "add",
        ip, "mask", "255.255.255.255",
        t.adapter,
    }
    
    if err := runCommand("netsh", args...); err != nil {
        return fmt.Errorf("failed to add route: %w", err)
    }

    t.routes.Store(ip, peerID)
    return nil
}

func (t *winTun) removeRoute(ip string) error {
    // Remove route from Windows routing table
    args := []string{
        "route", "delete",
        ip,
    }
    
    if err := runCommand("netsh", args...); err != nil {
        return fmt.Errorf("failed to remove route: %w", err)
    }

    t.routes.Delete(ip)
    return nil
}

func (t *winTun) readPackets(handler func([]byte, string) error) {
    buffer := make([]byte, t.mtu)
    for {
        select {
        case <-t.stopChan:
            return
        default:
            var read uint32
            err := windows.ReadFile(t.handle, buffer, &read, nil)
            if err != nil {
                continue
            }

            packet := make([]byte, read)
            copy(packet, buffer[:read])

            // Extract destination IP from packet
            if len(packet) < 20 {
                continue
            }
            dstIP := net.IP(packet[16:20]).String()

            // Find peer ID for destination
            if val, ok := t.routes.Load(dstIP); ok {
                if peerID, ok := val.(string); ok {
                    if err := handler(packet, peerID); err != nil {
                        // Log error but continue processing packets
                        fmt.Printf("Error handling packet: %v\n", err)
                    }
                }
            }
        }
    }
}

// Windows-specific helper functions

func createWinTunAdapter(name string) (windows.Handle, error) {
    // Load WinTun driver DLL and create adapter
    // Implementation depends on WinTun driver API
    return 0, fmt.Errorf("WinTun implementation not included")
}

func configureAdapter(name string, ip net.IP, mask net.IPMask) error {
    args := []string{
        "interface", "ipv4", "set",
        "address", name,
        "static", ip.String(),
        fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3]),
    }
    
    return runCommand("netsh", args...)
}

func runCommand(name string, args ...string) error {
    si := &windows.StartupInfo{}
    pi := &windows.ProcessInformation{}

    cmdline := name
    for _, arg := range args {
        cmdline += " " + arg
    }

    err := windows.CreateProcess(
        nil,
        syscall.StringToUTF16Ptr(cmdline),
        nil,
        nil,
        false,
        0,
        nil,
        nil,
        si,
        pi,
    )
    if err != nil {
        return err
    }

    defer windows.CloseHandle(pi.Thread)
    defer windows.CloseHandle(pi.Process)

    status, err := windows.WaitForSingleObject(pi.Process, windows.INFINITE)
    if err != nil {
        return err
    }
    if status != windows.WAIT_OBJECT_0 {
        return fmt.Errorf("process wait failed with status %d", status)
    }

    var exitCode uint32
    err = windows.GetExitCodeProcess(pi.Process, &exitCode)
    if err != nil {
        return err
    }
    if exitCode != 0 {
        return fmt.Errorf("process exited with code %d", exitCode)
    }

    return nil
}
