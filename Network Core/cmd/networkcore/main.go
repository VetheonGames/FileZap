package main

import (
    "context"
    "flag"
    "log"
    "os"
    "os/signal"
    "strings"
    "syscall"

    "github.com/VetheonGames/FileZap/NetworkCore/pkg/network"
    "github.com/multiformats/go-multiaddr"
)

func main() {
    log.Println("Starting FileZap Network Core...")

    // Parse command line flags
    bootstrapNodes := flag.String("bootstrap", "", "Comma-separated list of bootstrap node multiaddrs")
    flag.Parse()

    // Create context that will be cancelled on interrupt
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Create network engine
    engine, err := network.NewNetworkEngine(ctx)
    if err != nil {
        log.Fatalf("Failed to create network engine: %v", err)
    }
    defer engine.Close()

    // Log transport layer addresses
    transportHost := engine.GetTransportHost()
    transportAddrs := transportHost.Addrs()
    transportID := transportHost.ID()
    log.Println("Transport Layer (QUIC/UDP):")
    for _, addr := range transportAddrs {
        log.Printf("  Listening on: %s/p2p/%s", addr, transportID)
    }

    // Log metadata layer addresses
    metadataHost := engine.GetMetadataHost()
    metadataAddrs := metadataHost.Addrs()
    metadataID := metadataHost.ID()
    log.Println("Metadata Layer (TCP):")
    for _, addr := range metadataAddrs {
        log.Printf("  Listening on: %s/p2p/%s", addr, metadataID)
    }

    // Connect to bootstrap nodes if provided
    if *bootstrapNodes != "" {
        peers := strings.Split(*bootstrapNodes, ",")
        for _, peer := range peers {
            addr, err := multiaddr.NewMultiaddr(peer)
            if err != nil {
                log.Printf("Invalid bootstrap address %s: %v", peer, err)
                continue
            }
            if err := engine.Connect(addr); err != nil {
                log.Printf("Failed to connect to bootstrap node %s: %v", peer, err)
            } else {
                log.Printf("Connected to bootstrap node: %s", peer)
            }
        }
    }

    // Wait for interrupt signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan

    log.Println("Shutting down...")
}
