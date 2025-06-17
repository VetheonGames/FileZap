package client

import (
	"context"
"fmt"
"os"
"path/filepath"
"time"

"github.com/VetheonGames/FileZap/Divider/pkg/chunking"
"github.com/VetheonGames/FileZap/Divider/pkg/encryption"
"github.com/VetheonGames/FileZap/Divider/pkg/zap"
"github.com/VetheonGames/FileZap/NetworkCore/pkg/network"
"github.com/libp2p/go-libp2p/core/peer"
)

// ClientConfig holds configuration for the FileZap client
type ClientConfig struct {
    // Storage configuration
    StorageEnabled    bool    // Whether this client acts as a storage node
    StorageDirectory  string  // Directory to store chunks in
    MaxStorageSize    int64   // Maximum storage space in bytes (default 1GB)
    MinFreeSpace     int64   // Minimum free space to maintain in bytes

    // Network configuration
    WorkDirectory    string  // Directory for temporary files and .zap files
    BootstrapPeers  []string // Initial peers to connect to
    ListenAddresses []string // Addresses to listen on
}

// StorageStats represents storage node statistics
type StorageStats struct {
    UsedSpace     int64
    MaxSpace      int64
    ChunkCount    int
    RequestCount  int
    Uptime        float64
}

// DefaultClientConfig returns the default configuration
func DefaultClientConfig() *ClientConfig {
    return &ClientConfig{
        StorageEnabled:   false,
        StorageDirectory: "chunks",
        MaxStorageSize:   1024 * 1024 * 1024, // 1GB
        MinFreeSpace:     100 * 1024 * 1024,  // 100MB
        WorkDirectory:    "work",
    }
}

// FileZapClient handles all FileZap operations
type FileZapClient struct {
    ctx      context.Context
    cancel   context.CancelFunc
    network  *network.NetworkEngine
    config   *ClientConfig
    isStorer bool
}

// NewFileZapClient creates a new FileZap client with the given configuration
func NewFileZapClient(config *ClientConfig) (*FileZapClient, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize network engine
	netCfg := network.DefaultNetworkConfig()
	netEngine, err := network.NewNetworkEngine(ctx, netCfg)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create network engine: %w", err)
	}

// Create required directories
if err := os.MkdirAll(config.WorkDirectory, 0755); err != nil {
    cancel()
    return nil, fmt.Errorf("failed to create work directory: %w", err)
}
if config.StorageEnabled {
    if err := os.MkdirAll(config.StorageDirectory, 0755); err != nil {
        cancel()
        return nil, fmt.Errorf("failed to create storage directory: %w", err)
    }
}

client := &FileZapClient{
    ctx:     ctx,
    cancel:  cancel,
    network: netEngine,
    config:  config,
}

// Enable storage if configured
if config.StorageEnabled {
    if err := client.EnableStorageNode(); err != nil {
        cancel()
        return nil, fmt.Errorf("failed to enable storage node: %w", err)
    }
}

return client, nil
}

// UploadFile processes and uploads a file to the network
func (c *FileZapClient) UploadFile(filePath string) error {
// Create temporary directory for chunks
chunksDir := filepath.Join(c.config.WorkDirectory, "temp_chunks")
	if err := os.MkdirAll(chunksDir, 0755); err != nil {
		return fmt.Errorf("failed to create chunks directory: %w", err)
	}
	defer os.RemoveAll(chunksDir)

	// Split file into chunks
	chunks, err := chunking.SplitFile(filePath, chunking.DefaultChunkSize, chunksDir)
	if err != nil {
		return fmt.Errorf("failed to split file: %w", err)
	}

	// Generate encryption key
	key, err := encryption.GenerateKey()
	if err != nil {
		return fmt.Errorf("failed to generate encryption key: %w", err)
	}

	// Process chunks and build manifest
	var chunkMetadata []zap.ChunkMetadata
	chunkData := make(map[string][]byte)

	for _, chunk := range chunks {
		// Read chunk
		data, err := os.ReadFile(chunk.Filename)
		if err != nil {
			return fmt.Errorf("failed to read chunk: %w", err)
		}

		// Encrypt chunk
		encrypted, err := encryption.Encrypt(data, key)
		if err != nil {
			return fmt.Errorf("failed to encrypt chunk: %w", err)
		}

		// Create chunk metadata
		chunkMeta := zap.ChunkMetadata{
			Index: chunk.Index,
			Hash:  chunk.Hash,
			Size:  chunk.Size,
		}

		// Generate unique encrypted hash
		if err := chunkMeta.UpdateEncryptedHash(encrypted); err != nil {
			return fmt.Errorf("failed to generate encrypted hash: %w", err)
		}

		chunkMetadata = append(chunkMetadata, chunkMeta)
		chunkData[chunkMeta.EncryptedHash] = encrypted
	}

	// Generate unique ID for the file
	id, err := zap.GenerateID()
	if err != nil {
		return fmt.Errorf("failed to generate file ID: %w", err)
	}

	// Create manifest
	manifest := &network.ManifestInfo{
		Name:            id,
		ChunkHashes:     make([]string, len(chunkMetadata)),
		ReplicationGoal: network.DefaultReplicationGoal,
		Size:            int64(len(chunks)) * chunking.DefaultChunkSize,
	}

	// Add chunk hashes to manifest
	for i, meta := range chunkMetadata {
		manifest.ChunkHashes[i] = meta.EncryptedHash
	}

	// Upload to network
	if err := c.network.AddZapFile(manifest, chunkData); err != nil {
		return fmt.Errorf("failed to upload to network: %w", err)
	}

	// Create local .zap file for reference
	metadata := &zap.FileMetadata{
		ID:            id,
		OriginalName:  filepath.Base(filePath),
		ChunkCount:    len(chunks),
		TotalSize:     manifest.Size,
		EncryptionKey: key,
		Chunks:        chunkMetadata,
	}

if err := zap.CreateZapFile(metadata, c.config.WorkDirectory); err != nil {
		return fmt.Errorf("failed to create local .zap file: %w", err)
	}

	return nil
}

