cpackage server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

"github.com/VetheonGames/FileZap/Validator-Server/pkg/crypto"
"github.com/VetheonGames/FileZap/Validator-Server/pkg/keymanager"
"github.com/VetheonGames/FileZap/Validator-Server/pkg/peer"
"github.com/VetheonGames/FileZap/Validator-Server/pkg/quorum"
"github.com/VetheonGames/FileZap/Validator-Server/pkg/registry"
"github.com/gorilla/mux"
)

// Server represents the validator server
type Server struct {
peerManager   *peer.Manager
registry      *registry.Registry
keyManager    *keymanager.KeyManager
quorumManager *quorum.QuorumManager
rewardManager *crypto.RewardManager
router        *mux.Router
}

// NewServer creates a new validator server
func NewServer(dataDir string) (*Server, error) {
reg, err := registry.NewRegistry(dataDir)
if err != nil {
return nil, fmt.Errorf("failed to create registry: %v", err)
}

s := &Server{
peerManager:   peer.NewManager(5 * time.Minute),
registry:      reg,
keyManager:    keymanager.NewKeyManager(3), // Require 3 shares for key reconstruction
quorumManager: quorum.NewQuorumManager(300, 3), // 5 minute timeout, require 3 votes
rewardManager: crypto.NewRewardManager(),
router:        mux.NewRouter(),
}

// Create system account for rewards
if err := s.rewardManager.CreateAccount("system", 1000000); err != nil {
return nil, fmt.Errorf("failed to create system account: %v", err)
}

s.setupRoutes()
return s, nil
}

// setupRoutes configures the HTTP routes
func (s *Server) setupRoutes() {
    // Basic health check
    s.router.HandleFunc("/ping", s.handlePing).Methods("GET")

    // Peer management
    s.router.HandleFunc("/peer/status", s.handlePeerStatus).Methods("POST")
    s.router.HandleFunc("/peer/register", s.handlePeerRegister).Methods("POST")

    // File operations
    s.router.HandleFunc("/file/register", s.handleFileRegister).Methods("POST")
    s.router.HandleFunc("/file/info/{name}", s.handleFileInfo).Methods("GET")

    // Key management
    s.router.HandleFunc("/key/request", s.handleKeyRequest).Methods("POST")
    s.router.HandleFunc("/key/vote", s.handleKeyVote).Methods("POST")
    s.router.HandleFunc("/key/share", s.handleKeyShare).Methods("GET")

    // Account management
    s.router.HandleFunc("/account/create", s.handleAccountCreate).Methods("POST")
    s.router.HandleFunc("/account/balance", s.handleAccountBalance).Methods("GET")

    // Chunk management
    s.router.HandleFunc("/chunks/register", s.handleChunksRegister).Methods("POST")
    s.router.HandleFunc("/chunks/peers/{id}", s.handleGetChunkPeers).Methods("GET")
}

