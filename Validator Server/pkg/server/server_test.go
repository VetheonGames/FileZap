package server

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "os"
    "path/filepath"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func setupTestServer(t *testing.T) (*Server, string, func()) {
    // Create temporary directory for test data
    tempDir, err := os.MkdirTemp("", "validator_test_*")
    require.NoError(t, err)

    // Create server
    ctx := context.Background()
    srv, err := NewServer(ctx, tempDir)
    require.NoError(t, err)

    // Return cleanup function
    cleanup := func() {
        srv.Close()
        os.RemoveAll(tempDir)
    }

    return srv, tempDir, cleanup
}

func TestPingEndpoint(t *testing.T) {
    srv, _, cleanup := setupTestServer(t)
    defer cleanup()

    req := httptest.NewRequest("GET", "/ping", nil)
    resp := httptest.NewRecorder()

    srv.handlePing(resp, req)
    assert.Equal(t, http.StatusOK, resp.Code)
}

func TestPeerRegistration(t *testing.T) {
    srv, _, cleanup := setupTestServer(t)
    defer cleanup()

    // Test valid registration
    t.Run("valid registration", func(t *testing.T) {
        reqBody := map[string]string{
            "validator_id": "test_validator_1",
        }
        body, err := json.Marshal(reqBody)
        require.NoError(t, err)

        req := httptest.NewRequest("POST", "/peer/register", bytes.NewBuffer(body))
        resp := httptest.NewRecorder()

        srv.handlePeerRegister(resp, req)
        assert.Equal(t, http.StatusOK, resp.Code)

        // Verify account was created
        balance, err := srv.rewardManager.GetBalance("test_validator_1")
        assert.NoError(t, err)
        assert.Equal(t, float64(0), balance)
    })

    // Test invalid request body
    t.Run("invalid request body", func(t *testing.T) {
        req := httptest.NewRequest("POST", "/peer/register", bytes.NewBuffer([]byte("invalid json")))
        resp := httptest.NewRecorder()

        srv.handlePeerRegister(resp, req)
        assert.Equal(t, http.StatusBadRequest, resp.Code)
    })
}

func TestPeerStatus(t *testing.T) {
    srv, _, cleanup := setupTestServer(t)
    defer cleanup()

    // Test valid status update
    t.Run("valid status update", func(t *testing.T) {
        reqBody := peerStatusRequest{
            PeerID:        "test_peer_1",
            AvailableZaps: []string{"zap1", "zap2"},
        }
        body, err := json.Marshal(reqBody)
        require.NoError(t, err)

        req := httptest.NewRequest("POST", "/peer/status", bytes.NewBuffer(body))
        resp := httptest.NewRecorder()

        srv.handlePeerStatus(resp, req)
        assert.Equal(t, http.StatusOK, resp.Code)

        // Verify peer was added
        peer, exists := srv.peerManager.GetPeer("test_peer_1")
        assert.True(t, exists)
        assert.Equal(t, reqBody.AvailableZaps, peer.AvailableZaps)
    })

    // Test invalid request body
    t.Run("invalid request body", func(t *testing.T) {
        req := httptest.NewRequest("POST", "/peer/status", bytes.NewBuffer([]byte("invalid json")))
        resp := httptest.NewRecorder()

        srv.handlePeerStatus(resp, req)
        assert.Equal(t, http.StatusBadRequest, resp.Code)
    })
}

func TestFileOperations(t *testing.T) {
    srv, _, cleanup := setupTestServer(t)
    defer cleanup()

    // Add test peer
    srv.peerManager.UpdatePeer("test_peer_1", "localhost:8081", []string{})

    // Test file registration
    t.Run("file registration", func(t *testing.T) {
        fileInfo := struct {
            Name      string   `json:"name"`
            Size      int64    `json:"size"`
            Chunks    int      `json:"chunks"`
            PeerIDs   []string `json:"peer_ids"`
            Hash      string   `json:"hash"`
            Timestamp int64    `json:"timestamp"`
        }{
            Name:      "test.txt",
            Size:      1024,
            Chunks:    2,
            PeerIDs:   []string{"test_peer_1"},
            Hash:      "testhash",
            Timestamp: time.Now().Unix(),
        }

        body, err := json.Marshal(fileInfo)
        require.NoError(t, err)

        req := httptest.NewRequest("POST", "/file/register", bytes.NewBuffer(body))
        resp := httptest.NewRecorder()

        srv.handleFileRegister(resp, req)
        assert.Equal(t, http.StatusOK, resp.Code)

        // Verify response includes peer list
        var response map[string]interface{}
        err = json.Unmarshal(resp.Body.Bytes(), &response)
        require.NoError(t, err)
        assert.Equal(t, "success", response["status"])
        assert.NotNil(t, response["peers"])
    })

    // Test file info retrieval
    t.Run("file info retrieval", func(t *testing.T) {
        req := httptest.NewRequest("GET", "/file/info/test.txt", nil)
        resp := httptest.NewRecorder()

        srv.handleFileInfo(resp, req)
        assert.Equal(t, http.StatusOK, resp.Code)

        var response map[string]interface{}
        err := json.Unmarshal(resp.Body.Bytes(), &response)
        require.NoError(t, err)
        assert.NotNil(t, response["file_info"])
        assert.NotNil(t, response["peers"])
    })
}