// DownloadFile downloads and reconstructs a file from the network
func (c *FileZapClient) DownloadFile(zapPath, outputDir string) error {
	// Read zap file
	metadata, err := zap.ReadZapFile(zapPath)
	if err != nil {
		return fmt.Errorf("failed to read zap file: %w", err)
	}

	// Get chunks from network
	_, chunks, err := c.network.GetZapFile(metadata.ID)
	if err != nil {
		return fmt.Errorf("failed to get file from network: %w", err)
	}

	// Create temp directory for decrypted chunks
	tempDir := filepath.Join(outputDir, "temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Process chunks
	var chunkInfos []chunking.ChunkInfo
	for _, meta := range metadata.Chunks {
		encryptedData := chunks[meta.EncryptedHash]

		// Decrypt chunk
		decrypted, err := encryption.Decrypt(encryptedData, metadata.EncryptionKey)
		if err != nil {
			return fmt.Errorf("failed to decrypt chunk: %w", err)
		}

		// Save decrypted chunk
		chunkPath := filepath.Join(tempDir, meta.Hash)
		if err := os.WriteFile(chunkPath, decrypted, 0644); err != nil {
			return fmt.Errorf("failed to write decrypted chunk: %w", err)
		}

		chunkInfos = append(chunkInfos, chunking.ChunkInfo{
			Index:    meta.Index,
			Hash:     meta.Hash,
			Size:     meta.Size,
			Filename: chunkPath,
		})
	}

	// Reassemble file
	outputPath := filepath.Join(outputDir, metadata.OriginalName)
	if err := chunking.ReassembleFile(chunkInfos, outputPath); err != nil {
		return fmt.Errorf("failed to reassemble file: %w", err)
	}

	return nil
}

// EnableStorageNode enables the client to act as a storage node
func (c *FileZapClient) EnableStorageNode() error {
    c.isStorer = true
    
    // Register as storage node in the network
    if err := c.network.RegisterStorageNode(); err != nil {
        c.isStorer = false
        return fmt.Errorf("failed to register as storage node: %w", err)
    }

    // Start chunk acceptance loop
    go c.handleChunkStorage()
    return nil
}

// DisableStorageNode disables storage node functionality
func (c *FileZapClient) DisableStorageNode() error {
    c.isStorer = false
    
    // Unregister from network
    if err := c.network.UnregisterStorageNode(); err != nil {
        return fmt.Errorf("failed to unregister storage node: %w", err)
    }
    return nil
}

// handleChunkStorage processes incoming chunk storage requests when acting as a storage node
func (c *FileZapClient) handleChunkStorage() {
    for c.isStorer {
        select {
        case <-c.ctx.Done():
            return
        default:
            // Process any pending storage requests from network
            if request, err := c.network.GetStorageRequest(); err == nil {
                c.processStorageRequest(request)
            }
            time.Sleep(100 * time.Millisecond) // Prevent busy loop
        }
    }
}

// processStorageRequest handles an individual chunk storage request
func (c *FileZapClient) processStorageRequest(request *network.StorageRequest) {
    // Verify we can store this chunk
    if err := c.network.ValidateChunkRequest(request); err != nil {
        c.network.RejectStorageRequest(request, err.Error())
        return
    }

    // Store the chunk
    if err := c.network.StoreChunk(request); err != nil {
        c.network.RejectStorageRequest(request, err.Error())
        return
    }

    // Acknowledge successful storage
    c.network.AcknowledgeStorage(request)
}

// GetNodeID returns this client's node ID
func (c *FileZapClient) GetNodeID() string {
    return c.network.GetNodeID()
}

// GetPeers returns a list of connected peers
func (c *FileZapClient) GetPeers() []peer.ID {
    return c.network.GetPeers()
}

// GetStorageStats returns current storage node statistics
func (c *FileZapClient) GetStorageStats() *StorageStats {
    stats := &StorageStats{
        UsedSpace: 0,
        MaxSpace: c.config.MaxStorageSize,
        ChunkCount: 0,
        RequestCount: 0,
        Uptime: 100.0,
    }

    // Get actual stats from network
    if c.isStorer {
        chunks := c.network.GetStoredChunks()
        for _, chunk := range chunks {
            stats.UsedSpace += int64(len(chunk))
            stats.ChunkCount++
        }
        stats.RequestCount = c.network.GetRequestCount()
        // TODO: Calculate real uptime
    }

    return stats
}

// UpdateConfig updates the client configuration
func (c *FileZapClient) UpdateConfig(cfg *ClientConfig) error {
    c.config = cfg
    // Apply relevant settings to network engine
    // TODO: Implement network config update
    return nil
}

// ReportBadFile reports a malicious file to the network
func (c *FileZapClient) ReportBadFile(fileID string, reason string) error {
    return c.network.ReportBadFile(fileID, reason)
}

// Context returns the client's context
func (c *FileZapClient) Context() context.Context {
    return c.ctx
}

// Close shuts down the client
func (c *FileZapClient) Close() error {
    c.cancel()
    return c.network.Close()
}
