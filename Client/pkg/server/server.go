package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/VetheonGames/FileZap/Client/pkg/keymanager"
	"github.com/VetheonGames/FileZap/Client/pkg/overlay"
	"github.com/VetheonGames/FileZap/Client/pkg/peer"
	"github.com/VetheonGames/FileZap/Client/pkg/quorum"
	"github.com/VetheonGames/FileZap/Client/pkg/registry"
)

// IntegratedServer represents a FileZap node that acts as both client and master node
type IntegratedServer struct {
	ctx           context.Context
	cancel        context.CancelFunc
	peerManager   *peer.Manager
	registry      *registry.Registry
	keyManager    *keymanager.KeyManager
	quorumManager *quorum.QuorumManager
	overlay       overlay.Adapter
	nodeID        string
	isValidator   bool    // Whether this node participates in validation
	balance       float64 // Node's balance for reward system
	mu            sync.RWMutex
}

// NewIntegratedServer creates a new integrated client/master node
func NewIntegratedServer(ctx context.Context, dataDir string, startAsValidator bool) (*IntegratedServer, error) {
	ctx, cancel := context.WithCancel(ctx)

	// Initialize registry
	reg, err := registry.NewRegistry(dataDir)
	if err != nil {
		cancel()
		return nil, err
	}

	server := &IntegratedServer{
		ctx:           ctx,
		cancel:        cancel,
		peerManager:   peer.NewManager(300), // 5 minute timeout
		registry:      reg,
		keyManager:    keymanager.NewKeyManager(3),     // Require 3 shares for key reconstruction
		quorumManager: quorum.NewQuorumManager(300, 3), // 5 minute timeout, require 3 votes
		nodeID:        "",
		isValidator:   startAsValidator,
		balance:       0.0, // Initial balance
	}

	// Initialize overlay network
	overlay, err := overlay.NewAdapter(ctx)
	if err != nil {
		cancel()
		return nil, err
	}
	server.overlay = overlay
	server.nodeID = server.overlay.GetNodeID()

	// If starting as validator, join the validator DHT
	if startAsValidator {
		if err := server.joinValidatorNetwork(); err != nil {
			cancel()
			return nil, err
		}
	}

	server.setupHandlers()
	return server, nil
}

// joinValidatorNetwork joins the DHT network of validators
func (s *IntegratedServer) joinValidatorNetwork() error {
	// Register as validator in quorum manager
	s.quorumManager.RegisterValidator(s.nodeID)

	// Start periodic validation duty checks
	go s.runValidationDuties()

	return nil
}

// runValidationDuties performs periodic validation tasks
func (s *IntegratedServer) runValidationDuties() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkPendingValidations()
			s.maintainReplication()
		}
	}
}

// checkPendingValidations processes any pending validation requests
func (s *IntegratedServer) checkPendingValidations() {
	s.mu.RLock()
	if !s.isValidator {
		s.mu.RUnlock()
		return
	}
	s.mu.RUnlock()

	// Get pending requests from quorum manager
	sessions := s.quorumManager.GetPendingSessions()
	for _, session := range sessions {
		// Verify the request
		if s.verifyKeyRequest(session.FileID, session.ClientID) {
			if err := s.quorumManager.SubmitVote(session.FileID, session.ClientID, s.nodeID, true); err != nil {
				log.Printf("Failed to submit vote for session %s: %v", session.FileID, err)
			}
		}
	}
}

// verifyKeyRequest validates a key request
func (s *IntegratedServer) verifyKeyRequest(fileID string, _ string) bool {
	// Get file info
	file, exists := s.registry.GetFileByID(fileID)
	if !exists {
		return false
	}

	// Verify file exists and meets minimum replication
	peers := s.registry.GetPeersForFile(file.ID)
	if len(peers) < file.ReplicationGoal {
		return false // File is not sufficiently replicated yet
	}

	// Check if client has proper permissions
	// For now, implement basic verification. Can be expanded later.
	return true
}

// maintainReplication ensures proper file replication
func (s *IntegratedServer) maintainReplication() {
	files := s.registry.GetAllFiles()
	for _, file := range files {
		peers := s.registry.GetPeersForFile(file.ID)
		if len(peers) < file.ReplicationGoal {
			s.findNewReplicators(file.ID, file.ReplicationGoal-len(peers))
		}
	}
}

