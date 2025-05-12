package validator

import (
    "bytes"
    "crypto/rand"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "time"

    "github.com/VetheonGames/FileZap/NetworkCore/pkg/types"
)

// Client represents a validator network client
type Client struct {
    baseURL    string
    httpClient     *http.Client
    clientID       string
    connected      bool
    address        string
    storageDir     string
    availableChunks []string
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
func NewClient(baseURL string) *Client {
    clientID := generateClientID()
    return &Client{
        baseURL:    "http://" + baseURL,
        httpClient: &http.Client{},
        clientID:   clientID,
        connected:  false,
        address:    baseURL,
    }
}

// SetAddress updates the validator server address
func (c *Client) SetAddress(address string) {
    c.address = address
    c.baseURL = "http://" + address
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
    req, err := http.NewRequest("GET", fmt.Sprintf("%s/file/info/%s", c.baseURL, fileName), nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %v", err)
    }

    req.Header.Set("X-Client-ID", c.clientID)
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to request file info: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
    }

    var fileInfo types.FileInfo
    if err := json.NewDecoder(resp.Body).Decode(&fileInfo); err != nil {
        return nil, fmt.Errorf("failed to decode response: %v", err)
    }

    return &fileInfo, nil
}

// MaintainConnection keeps the connection with the validator alive
func (c *Client) MaintainConnection() {
    ticker := time.NewTicker(30 * time.Second)
    for range ticker.C {
        c.connected = c.IsConnected()
        if !c.connected {
            log.Printf("Lost connection to validator server at %s, attempting to reconnect...", c.address)
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

    reqData, err := json.Marshal(data)
    if err != nil {
        return fmt.Errorf("failed to marshal files data: %v", err)
    }

    req, err := http.NewRequest("POST", c.baseURL+"/files/update", bytes.NewBuffer(reqData))
    if err != nil {
        return fmt.Errorf("failed to create request: %v", err)
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Client-ID", c.clientID)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to update files: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("server returned status %d", resp.StatusCode)
    }

    return nil
}

// IsConnected checks if the client can connect to the validator network
func (c *Client) IsConnected() bool {
    resp, err := c.httpClient.Get(c.baseURL + "/ping")
    if err != nil {
        return false
    }
    defer resp.Body.Close()
    return resp.StatusCode == http.StatusOK
}

// SetStorageDir sets the directory where chunks are stored
func (c *Client) SetStorageDir(dir string) {
    c.storageDir = dir
}

// RegisterAvailableChunks informs the validator of chunks we have available
func (c *Client) RegisterAvailableChunks(chunks []string) error {
    data := struct {
        PeerID   string   `json:"peer_id"`
        Address  string   `json:"address"`
        ChunkIDs []string `json:"chunk_ids"`
    }{
        PeerID:   c.clientID,
        Address:  c.address,
        ChunkIDs: chunks,
    }

    reqData, err := json.Marshal(data)
    if err != nil {
        return fmt.Errorf("failed to marshal chunks data: %v", err)
    }

    req, err := http.NewRequest("POST", c.baseURL+"/chunks/register", bytes.NewBuffer(reqData))
    if err != nil {
        return fmt.Errorf("failed to create request: %v", err)
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Client-ID", c.clientID)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to register chunks: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("server returned status %d", resp.StatusCode)
    }

    c.availableChunks = chunks
    return nil
}

// GetChunkPeers requests a list of peers that have a specific chunk
func (c *Client) GetChunkPeers(chunkID string) ([]types.PeerChunkInfo, error) {
    req, err := http.NewRequest("GET", fmt.Sprintf("%s/chunks/peers/%s", c.baseURL, chunkID), nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %v", err)
    }

    req.Header.Set("X-Client-ID", c.clientID)
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to get chunk peers: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
    }

    var peers []types.PeerChunkInfo
    if err := json.NewDecoder(resp.Body).Decode(&peers); err != nil {
        return nil, fmt.Errorf("failed to decode response: %v", err)
    }

    return peers, nil
}

// UploadZapFile registers a file with the validator network
func (c *Client) UploadZapFile(fileInfo types.FileInfo) error {
    data, err := json.Marshal(fileInfo)
    if err != nil {
        return fmt.Errorf("failed to marshal file info: %v", err)
    }

    resp, err := c.httpClient.Post(c.baseURL+"/file/register", "application/json", bytes.NewBuffer(data))
    if err != nil {
        return fmt.Errorf("failed to register file: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
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

    reqData, err := json.Marshal(data)
    if err != nil {
        return fmt.Errorf("failed to marshal key data: %v", err)
    }

    req, err := http.NewRequest("POST", c.baseURL+"/key/register", bytes.NewBuffer(reqData))
    if err != nil {
        return fmt.Errorf("failed to create request: %v", err)
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Client-ID", c.clientID)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to register key: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
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

    reqData, err := json.Marshal(data)
    if err != nil {
        return "", fmt.Errorf("failed to marshal request data: %v", err)
    }

    req, err := http.NewRequest("POST", c.baseURL+"/key/request", bytes.NewBuffer(reqData))
    if err != nil {
        return "", fmt.Errorf("failed to create request: %v", err)
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Client-ID", c.clientID)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return "", fmt.Errorf("failed to request key: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        var errResp struct {
            Error string `json:"error"`
        }
        if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
            return "", fmt.Errorf("server error: %s", errResp.Error)
        }
        return "", fmt.Errorf("server returned status %d", resp.StatusCode)
    }

    var response struct {
        Key string `json:"key"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        return "", fmt.Errorf("failed to decode response: %v", err)
    }

    return response.Key, nil
}
