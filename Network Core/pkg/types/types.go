package types

// PeerChunkInfo represents chunk availability information for a peer
type PeerChunkInfo struct {
	PeerID    string   `json:"peer_id"`
	ChunkIDs  []string `json:"chunk_ids"`
	Address   string   `json:"address"`
	Available bool     `json:"available"`
}

// FileInfo represents a registered .zap file
type FileInfo struct {
	Name      string          `json:"name"`
	ChunkIDs  []string        `json:"chunk_ids"`
	Available bool            `json:"available"`
	Peers     []PeerChunkInfo `json:"peers"`
}

// ChunkStorageConfig represents chunk storage configuration
type ChunkStorageConfig struct {
	StorageDir     string `json:"storage_dir"`
	MaxStorageSize int64  `json:"max_storage_size"` // in bytes
	AutoCleanup    bool   `json:"auto_cleanup"`     // remove old chunks when space is needed
}