// findNewReplicators locates new peers to replicate a file
func (s *IntegratedServer) findNewReplicators(fileID string, needed int) {
	peers := s.peerManager.GetAllPeers()
	currentPeers := s.registry.GetPeersForFile(fileID)

	for _, peer := range peers {
		if !contains(currentPeers, peer.ID) {
			data := map[string]string{
				"file_id": fileID,
				"action":  "replicate",
			}
			if err := s.overlay.NotifyPeer(peer.ID, "replication_request", data); err != nil {
				log.Printf("Failed to notify peer %s for replication: %v", peer.ID, err)
				continue
			}
			needed--
			if needed == 0 {
				break
			}
		}
	}
}

// contains checks if a string slice contains a value
func contains(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}

// setupHandlers configures all the overlay network handlers
func (s *IntegratedServer) setupHandlers() {
	// Register basic peer management handlers
	s.overlay.HandleFunc("POST", "/peer/register", s.handlePeerRegister)
	s.overlay.HandleFunc("POST", "/peer/status", s.handlePeerStatus)

	// Register file operation handlers
	s.overlay.HandleFunc("POST", "/file/register", s.handleFileRegister)
	s.overlay.HandleFunc("GET", "/file/info/{name}", s.handleFileInfo)

	// Register key management handlers
	s.overlay.HandleFunc("POST", "/key/request", s.handleKeyRequest)
	s.overlay.HandleFunc("POST", "/key/vote", s.handleKeyVote)
	s.overlay.HandleFunc("GET", "/key/share", s.handleKeyShare)

	// Register chunk management handlers
	s.overlay.HandleFunc("POST", "/chunks/register", s.handleChunksRegister)
	s.overlay.HandleFunc("GET", "/chunks/peers/{id}", s.handleGetChunkPeers)
}

// Start begins the integrated server operations
func (s *IntegratedServer) Start() error {
	log.Printf("Starting integrated FileZap node (NodeID: %s)", s.nodeID)

	// Start DHT and peer discovery
	if err := s.overlay.StartDiscovery(); err != nil {
		return err
	}

	// Start periodic peer health checks
	go s.peerManager.StartHealthChecks(s.ctx)

	// Start manifest replication monitoring
	go s.monitorManifestReplication()

	return nil
}

// Stop gracefully shuts down the integrated server
func (s *IntegratedServer) Stop() error {
	s.cancel()
	return s.overlay.Close()
}

// GetNodeID returns the node's peer ID
func (s *IntegratedServer) GetNodeID() string {
	return s.nodeID
}

// Handler implementations
func (s *IntegratedServer) handlePeerRegister(r *overlay.Request) (*overlay.Response, error) {
	var req struct {
		ValidatorID string `json:"validator_id"`
	}
	if err := r.UnmarshalJSON(&req); err != nil {
		return &overlay.Response{
			StatusCode: 400,
			Body:       []byte(`{"error":"Invalid request body"}`),
		}, nil
	}

	// Register with quorum manager
	s.quorumManager.RegisterValidator(req.ValidatorID)

	return &overlay.Response{StatusCode: 200}, nil
}

func (s *IntegratedServer) handlePeerStatus(r *overlay.Request) (*overlay.Response, error) {
	var req struct {
		PeerID        string   `json:"peer_id"`
		AvailableZaps []string `json:"available_zaps"`
	}
	if err := r.UnmarshalJSON(&req); err != nil {
		return &overlay.Response{
			StatusCode: 400,
			Body:       []byte(`{"error":"Invalid request body"}`),
		}, nil
	}

	s.peerManager.UpdatePeer(req.PeerID, "", req.AvailableZaps)
	for _, zapID := range req.AvailableZaps {
		if err := s.registry.AddPeerToFile(zapID, req.PeerID); err != nil {
			log.Printf("Failed to update peer-file association: %v", err)
		}
	}

	return &overlay.Response{StatusCode: 200}, nil
}

func (s *IntegratedServer) handleFileRegister(r *overlay.Request) (*overlay.Response, error) {
	var fileInfo registry.FileInfo
	if err := r.UnmarshalJSON(&fileInfo); err != nil {
		return &overlay.Response{
			StatusCode: 400,
			Body:       []byte(`{"error":"Invalid request body"}`),
		}, nil
	}

	if err := s.registry.RegisterFile(&fileInfo); err != nil {
		return &overlay.Response{
			StatusCode: 500,
			Body:       []byte(`{"error":"Failed to register file"}`),
		}, nil
	}

	availablePeers := s.peerManager.GetAllPeers()

	resp, err := overlay.MarshalJSON(map[string]interface{}{
		"status": "success",
		"peers":  availablePeers,
	})
	if err != nil {
		return nil, err
	}

	return &overlay.Response{
		StatusCode: 200,
		Body:       resp,
	}, nil
}