// handleChunksRegister registers available chunks from a peer
func (s *Server) handleChunksRegister(w http.ResponseWriter, r *http.Request) {
    var req struct {
        PeerID   string   `json:"peer_id"`
        Address  string   `json:"address"`
        ChunkIDs []string `json:"chunk_ids"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Store chunk availability info
    s.registry.RegisterPeerChunks(req.PeerID, req.Address, req.ChunkIDs)

    w.WriteHeader(http.StatusOK)
}

// handleGetChunkPeers returns a list of peers that have a specific chunk
func (s *Server) handleGetChunkPeers(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    chunkID := vars["id"]
    if chunkID == "" {
        http.Error(w, "Missing chunk ID", http.StatusBadRequest)
        return
    }

    // Get peers that have this chunk
    peers := s.registry.GetPeersForChunk(chunkID)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(peers)
}

// handlePeerRegister handles validator registration
func (s *Server) handlePeerRegister(w http.ResponseWriter, r *http.Request) {
var req struct {
ValidatorID string `json:"validator_id"`
}

if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
http.Error(w, "Invalid request body", http.StatusBadRequest)
return
}

// Register with quorum manager
s.quorumManager.RegisterValidator(req.ValidatorID)

// Create reward account for validator
if err := s.rewardManager.CreateAccount(req.ValidatorID, 0); err != nil {
if err.Error() != "account already exists" {
http.Error(w, fmt.Sprintf("Failed to create account: %v", err), http.StatusInternalServerError)
return
}
}

w.WriteHeader(http.StatusOK)
}

// handleKeyRequest handles client requests for decryption keys
func (s *Server) handleKeyRequest(w http.ResponseWriter, r *http.Request) {
var req struct {
FileID    string `json:"file_id"`
ClientID  string `json:"client_id"`
PublicKey []byte `json:"public_key"`
}

if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
http.Error(w, "Invalid request body", http.StatusBadRequest)
return
}

// Charge for key request
if err := s.rewardManager.ChargeForOperation(req.ClientID, "download"); err != nil {
http.Error(w, fmt.Sprintf("Failed to process payment: %v", err), http.StatusPaymentRequired)
return
}

// Create key request
keyReq := &keymanager.KeyRequest{
FileID:      req.FileID,
ClientID:    req.ClientID,
PublicKey:   req.PublicKey,
RequestTime: time.Now().Unix(),
}

if err := s.keyManager.RegisterKeyRequest(keyReq); err != nil {
http.Error(w, fmt.Sprintf("Failed to register key request: %v", err), http.StatusInternalServerError)
return
}

// Start vote session
if err := s.quorumManager.CreateVoteSession(req.FileID, req.ClientID); err != nil {
http.Error(w, fmt.Sprintf("Failed to create vote session: %v", err), http.StatusInternalServerError)
return
}

w.WriteHeader(http.StatusAccepted)
}

// handleKeyVote handles validator votes on key requests
func (s *Server) handleKeyVote(w http.ResponseWriter, r *http.Request) {
var req struct {
FileID      string `json:"file_id"`
ClientID    string `json:"client_id"`
ValidatorID string `json:"validator_id"`
Approved    bool   `json:"approved"`
}

if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
http.Error(w, "Invalid request body", http.StatusBadRequest)
return
}

// Submit vote
if err := s.quorumManager.SubmitVote(req.FileID, req.ClientID, req.ValidatorID, req.Approved); err != nil {
http.Error(w, fmt.Sprintf("Failed to submit vote: %v", err), http.StatusInternalServerError)
return
}

// Check if we have quorum
approved, err := s.quorumManager.CheckQuorum(req.FileID, req.ClientID)
if err != nil {
http.Error(w, fmt.Sprintf("Failed to check quorum: %v", err), http.StatusInternalServerError)
return
}

if approved {
// Reward validator for participation
if err := s.rewardManager.RewardForValidation(req.ValidatorID); err != nil {
log.Printf("Failed to process validator reward: %v", err)
}
}

w.WriteHeader(http.StatusOK)
json.NewEncoder(w).Encode(map[string]bool{"approved": approved})
}

// handleKeyShare returns a validator's key share for a file
func (s *Server) handleKeyShare(w http.ResponseWriter, r *http.Request) {
fileID := r.URL.Query().Get("file_id")
validatorID := r.URL.Query().Get("validator_id")

share, err := s.keyManager.GetKeyShare(fileID, validatorID)
if err != nil {
http.Error(w, fmt.Sprintf("Failed to get key share: %v", err), http.StatusNotFound)
return
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(share)
}

// handleAccountCreate creates a new account
func (s *Server) handleAccountCreate(w http.ResponseWriter, r *http.Request) {
var req struct {
ID      string  `json:"id"`
Balance float64 `json:"balance"`
}

if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
http.Error(w, "Invalid request body", http.StatusBadRequest)
return
}

if err := s.rewardManager.CreateAccount(req.ID, req.Balance); err != nil {
http.Error(w, fmt.Sprintf("Failed to create account: %v", err), http.StatusInternalServerError)
return
}

w.WriteHeader(http.StatusCreated)
}

// handleAccountBalance returns an account's balance
func (s *Server) handleAccountBalance(w http.ResponseWriter, r *http.Request) {
id := r.URL.Query().Get("id")
if id == "" {
http.Error(w, "Missing account ID", http.StatusBadRequest)
return
}

balance, err := s.rewardManager.GetBalance(id)
if err != nil {
http.Error(w, fmt.Sprintf("Failed to get balance: %v", err), http.StatusNotFound)
return
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(map[string]float64{"balance": balance})
}

// ListenAndServe starts the server
func (s *Server) ListenAndServe(addr string) error {
	log.Printf("Starting validator server on %s", addr)
	return http.ListenAndServe(addr, s.router)
}

func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

type peerStatusRequest struct {
	PeerID        string   `json:"peer_id"`
	AvailableZaps []string `json:"available_zaps"`
}

func (s *Server) handlePeerStatus(w http.ResponseWriter, r *http.Request) {
	var req peerStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update peer status
	s.peerManager.UpdatePeer(req.PeerID, r.RemoteAddr, req.AvailableZaps)

	// Update file registry
	for _, zapID := range req.AvailableZaps {
		if err := s.registry.AddPeerToFile(zapID, req.PeerID); err != nil {
			log.Printf("Failed to update peer-file association: %v", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleFileRegister(w http.ResponseWriter, r *http.Request) {
	var fileInfo registry.FileInfo
	if err := json.NewDecoder(r.Body).Decode(&fileInfo); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Register the file
	if err := s.registry.RegisterFile(&fileInfo); err != nil {
		http.Error(w, fmt.Sprintf("Failed to register file: %v", err), http.StatusInternalServerError)
		return
	}

	// Get available peers for chunk distribution
	peers := s.peerManager.GetAllPeers()
	availablePeers := make([]peer.Peer, 0)
	for _, p := range peers {
		availablePeers = append(availablePeers, *p)
	}

	// Return list of available peers
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"peers":  availablePeers,
	})
}

func (s *Server) handleFileInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	// Get file info
	fileInfo, exists := s.registry.GetFileByName(name)
	if !exists {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Get peers that have this file
	peersWithFile := []*peer.Peer{}
	for _, peerID := range fileInfo.PeerIDs {
		if p, exists := s.peerManager.GetPeer(peerID); exists {
			peersWithFile = append(peersWithFile, p)
		}
	}

	// Return file info and available peers
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"file_info": fileInfo,
		"peers":     peersWithFile,
	})
}
