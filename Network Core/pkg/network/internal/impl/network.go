package impl

import (
    "context"
    "fmt"

    "github.com/libp2p/go-libp2p/core/peer"
    "github.com/libp2p/go-libp2p/core/host"
    ma "github.com/multiformats/go-multiaddr"

    "github.com/VetheonGames/FileZap/NetworkCore/pkg/vpn"
    "github.com/VetheonGames/FileZap/NetworkCore/pkg/network/api"
    "github.com/VetheonGames/FileZap/NetworkCore/pkg/network/internal"
)

// networkImpl implements the public Network interface using internal components
type networkImpl struct {
	engine internal.NetworkEngine
}

// NewNetwork creates a new Network implementation
func NewNetwork(ctx context.Context, cfg *api.NetworkConfig, enableVPN bool) (api.Network, error) {
// Convert public config to internal config
internalCfg := ConvertConfig(cfg)

	// Create internal engine
	engine, err := internal.NewNetworkEngine(ctx, internalCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create network engine: %w", err)
	}

	return &networkImpl{engine: engine}, nil
}

// Implement Network interface

func (n *networkImpl) Connect(addr ma.Multiaddr) error {
    return n.engine.Connect(addr)
}

func (n *networkImpl) Close() error {
	return n.engine.Close()
}

func (n *networkImpl) GetNodeID() string {
	return n.engine.GetNodeID()
}

func (n *networkImpl) GetTransportHost() host.Host {
	return n.engine.GetTransportHost()
}

func (n *networkImpl) GetMetadataHost() host.Host {
	return n.engine.GetMetadataHost()
}

func (n *networkImpl) GetVPNManager() *vpn.VPNManager {
	return n.engine.GetVPNManager()
}

func (n *networkImpl) Bootstrap(addrs []peer.AddrInfo) error {
    return n.engine.Bootstrap(addrs)
}

func (n *networkImpl) GetPeers() []peer.ID {
    return n.engine.GetPeers()
}

func (n *networkImpl) GetVPNStatus() *api.VPNStatus {
	internalStatus := n.engine.GetVPNStatus()
return ConvertVPNStatus(internalStatus)
}

// File operations

func (n *networkImpl) AddZapFile(manifest *api.ManifestInfo, chunks map[string][]byte) error {
	internalManifest := &internal.ManifestInfo{
		Name:            manifest.Name,
		Owner:           manifest.Owner,
		ChunkHashes:     manifest.ChunkHashes,
		Size:            manifest.Size,
		Created:         manifest.Created,
		Modified:        manifest.Modified,
		ReplicationGoal: manifest.ReplicationGoal,
		UpdatedAt:       manifest.UpdatedAt,
	}
	return n.engine.AddZapFile(internalManifest, chunks)
}

func (n *networkImpl) GetZapFile(name string) (*api.ManifestInfo, map[string][]byte, error) {
	manifest, chunks, err := n.engine.GetZapFile(name)
	if err != nil {
		return nil, nil, err
	}

	apiManifest := &api.ManifestInfo{
		Name:            manifest.Name,
		Owner:           manifest.Owner,
		ChunkHashes:     manifest.ChunkHashes,
		Size:            manifest.Size,
		Created:         manifest.Created,
		Modified:        manifest.Modified,
		ReplicationGoal: manifest.ReplicationGoal,
		UpdatedAt:       manifest.UpdatedAt,
	}
	return apiManifest, chunks, nil
}

func (n *networkImpl) ReportBadFile(name string, reason string) error {
	return n.engine.ReportBadFile(name, reason)
}

// Storage operations

func (n *networkImpl) RegisterStorageNode() error {
	return n.engine.RegisterStorageNode()
}

func (n *networkImpl) UnregisterStorageNode() error {
	return n.engine.UnregisterStorageNode()
}

func (n *networkImpl) GetStorageRequest() (*api.StorageRequest, error) {
	req, err := n.engine.GetStorageRequest()
	if err != nil {
		return nil, err
	}
	return &api.StorageRequest{
		ChunkHash: req.ChunkHash,
		Data:      req.Data,
		Size:      req.Size,
		Owner:     req.Owner,
	}, nil
}

func (n *networkImpl) ValidateChunkRequest(req *api.StorageRequest) error {
	internalReq := &internal.StorageRequest{
		ChunkHash: req.ChunkHash,
		Data:      req.Data,
		Size:      req.Size,
		Owner:     req.Owner,
	}
	return n.engine.ValidateChunkRequest(internalReq)
}

func (n *networkImpl) StoreChunk(req *api.StorageRequest) error {
	internalReq := &internal.StorageRequest{
		ChunkHash: req.ChunkHash,
		Data:      req.Data,
		Size:      req.Size,
		Owner:     req.Owner,
	}
	return n.engine.StoreChunk(internalReq)
}

func (n *networkImpl) RejectStorageRequest(req *api.StorageRequest, reason string) error {
	internalReq := &internal.StorageRequest{
		ChunkHash: req.ChunkHash,
		Data:      req.Data,
		Size:      req.Size,
		Owner:     req.Owner,
	}
	return n.engine.RejectStorageRequest(internalReq, reason)
}


func (n *networkImpl) AcknowledgeStorage(req *api.StorageRequest) error {
	internalReq := &internal.StorageRequest{
		ChunkHash: req.ChunkHash,
		Data:      req.Data,
		Size:      req.Size,
		Owner:     req.Owner,
	}
	return n.engine.AcknowledgeStorage(internalReq)
}