func (s *IntegratedServer) handleFileInfo(r *overlay.Request) (*overlay.Response, error) {
	name := r.PathParam("name")
	fileInfo, exists := s.registry.GetFileByName(name)
	if !exists {
		return &overlay.Response{
			StatusCode: 404,
			Body:       []byte(`{"error":"File not found"}`),
		}, nil
	}

	peersWithFile := []*peer.Peer{}
	for _, peerID := range fileInfo.PeerIDs {
		if p, exists := s.peerManager.GetPeer(peerID); exists {
			peersWithFile = append(peersWithFile, p)
		}
	}

	resp, err := overlay.MarshalJSON(map[string]interface{}{
		"file_info": fileInfo,
		"peers":     peersWithFile,
	})
	if err != nil {
		return nil, err
	}

	return &overlay.Response{
		StatusCode: 200,
		Body:       resp,
	}, nil
}

func (s *IntegratedServer) handleKeyRequest(r *overlay.Request) (*overlay.Response, error) {
	var req struct {
		FileID    string `json:"file_id"`
		ClientID  string `json:"client_id"`
		PublicKey []byte `json:"public_key"`
	}
	if err := r.UnmarshalJSON(&req); err != nil {
		return &overlay.Response{
			StatusCode: 400,
			Body:       []byte(`{"error":"Invalid request body"}`),
		}, nil
	}

	keyReq := &keymanager.KeyRequest{
		FileID:      req.FileID,
		ClientID:    req.ClientID,
		PublicKey:   req.PublicKey,
		RequestTime: time.Now().Unix(),
	}

	if err := s.keyManager.RegisterKeyRequest(keyReq); err != nil {
		return &overlay.Response{
			StatusCode: 500,
			Body:       []byte(`{"error":"Failed to register key request"}`),
		}, nil
	}

	if err := s.quorumManager.CreateVoteSession(req.FileID, req.ClientID); err != nil {
		return &overlay.Response{
			StatusCode: 500,
			Body:       []byte(`{"error":"Failed to create vote session"}`),
		}, nil
	}

	return &overlay.Response{StatusCode: 202}, nil
}

func (s *IntegratedServer) handleKeyVote(r *overlay.Request) (*overlay.Response, error) {
	var req struct {
		FileID      string `json:"file_id"`
		ClientID    string `json:"client_id"`
		ValidatorID string `json:"validator_id"`
		Approved    bool   `json:"approved"`
	}
	if err := r.UnmarshalJSON(&req); err != nil {
		return &overlay.Response{
			StatusCode: 400,
			Body:       []byte(`{"error":"Invalid request body"}`),
		}, nil
	}

	if err := s.quorumManager.SubmitVote(req.FileID, req.ClientID, req.ValidatorID, req.Approved); err != nil {
		return &overlay.Response{
			StatusCode: 500,
			Body:       []byte(`{"error":"Failed to submit vote"}`),
		}, nil
	}

	approved, err := s.quorumManager.CheckQuorum(req.FileID, req.ClientID)
	if err != nil {
		return &overlay.Response{
			StatusCode: 500,
			Body:       []byte(`{"error":"Failed to check quorum"}`),
		}, nil
	}

	resp, err := overlay.MarshalJSON(map[string]bool{"approved": approved})
	if err != nil {
		return nil, err
	}

	return &overlay.Response{
		StatusCode: 200,
		Body:       resp,
	}, nil
}

func (s *IntegratedServer) handleKeyShare(r *overlay.Request) (*overlay.Response, error) {
	fileID := r.QueryParam("file_id")
	validatorID := r.QueryParam("validator_id")
	if fileID == "" || validatorID == "" {
		return &overlay.Response{
			StatusCode: 400,
			Body:       []byte(`{"error":"Missing file_id or validator_id"}`),
		}, nil
	}

	share, err := s.keyManager.GetKeyShare(fileID, validatorID)
	if err != nil {
		return &overlay.Response{
			StatusCode: 404,
			Body:       []byte(`{"error":"Failed to get key share"}`),
		}, nil
	}

	resp, err := overlay.MarshalJSON(share)
	if err != nil {
		return nil, err
	}

	return &overlay.Response{
		StatusCode: 200,
		Body:       resp,
	}, nil
}

