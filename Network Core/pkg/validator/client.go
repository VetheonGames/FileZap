package validator

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/VetheonGames/FileZap/NetworkCore/pkg/overlay"
	"github.com/VetheonGames/FileZap/NetworkCore/pkg/types"
)

// Client represents a validator network client
type Client struct {
	network         *overlay.NetworkAdapter
	validatorID     string
	clientID        string
	connected       bool
	storageDir      string
	availableChunks []string
	ctx             context.Context
	cancel          context.CancelFunc
}

// ZapFileInfo represents detailed information about a .zap file
type ZapFileInfo struct {
	Name      string   `json:"name"`
	ChunkIDs  []string `json:"chunk_ids"`
	Available bool     `json:"available"`
	Size      int64    `json:"size"`
	CreatedAt string   `json:"created_at"`
}

// NewClient creates a new validator client
func NewClient(validatorID string) (*Client, error) {
	ctx, cancel := context.WithCancel(context.Background())

	network, err := overlay.NewNetworkAdapter(ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create network adapter: %v", err)
	}

	clientID := generateClientID()
	return &Client{
		network:     network,
		validatorID: validatorID,
		clientID:    clientID,
		connected:   false,
		ctx:         ctx,
		cancel:      cancel,
	}, nil
}

// Close shuts down the client
func (c *Client) Close() error {
	c.cancel()
	return c.network.Close()
}

// generateClientID creates a unique client identifier
func generateClientID() string {
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		return fmt.Sprintf("client-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("client-%x", id)
}

// RequestZapFile requests information about a .zap file from the validator
func (c *Client) RequestZapFile(fileName string) (*types.FileInfo, error) {
	resp, err := c.network.SendRequest(c.validatorID, "GET", fmt.Sprintf("/file/info/%s", fileName), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var fileInfo types.FileInfo
	if err := json.Unmarshal(resp.Body, &fileInfo); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &fileInfo, nil
}

// MaintainConnection keeps the connection with the validator alive
func (c *Client) MaintainConnection() {
	ticker := time.NewTicker(30 * time.Second)
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.connected = c.IsConnected()
			if !c.connected {
				log.Printf("Lost connection to validator, attempting to reconnect...")
			}
		}
	}
}

// UpdateAvailableZaps updates the validator with our available zap files
func (c *Client) UpdateAvailableZaps(files []types.FileInfo) error {
	data := struct {
		Files []types.FileInfo `json:"files"`
	}{
		Files: files,
	}

	resp, err := c.network.SendRequest(c.validatorID, "POST", "/files/update", data)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return nil
}

// IsConnected checks if the client can connect to the validator network
func (c *Client) IsConnected() bool {
	resp, err := c.network.SendRequest(c.validatorID, "GET", "/ping", nil)
	if err != nil {
		return false
	}
	return resp.StatusCode == 200
}

// SetStorageDir sets the directory where chunks are stored
func (c *Client) SetStorageDir(dir string) {
	c.storageDir = dir
}

// RegisterAvailableChunks informs the validator of chunks we have available
func (c *Client) RegisterAvailableChunks(chunks []string) error {
	data := struct {
		PeerID   string   `json:"peer_id"`
		ChunkIDs []string `json:"chunk_ids"`
	}{
		PeerID:   c.network.GetNodeID(),
		ChunkIDs: chunks,
	}

	resp, err := c.network.SendRequest(c.validatorID, "POST", "/chunks/register", data)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	c.availableChunks = chunks
	return nil
}

// GetChunkPeers requests a list of peers that have a specific chunk
func (c *Client) GetChunkPeers(chunkID string) ([]types.PeerChunkInfo, error) {
	resp, err := c.network.SendRequest(c.validatorID, "GET", fmt.Sprintf("/chunks/peers/%s", chunkID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var peers []types.PeerChunkInfo
	if err := json.Unmarshal(resp.Body, &peers); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return peers, nil
}

// UploadZapFile registers a file with the validator network
func (c *Client) UploadZapFile(fileInfo types.FileInfo) error {
	resp, err := c.network.SendRequest(c.validatorID, "POST", "/file/register", fileInfo)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return nil
}

// RegisterEncryptionKey registers an encryption key with the validator network
func (c *Client) RegisterEncryptionKey(fileID string, key string, publicKey []byte) error {
	data := struct {
		FileID    string `json:"file_id"`
		Key       string `json:"key"`
		PublicKey []byte `json:"public_key"`
		ClientID  string `json:"client_id"`
	}{
		FileID:    fileID,
		Key:       key,
		PublicKey: publicKey,
		ClientID:  c.clientID,
	}

	resp, err := c.network.SendRequest(c.validatorID, "POST", "/key/register", data)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return nil
}

// RequestDecryptionKey requests a decryption key from the validator network
func (c *Client) RequestDecryptionKey(fileID string, publicKey []byte) (string, error) {
	data := struct {
		FileID    string `json:"file_id"`
		ClientID  string `json:"client_id"`
		PublicKey []byte `json:"public_key"`
	}{
		FileID:    fileID,
		ClientID:  c.clientID,
		PublicKey: publicKey,
	}

	resp, err := c.network.SendRequest(c.validatorID, "POST", "/key/request", data)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var response struct {
		Key   string `json:"key"`
		Error string `json:"error,omitempty"`
	}

	if err := json.Unmarshal(resp.Body, &response); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	if response.Error != "" {
		return "", fmt.Errorf("server error: %s", response.Error)
	}

	return response.Key, nil
}
