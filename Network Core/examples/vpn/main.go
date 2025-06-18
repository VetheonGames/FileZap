package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "os"
    "os/signal"
    "strings"
    "syscall"
    "time"

    "github.com/libp2p/go-libp2p"
    "github.com/libp2p/go-libp2p/core/crypto"
    dht "github.com/libp2p/go-libp2p-kad-dht"
    pubsub "github.com/libp2p/go-libp2p-pubsub"
    "github.com/libp2p/go-libp2p/p2p/security/noise"
    libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"

    "github.com/VetheonGames/FileZap/NetworkCore/pkg/vpn"
)

func main() {
    // Parse command line flags
    listenPort := flag.Int("port", 0, "Port to listen on")
    networkKey := flag.String("network-key", "MeshGenesisKey", "Shared network key for peer discovery")
    networkCIDR := flag.String("network", "10.42.0.0/16", "VPN network CIDR")
    ifName := flag.String("interface", "tun0", "TUN interface name")
    flag.Parse()

    // Create context
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

// Generate deterministic key from network key
    priv, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, -1,
        strings.NewReader(*networkKey))
    if err != nil {
        log.Fatal(err)
    }

    // Setup libp2p host with QUIC transport
    host, err := libp2p.New(
        libp2p.ListenAddrStrings(
            fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic", *listenPort),
            fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *listenPort),
        ),
        libp2p.Identity(priv),
        libp2p.Transport(libp2pquic.NewTransport),
        libp2p.DefaultTransports,
        libp2p.Security(noise.ID, noise.New),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer host.Close()

    // Create DHT
    kadDHT, err := dht.New(ctx, host,
        dht.Mode(dht.ModeServer),
        dht.ProtocolPrefix("/meshvpn"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer kadDHT.Close()

    // Bootstrap DHT
    if err := kadDHT.Bootstrap(ctx); err != nil {
        log.Fatal(err)
    }

    // Create pubsub
    ps, err := pubsub.NewGossipSub(ctx, host,
        pubsub.WithMessageSigning(true),
        pubsub.WithStrictSignatureVerification(true),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Create VPN
    vpnConfig := vpn.DefaultConfig()
    vpnConfig.NetworkCIDR = *networkCIDR
    vpnConfig.InterfaceName = *ifName

    vpnManager, err := vpn.NewVPNManager(ctx, host, vpnConfig)
    if err != nil {
        log.Fatal(err)
    }
    defer vpnManager.Close()

    // Create discovery service
    discovery, err := vpn.NewDiscovery(ctx, host, kadDHT, ps, vpnManager)
    if err != nil {
        log.Fatal(err)
    }
    defer discovery.Close()

    // Print node info
    log.Printf("Peer ID: %s", host.ID().String())
    log.Printf("Listening on:")
    for _, addr := range host.Addrs() {
        log.Printf("  %s/p2p/%s", addr, host.ID())
    }

    // Wait for interrupt
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan

    log.Println("Shutting down...")
    
    // Give time for cleanup
    time.Sleep(time.Second)
}
