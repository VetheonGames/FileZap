package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/VetheonGames/FileZap/Validator-Server/pkg/peer"
	"github.com/VetheonGames/FileZap/Validator-Server/pkg/registry"
	"github.com/gorilla/mux"
)

// Server represents the validator server
type Server struct {
	peerManager *peer.Manager
	registry    *registry.Registry
	router      *mux.Router
}

// NewServer creates a new validator server
func NewServer(dataDir string) (*Server, error) {
	reg, err := registry.NewRegistry(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry: %v", err)
	}

	s := &Server{
		peerManager: peer.NewManager(5 * time.Minute),
		registry:    reg,
		router:      mux.NewRouter(),
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

	// File operations
	s.router.HandleFunc("/file/register", s.handleFileRegister).Methods("POST")
	s.router.HandleFunc("/file/info/{name}", s.handleFileInfo).Methods("GET")
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
