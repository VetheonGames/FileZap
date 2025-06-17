package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/VetheonGames/FileZap/NetworkCore/pkg/network"
)

func main() {
    // Parse flags for network configuration
    storageDir := flag.String("storage", "storage", "Directory for storing chunks")
    metadataDir := flag.String("metadata", "metadata", "Directory for storing metadata")
    port := flag.Int("port", 6001, "Port to listen on")
    flag.Parse()

    // Create base context
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Create network config
    cfg := network.DefaultNetworkConfig()
    cfg.ChunkCacheDir = *storageDir
    cfg.MetadataStore = *metadataDir
    cfg.Transport.ListenPort = *port

    // Create network engine
    engine, err := network.NewNetworkEngine(ctx, cfg)
    if err != nil {
        log.Fatalf("Failed to create network engine: %v", err)
    }
    defer engine.Close()

    // Print network information
    log.Printf("Network node started")
    log.Printf("Node ID: %s", engine.GetNodeID())
    log.Printf("Transport addresses:")
    for _, addr := range engine.GetTransportHost().Addrs() {
        log.Printf("  - %s/p2p/%s", addr, engine.GetNodeID())
    }
    log.Printf("Metadata addresses:")
    for _, addr := range engine.GetMetadataHost().Addrs() {
        log.Printf("  - %s/p2p/%s", addr, engine.GetNodeID())
    }

    // Handle signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    // Wait for interrupt
    <-sigChan
    fmt.Println("\nShutting down...")

    // Give pending operations a chance to complete
    time.Sleep(2 * time.Second)
}
