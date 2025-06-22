package network

import (
    "context"

    "github.com/libp2p/go-libp2p-kad-dht"
    "github.com/libp2p/go-libp2p-pubsub"
    "github.com/libp2p/go-libp2p/core/host"
)

// managerFactory creates network component instances
type managerFactory struct{}

// NewFactory creates a new manager factory
func NewFactory() *managerFactory {
    return &managerFactory{}
}

// CreateGossipManager creates a new gossip manager instance
func (f *managerFactory) CreateGossipManager(ctx context.Context, h host.Host, ps *pubsub.PubSub) (GossipManager, error) {
    return NewGossipManager(ctx, h, ps)
}

// CreateManifestManager creates a new manifest manager instance
func (f *managerFactory) CreateManifestManager(ctx context.Context, h host.Host, dht *dht.IpfsDHT, ps *pubsub.PubSub) (*ManifestManager, error) {
    return NewManifestManager(ctx, h, dht, ps)
}

// CreateQuorumManager creates a new quorum manager instance
func (f *managerFactory) CreateQuorumManager(ctx context.Context, h host.Host, ps *pubsub.PubSub, g GossipManager) (QuorumManager, error) {
    return newQuorumManagerImpl(ctx, h, ps, g)
}
