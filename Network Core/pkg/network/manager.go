package network

import (
    "sync"

    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/peer"
    "github.com/libp2p/go-libp2p/core/protocol"
    "github.com/quic-go/quic-go"
)

// NewChunkStore creates a new chunk store
func NewChunkStore(host host.Host) *ChunkStore {
    cs := &ChunkStore{
        chunks:     make(map[string][]byte),
        transfers:  NewTransferManager(host),
        totalSize:  0,
        mu:         sync.RWMutex{},
    }

    // Register chunk protocol handler
    host.SetStreamHandler(protocol.ID(chunkProtocol), cs.handleChunkStream)

    return cs
}

// NewTransferManager creates a new transfer manager
func NewTransferManager(host host.Host) *TransferManager {
    return &TransferManager{
        host:     host,
        sessions: make(map[peer.ID]*quic.Connection),
        mu:       sync.RWMutex{},
    }
}
