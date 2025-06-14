package network

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
"github.com/libp2p/go-libp2p/core/peer"
"github.com/libp2p/go-libp2p/p2p/security/noise"
ma "github.com/multiformats/go-multiaddr"
)

// NewNetworkEngine creates a new P2P network engine
func NewNetworkEngine(ctx context.Context) (*NetworkEngine, error) {
// Create transport node for chunk transfer
transportNode, err := newNetworkNode(ctx, []string{
"/ip4/127.0.0.1/tcp/0",
}, false)
if err != nil {
return nil, fmt.Errorf("failed to create transport node: %w", err)
}

// Create metadata node for manifest sharing
metadataNode, err := newNetworkNode(ctx, []string{
"/ip4/127.0.0.1/tcp/0",
}, false)
if err != nil {
return nil, fmt.Errorf("failed to create metadata node: %w", err)
}

	// Initialize components
	manifests := NewManifestManager(ctx, metadataNode.host.ID(), metadataNode.dht, metadataNode.pubsub)
	chunkStore := NewChunkStore(transportNode.host)

	return &NetworkEngine{
		ctx:           ctx,
		transportNode: transportNode,
		metadataNode:  metadataNode,
		manifests:     manifests,
		chunkStore:    chunkStore,
	}, nil
}

// newNetworkNode creates a new libp2p node with DHT and pubsub
func newNetworkNode(ctx context.Context, listenAddrs []string, useQuic bool) (*NetworkNode, error) {
opts := []libp2p.Option{
libp2p.ListenAddrStrings(listenAddrs...),
libp2p.Security(noise.ID, noise.New),
libp2p.DefaultTransports,
}

// Create libp2p host
	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

// Create DHT
kdht, err := dht.New(ctx, h, 
    dht.Mode(dht.ModeServer),
    dht.BootstrapPeers(dht.GetDefaultBootstrapPeerAddrInfos()...),
    dht.ProtocolPrefix("/filezap"),
)
if err != nil {
    return nil, fmt.Errorf("failed to create DHT: %w", err)
}

// Bootstrap the DHT with retry
bootstrapCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()

if err := kdht.Bootstrap(bootstrapCtx); err != nil {
    return nil, fmt.Errorf("failed to bootstrap DHT: %w", err)
}

// Wait for at least one connection
connected := make(chan struct{})
go func() {
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()
    for {
        select {
        case <-bootstrapCtx.Done():
            return
        case <-ticker.C:
            if len(h.Network().Peers()) > 0 {
                close(connected)
                return
            }
        }
    }
}()

select {
case <-connected:
    // Successfully connected
case <-bootstrapCtx.Done():
    if len(h.Network().Peers()) == 0 {
        return nil, fmt.Errorf("failed to connect to any peers during bootstrap")
    }
}

	// Create pubsub
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub: %w", err)
	}

	// Create overlay network
	overlay := &OverlayNetwork{
		neighbors: make(map[peer.ID]time.Time),
		maxPeers:  50,
	}

	node := &NetworkNode{
		host:    h,
		dht:     kdht,
		pubsub:  ps,
		overlay: overlay,
	}
	overlay.node = node

	return node, nil
}

// Connect connects to a peer using their multiaddr on both transport and metadata networks
func (e *NetworkEngine) Connect(addr ma.Multiaddr) error {
	peerInfo, err := peer.AddrInfoFromP2pAddr(addr)
	if err != nil {
		return fmt.Errorf("invalid peer address: %w", err)
	}

	// Connect on transport network
	if err := e.transportNode.host.Connect(e.ctx, *peerInfo); err != nil {
		return fmt.Errorf("failed to connect transport: %w", err)
	}

	// Connect on metadata network
	if err := e.metadataNode.host.Connect(e.ctx, *peerInfo); err != nil {
		return fmt.Errorf("failed to connect metadata: %w", err)
	}

	return nil
}

// Bootstrap connects to bootstrap nodes to join both networks
func (e *NetworkEngine) Bootstrap(bootstrapPeers []ma.Multiaddr) error {
	for _, addr := range bootstrapPeers {
		if err := e.Connect(addr); err != nil {
			return fmt.Errorf("failed to connect to bootstrap peer %s: %w", addr, err)
		}

		// Update overlay network neighbors
		peerInfo, _ := peer.AddrInfoFromP2pAddr(addr)
		e.transportNode.overlay.AddNeighbor(peerInfo.ID)
		e.metadataNode.overlay.AddNeighbor(peerInfo.ID)
	}
	return nil
}

// AddZapFile adds a new .zap file to the network
func (e *NetworkEngine) AddZapFile(manifest *ManifestInfo, chunks map[string][]byte) error {
	// Store chunks in the chunk store
	for hash, data := range chunks {
		e.chunkStore.Store(hash, data)
	}

	// Store and replicate manifest
	return e.manifests.AddManifest(manifest)
}

// GetZapFile retrieves a .zap file from the network
func (e *NetworkEngine) GetZapFile(name string) (*ManifestInfo, map[string][]byte, error) {
	// Get manifest from DHT
	manifest, err := e.manifests.GetManifest(name)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get manifest: %w", err)
	}

	// Download chunks
	chunks := make(map[string][]byte)
	for _, hash := range manifest.ChunkHashes {
		data, err := e.chunkStore.transfers.Download(manifest.Owner, hash)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to download chunk %s: %w", hash, err)
		}
		chunks[hash] = data
	}

	return manifest, chunks, nil
}

// GetTransportHost returns the transport network's libp2p host instance
func (e *NetworkEngine) GetTransportHost() host.Host {
	return e.transportNode.host
}

// GetMetadataHost returns the metadata network's libp2p host instance
func (e *NetworkEngine) GetMetadataHost() host.Host {
	return e.metadataNode.host
}

// Close shuts down the network engine
func (e *NetworkEngine) Close() error {
	if err := e.transportNode.dht.Close(); err != nil {
		return fmt.Errorf("failed to close transport DHT: %w", err)
	}
	if err := e.transportNode.host.Close(); err != nil {
		return fmt.Errorf("failed to close transport host: %w", err)
	}
	if err := e.metadataNode.dht.Close(); err != nil {
		return fmt.Errorf("failed to close metadata DHT: %w", err)
	}
	if err := e.metadataNode.host.Close(); err != nil {
		return fmt.Errorf("failed to close metadata host: %w", err)
	}
	return nil
}

// AddNeighbor adds a peer to the overlay network
func (o *OverlayNetwork) AddNeighbor(p peer.ID) {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Only add if we haven't reached max peers
	if len(o.neighbors) < o.maxPeers {
		if err := o.node.host.Network().ClosePeer(p); err != nil {
			return
		}
		o.neighbors[p] = time.Now()
	}
}
