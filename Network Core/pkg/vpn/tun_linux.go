//go:build linux

package vpn

import (
    "fmt"
    "net"
    "os"
    "sync"
    "syscall"
    "unsafe"

    "golang.org/x/sys/unix"
)

const (
    IFF_TUN   = 0x0001
    IFF_NO_PI = 0x1000
)

type linuxTun struct {
    fd        *os.File
    device    string
    network   string
    routes    sync.Map
    stopChan  chan struct{}
}

type ifReq struct {
    Name  [0x10]byte
    Flags uint16
    pad   [0x28 - 0x10 - 2]byte
}

func newTunDevice(cfg TUNConfig) (tunHandle, error) {
    // Open TUN device
    fd, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
    if err != nil {
        return nil, fmt.Errorf("failed to open /dev/net/tun: %w", err)
    }

    // Prepare interface request
    var req ifReq
    copy(req.Name[:], cfg.Name)
    req.Flags = IFF_TUN | IFF_NO_PI

    // Configure TUN interface
    _, _, errno := syscall.Syscall(
        syscall.SYS_IOCTL,
        fd.Fd(),
        uintptr(unix.TUNSETIFF),
        uintptr(unsafe.Pointer(&req)),
    )
    if errno != 0 {
        fd.Close()
        return nil, fmt.Errorf("failed to configure TUN: %w", errno)
    }

    tun := &linuxTun{
        fd:       fd,
        device:   cfg.Name,
        network:  cfg.Network,
        stopChan: make(chan struct{}),
    }

    // Configure interface address and routes
    if err := configureInterface(cfg.Name, cfg.PeerIP, cfg.NetMask); err != nil {
        tun.close()
        return nil, fmt.Errorf("failed to configure interface: %w", err)
    }

    return tun, nil
}

func (t *linuxTun) start(mtu int, handler func([]byte, string) error) error {
    go t.readPackets(mtu, handler)
    return nil
}

func (t *linuxTun) close() error {
    close(t.stopChan)
    return t.fd.Close()
}

func (t *linuxTun) write(packet []byte) error {
    _, err := t.fd.Write(packet)
    return err
}

func (t *linuxTun) updateRoute(ip string, peerID string) error {
    parsedIP := net.ParseIP(ip)
    if parsedIP == nil {
        return fmt.Errorf("invalid IP address: %s", ip)
    }

    // Add route using ip route command
    args := []string{
        "route", "add",
        ip + "/32",
        "dev", t.device,
    }

    if err := runCommand("ip", args...); err != nil {
        return fmt.Errorf("failed to add route: %w", err)
    }

    t.routes.Store(ip, peerID)
    return nil
}

func (t *linuxTun) removeRoute(ip string) error {
    // Remove route using ip route command
    args := []string{
        "route", "del",
        ip + "/32",
    }

    if err := runCommand("ip", args...); err != nil {
        return fmt.Errorf("failed to remove route: %w", err)
    }

    t.routes.Delete(ip)
    return nil
}

func (t *linuxTun) readPackets(mtu int, handler func([]byte, string) error) {
    buffer := make([]byte, mtu)
    for {
        select {
        case <-t.stopChan:
            return
        default:
            n, err := t.fd.Read(buffer)
            if err != nil {
                continue
            }

            packet := make([]byte, n)
            copy(packet, buffer[:n])

            // Extract destination IP from packet
            if len(packet) < 20 {
                continue
            }
            dstIP := net.IP(packet[16:20]).String()

            // Find peer ID for destination
            if val, ok := t.routes.Load(dstIP); ok {
                if peerID, ok := val.(string); ok {
                    if err := handler(packet, peerID); err != nil {
                        fmt.Printf("Error handling packet: %v\n", err)
                    }
                }
            }
        }
    }
}

// Linux-specific helper functions

func configureInterface(name string, ip net.IP, mask net.IPMask) error {
    // Bring interface up
    if err := runCommand("ip", "link", "set", "dev", name, "up"); err != nil {
        return fmt.Errorf("failed to bring interface up: %w", err)
    }

    // Configure IP address
    addr := fmt.Sprintf("%s/%d", ip.String(), networkMaskToCIDR(mask))
    if err := runCommand("ip", "addr", "add", addr, "dev", name); err != nil {
        return fmt.Errorf("failed to set interface address: %w", err)
    }

    return nil
}

func networkMaskToCIDR(mask net.IPMask) int {
    ones, _ := mask.Size()
    return ones
}

func runCommand(name string, args ...string) error {
    attr := &syscall.ProcAttr{
        Files: []uintptr{0, 1, 2},
    }

    pid, err := syscall.ForkExec(name, append([]string{name}, args...), attr)
    if err != nil {
        return err
    }

    var status syscall.WaitStatus
    _, err = syscall.Wait4(pid, &status, 0, nil)
    if err != nil {
        return err
    }

    if !status.Success() {
        return fmt.Errorf("command %s failed with status %d", name, status.ExitStatus())
    }

    return nil
}
