package validator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// Client handles communication with validator servers
type Client struct {
	serverAddress string
	connected     bool
	httpClient    *http.Client
	mu            sync.RWMutex

	// State information
	myID          string
	availableZaps []string
}

// GetAddress returns the validator server address
func (c *Client) GetAddress() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverAddress
}

// SetAddress updates the validator server address
func (c *Client) SetAddress(address string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.serverAddress = address
}

type FileInfo struct {
	Name      string   `json:"name"`
	ChunkIDs  []string `json:"chunk_ids"`
	Available bool     `json:"available"`
}

// NewClient creates a new validator client
func NewClient(serverAddress string) *Client {
	return &Client{
		serverAddress: serverAddress,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// MaintainConnection periodically checks connection with validator
func (c *Client) MaintainConnection() {
	ticker := time.NewTicker(30 * time.Second)
	for range ticker.C {
		err := c.ping()
		c.mu.Lock()
		c.connected = err == nil
		c.mu.Unlock()

		if err != nil {
			log.Printf("Failed to ping validator: %v", err)
			continue
		}

		// Update our status with the validator
		err = c.updateStatus()
		if err != nil {
			log.Printf("Failed to update status: %v", err)
		}
	}
}

// ping checks if the validator is reachable
func (c *Client) ping() error {
	resp, err := c.httpClient.Get(fmt.Sprintf("http://%s/ping", c.serverAddress))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// updateStatus sends our current state to the validator
func (c *Client) updateStatus() error {
	c.mu.RLock()
	data := map[string]interface{}{
		"peer_id":        c.myID,
		"available_zaps": c.availableZaps,
	}
	c.mu.RUnlock()

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Post(
		fmt.Sprintf("http://%s/peer/status", c.serverAddress),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// UploadZapFile notifies validator about a new .zap file
func (c *Client) UploadZapFile(fileInfo FileInfo) error {
	jsonData, err := json.Marshal(fileInfo)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Post(
		fmt.Sprintf("http://%s/file/register", c.serverAddress),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// RequestZapFile requests information about where to find a .zap file
func (c *Client) RequestZapFile(fileName string) (*FileInfo, error) {
	resp, err := c.httpClient.Get(
		fmt.Sprintf("http://%s/file/info/%s", c.serverAddress, fileName),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var fileInfo FileInfo
	err = json.NewDecoder(resp.Body).Decode(&fileInfo)
	if err != nil {
		return nil, err
	}

	return &fileInfo, nil
}

// UpdateAvailableZaps updates the list of .zap files we have available
func (c *Client) UpdateAvailableZaps(zaps []string) {
	c.mu.Lock()
	c.availableZaps = zaps
	c.mu.Unlock()
}

// IsConnected returns the current connection state
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}
