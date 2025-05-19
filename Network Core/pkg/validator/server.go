package validator

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"

    "github.com/VetheonGames/FileZap/NetworkCore/pkg/overlay"
    "github.com/VetheonGames/FileZap/NetworkCore/pkg/types"
)

// Server represents a validator server that uses the overlay network
type Server struct {
    network     *overlay.ServerAdapter
    ctx         context.Context
    cancel      context.CancelFunc
    files       map[string]*types.FileInfo
    chunks      map[string][]types.PeerChunkInfo
    keys        map[string]string
    publicKeys  map[string][]byte
}

// NewServer creates a new validator server
func NewServer(ctx context.Context) (*Server, error) {
    ctx, cancel := context.WithCancel(ctx)
    
    network, err := overlay.NewServerAdapter(ctx)
    if err != nil {
        cancel()
        return nil, fmt.Errorf("failed to create network adapter: %v", err)
    }

    server := &Server{
        network:    network,
        ctx:       ctx,
        cancel:    cancel,
        files:     make(map[string]*types.FileInfo),
        chunks:    make(map[string][]types.PeerChunkInfo),
        keys:      make(map[string]string),
        publicKeys: make(map[string][]byte),
    }

    // Register handlers
    server.registerHandlers()

    return server, nil
}

// Close shuts down the server
func (s *Server) Close() error {
    s.cancel()
    return s.network.Close()
}

// GetNodeID returns the server's overlay node ID
func (s *Server) GetNodeID() string {
    return s.network.GetNodeID()
}

func (s *Server) registerHandlers() {
    // File operations
    s.network.HandleFunc("GET", "/file/info/{name}", s.handleGetFileInfo)
    s.network.HandleFunc("POST", "/file/register", s.handleRegisterFile)
    s.network.HandleFunc("POST", "/files/update", s.handleUpdateFiles)
    
    // Chunk operations
    s.network.HandleFunc("POST", "/chunks/register", s.handleRegisterChunks)
    s.network.HandleFunc("GET", "/chunks/peers/{id}", s.handleGetChunkPeers)
    
    // Key operations
    s.network.HandleFunc("POST", "/key/register", s.handleRegisterKey)
    s.network.HandleFunc("POST", "/key/request", s.handleRequestKey)
    
    // Health check
    s.network.HandleFunc("GET", "/ping", s.handlePing)
}

func (s *Server) handleGetFileInfo(r *overlay.Request) (*overlay.Response, error) {
    fileName := r.Path[len("/file/info/"):]
    fileInfo, exists := s.files[fileName]
    if !exists {
        return &overlay.Response{
            StatusCode: http.StatusNotFound,
            Body:      []byte(`{"error":"file not found"}`),
        }, nil
    }

    data, err := json.Marshal(fileInfo)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal file info: %v", err)
    }

    return &overlay.Response{
        StatusCode: http.StatusOK,
        Body:      data,
    }, nil
}

func (s *Server) handleRegisterFile(r *overlay.Request) (*overlay.Response, error) {
    var fileInfo types.FileInfo
    if err := json.Unmarshal(r.Body, &fileInfo); err != nil {
        return nil, fmt.Errorf("failed to unmarshal request: %v", err)
    }

    s.files[fileInfo.Name] = &fileInfo

    return &overlay.Response{
        StatusCode: http.StatusOK,
        Body:      []byte(`{"status":"ok"}`),
    }, nil
}

func (s *Server) handleUpdateFiles(r *overlay.Request) (*overlay.Response, error) {
    var data struct {
        Files []types.FileInfo `json:"files"`
    }
    if err := json.Unmarshal(r.Body, &data); err != nil {
        return nil, fmt.Errorf("failed to unmarshal request: %v", err)
    }

    for _, file := range data.Files {
        s.files[file.Name] = &file
    }

    return &overlay.Response{
        StatusCode: http.StatusOK,
        Body:      []byte(`{"status":"ok"}`),
    }, nil
}

func (s *Server) handleRegisterChunks(r *overlay.Request) (*overlay.Response, error) {
    var data struct {
        PeerID   string   `json:"peer_id"`
        ChunkIDs []string `json:"chunk_ids"`
    }
    if err := json.Unmarshal(r.Body, &data); err != nil {
        return nil, fmt.Errorf("failed to unmarshal request: %v", err)
    }

    for _, chunkID := range data.ChunkIDs {
        peerInfo := types.PeerChunkInfo{
            PeerID:    data.PeerID,
            ChunkIDs:  []string{chunkID},
            Available: true,
        }
        s.chunks[chunkID] = append(s.chunks[chunkID], peerInfo)
    }

    return &overlay.Response{
        StatusCode: http.StatusOK,
        Body:      []byte(`{"status":"ok"}`),
    }, nil
}

func (s *Server) handleGetChunkPeers(r *overlay.Request) (*overlay.Response, error) {
    chunkID := r.Path[len("/chunks/peers/"):]
    peers := s.chunks[chunkID]

    data, err := json.Marshal(peers)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal peers: %v", err)
    }

    return &overlay.Response{
        StatusCode: http.StatusOK,
        Body:      data,
    }, nil
}

func (s *Server) handleRegisterKey(r *overlay.Request) (*overlay.Response, error) {
    var data struct {
        FileID    string `json:"file_id"`
        Key       string `json:"key"`
        PublicKey []byte `json:"public_key"`
        ClientID  string `json:"client_id"`
    }
    if err := json.Unmarshal(r.Body, &data); err != nil {
        return nil, fmt.Errorf("failed to unmarshal request: %v", err)
    }

    s.keys[data.FileID] = data.Key
    s.publicKeys[data.ClientID] = data.PublicKey

    return &overlay.Response{
        StatusCode: http.StatusOK,
        Body:      []byte(`{"status":"ok"}`),
    }, nil
}

func (s *Server) handleRequestKey(r *overlay.Request) (*overlay.Response, error) {
    var data struct {
        FileID    string `json:"file_id"`
        ClientID  string `json:"client_id"`
        PublicKey []byte `json:"public_key"`
    }
    if err := json.Unmarshal(r.Body, &data); err != nil {
        return nil, fmt.Errorf("failed to unmarshal request: %v", err)
    }

    key, exists := s.keys[data.FileID]
    if !exists {
        return &overlay.Response{
            StatusCode: http.StatusNotFound,
            Body:      []byte(`{"error":"key not found"}`),
        }, nil
    }

    // In a real implementation, we would encrypt the key with the client's public key here

    resp := struct {
        Key string `json:"key"`
    }{
        Key: key,
    }

    respData, err := json.Marshal(resp)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal response: %v", err)
    }

    return &overlay.Response{
        StatusCode: http.StatusOK,
        Body:      respData,
    }, nil
}

func (s *Server) handlePing(r *overlay.Request) (*overlay.Response, error) {
    return &overlay.Response{
        StatusCode: http.StatusOK,
        Body:      []byte(`{"status":"ok"}`),
    }, nil
}