func TestAccountOperations(t *testing.T) {
    srv, _, cleanup := setupTestServer(t)
    defer cleanup()

    // Test account creation
    t.Run("account creation", func(t *testing.T) {
        reqBody := map[string]interface{}{
            "id":      "test_user",
            "balance": 100.0,
        }
        body, err := json.Marshal(reqBody)
        require.NoError(t, err)

        req := httptest.NewRequest("POST", "/account/create", bytes.NewBuffer(body))
        resp := httptest.NewRecorder()

        srv.handleAccountCreate(resp, req)
        assert.Equal(t, http.StatusCreated, resp.Code)

        // Verify balance
        balance, err := srv.rewardManager.GetBalance("test_user")
        assert.NoError(t, err)
        assert.Equal(t, 100.0, balance)
    })

    // Test balance retrieval
    t.Run("balance retrieval", func(t *testing.T) {
        req := httptest.NewRequest("GET", "/account/balance?id=test_user", nil)
        resp := httptest.NewRecorder()

        srv.handleAccountBalance(resp, req)
        assert.Equal(t, http.StatusOK, resp.Code)

        var response map[string]float64
        err := json.Unmarshal(resp.Body.Bytes(), &response)
        require.NoError(t, err)
        assert.Equal(t, 100.0, response["balance"])
    })
}

func TestKeyOperations(t *testing.T) {
    srv, _, cleanup := setupTestServer(t)
    defer cleanup()

    // Create test account with sufficient balance
    err := srv.rewardManager.CreateAccount("test_client", 1000.0)
    require.NoError(t, err)

    // Test key request
    t.Run("key request", func(t *testing.T) {
        reqBody := map[string]interface{}{
            "file_id":    "test_file",
            "client_id":  "test_client",
            "public_key": []byte("test_key"),
        }
        body, err := json.Marshal(reqBody)
        require.NoError(t, err)

        req := httptest.NewRequest("POST", "/key/request", bytes.NewBuffer(body))
        resp := httptest.NewRecorder()

        srv.handleKeyRequest(resp, req)
        assert.Equal(t, http.StatusAccepted, resp.Code)
    })

    // Test key vote
    t.Run("key vote", func(t *testing.T) {
        reqBody := map[string]interface{}{
            "file_id":      "test_file",
            "client_id":    "test_client",
            "validator_id": "test_validator",
            "approved":     true,
        }
        body, err := json.Marshal(reqBody)
        require.NoError(t, err)

        req := httptest.NewRequest("POST", "/key/vote", bytes.NewBuffer(body))
        resp := httptest.NewRecorder()

        srv.handleKeyVote(resp, req)
        assert.Equal(t, http.StatusOK, resp.Code)

        var response map[string]bool
        err = json.Unmarshal(resp.Body.Bytes(), &response)
        require.NoError(t, err)
        assert.NotNil(t, response["approved"])
    })
}

func TestChunkOperations(t *testing.T) {
    srv, _, cleanup := setupTestServer(t)
    defer cleanup()

    // Test chunk registration
    t.Run("chunk registration", func(t *testing.T) {
        reqBody := map[string]interface{}{
            "peer_id":   "test_peer",
            "address":   "localhost:8081",
            "chunk_ids": []string{"chunk1", "chunk2"},
        }
        body, err := json.Marshal(reqBody)
        require.NoError(t, err)

        req := httptest.NewRequest("POST", "/chunks/register", bytes.NewBuffer(body))
        resp := httptest.NewRecorder()

        srv.handleChunksRegister(resp, req)
        assert.Equal(t, http.StatusOK, resp.Code)

        // Verify chunks were registered
        peers := srv.registry.GetPeersForChunk("chunk1")
        assert.Contains(t, peers, "test_peer")
    })

    // Test get chunk peers
    t.Run("get chunk peers", func(t *testing.T) {
        req := httptest.NewRequest("GET", "/chunks/peers/chunk1", nil)
        resp := httptest.NewRecorder()

        srv.handleGetChunkPeers(resp, req)
        assert.Equal(t, http.StatusOK, resp.Code)

        var peers []string
        err := json.Unmarshal(resp.Body.Bytes(), &peers)
        require.NoError(t, err)
        assert.Contains(t, peers, "test_peer")
    })
}