func (s *IntegratedServer) handleChunksRegister(r *overlay.Request) (*overlay.Response, error) {
	var req struct {
		PeerID   string   `json:"peer_id"`
		Address  string   `json:"address"`
		ChunkIDs []string `json:"chunk_ids"`
	}
	if err := r.UnmarshalJSON(&req); err != nil {
		return &overlay.Response{
			StatusCode: 400,
			Body:       []byte(`{"error":"Invalid request body"}`),
		}, nil
	}

	s.registry.RegisterPeerChunks(req.PeerID, req.Address, req.ChunkIDs)

	return &overlay.Response{StatusCode: 200}, nil
}

func (s *IntegratedServer) handleGetChunkPeers(r *overlay.Request) (*overlay.Response, error) {
	chunkID := r.PathParam("id")
	if chunkID == "" {
		return &overlay.Response{
			StatusCode: 400,
			Body:       []byte(`{"error":"Missing chunk ID"}`),
		}, nil
	}

	peers := s.registry.GetPeersForChunk(chunkID)

	resp, err := overlay.MarshalJSON(peers)
	if err != nil {
		return nil, err
	}

	return &overlay.Response{
		StatusCode: 200,
		Body:       resp,
	}, nil
}

func (s *IntegratedServer) monitorManifestReplication() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			files := s.registry.GetAllFiles()
			for _, file := range files {
				peers := s.registry.GetPeersForFile(file.ID)
				if len(peers) < file.ReplicationGoal {
					// Find additional peers to replicate to
					allPeers := s.peerManager.GetAllPeers()
					for _, p := range allPeers {
						if !containsPeer(peers, p.ID) {
							// Notify peer to fetch file
							if err := s.overlay.NotifyPeer(p.ID, "replicate", map[string]string{
								"file_id": file.ID,
							}); err != nil {
								log.Printf("Failed to notify peer %s for replication: %v", p.ID, err)
								continue
							}
						}
						if len(peers)+1 >= file.ReplicationGoal {
							break
						}
					}
				}
			}
		}
	}
}

// Network methods used by UI and operations
func (s *IntegratedServer) GetPeers() []string {
	return s.overlay.Peers()
}

func (s *IntegratedServer) GetPeersWithFile(fileID string) []string {
	return s.registry.GetPeersForFile(fileID)
}

func (s *IntegratedServer) RegisterFile(fileInfo *FileInfo) error {
	// Convert internal FileInfo to registry.FileInfo
	info := &registry.FileInfo{
		ID:              fileInfo.ID,
		Name:            fileInfo.Name,
		ChunkCount:      len(fileInfo.Chunks),
		TotalSize:       fileInfo.TotalSize,
		ZapMetadata:     fileInfo.Metadata,
		ReplicationGoal: 3, // Default replication goal
	}

	// Register with local registry
	if err := s.registry.RegisterFile(info); err != nil {
		return err
	}

	// Notify peers about new file
	data := map[string]string{
		"file_id": info.ID,
		"name":    info.Name,
	}

	// Broadcast to all peers
	peers := s.overlay.Peers()
	for _, peerID := range peers {
		if err := s.overlay.NotifyPeer(peerID, "new_file", data); err != nil {
			log.Printf("Failed to notify peer %s about new file: %v", peerID, err)
		}
	}

	return nil
}

func (s *IntegratedServer) FetchChunks(fileInfo *FileInfo, peerID string) error {
	// Request chunks from peer
	for _, chunk := range fileInfo.Chunks {
		req := &overlay.Request{
			Method: "GET",
			Path:   fmt.Sprintf("/chunks/%s", chunk.ID),
		}

		resp, err := s.overlay.SendMessage(s.ctx, peerID, req)
		if err != nil {
			return fmt.Errorf("failed to fetch chunk %s: %v", chunk.ID, err)
		}

		// Save chunk data
		chunkPath := filepath.Join(fileInfo.ChunkDir, chunk.ID)
		if err := os.WriteFile(chunkPath, resp.Body, 0644); err != nil {
			return fmt.Errorf("failed to save chunk %s: %v", chunk.ID, err)
		}
	}

	return nil
}

// Helper functions
func containsPeer(peers []string, id string) bool {
	for _, p := range peers {
		if p == id {
			return true
		}
	}
	return false
}
