package server

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "strings"
    "time"

	"github.com/VetheonGames/FileZap/Validator-Server/pkg/crypto"
	"github.com/VetheonGames/FileZap/Validator-Server/pkg/keymanager"
	"github.com/VetheonGames/FileZap/Validator-Server/pkg/overlay"
	"github.com/VetheonGames/FileZap/Validator-Server/pkg/peer"
	"github.com/VetheonGames/FileZap/Validator-Server/pkg/quorum"
	"github.com/VetheonGames/FileZap/Validator-Server/pkg/registry"
	"github.com/gorilla/mux"
)

// Server represents the validator server with support for both HTTP and overlay networking
type Server struct {
	peerManager   *peer.Manager
	registry      *registry.Registry
	keyManager    *keymanager.KeyManager
	quorumManager *quorum.QuorumManager
	rewardManager *crypto.RewardManager
	router        *mux.Router
	overlay       *overlay.Adapter
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewServer creates a new validator server with both HTTP and overlay networking support
func NewServer(ctx context.Context, dataDir string) (*Server, error) {
	ctx, cancel := context.WithCancel(ctx)

	reg, err := registry.NewRegistry(dataDir)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create registry: %v", err)
	}

	overlay, err := overlay.NewAdapter(ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create overlay adapter: %v", err)
	}

	s := &Server{
		peerManager:   peer.NewManager(5 * time.Minute),
		registry:      reg,
		keyManager:    keymanager.NewKeyManager(3),     // Require 3 shares for key reconstruction
		quorumManager: quorum.NewQuorumManager(300, 3), // 5 minute timeout, require 3 votes
		rewardManager: crypto.NewRewardManager(),
		router:        mux.NewRouter(),
		overlay:       overlay,
		ctx:           ctx,
		cancel:        cancel,
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

// setupOverlayHandlers configures the overlay network handlers
func (s *Server) setupOverlayHandlers() {
    // Basic health check
    s.overlay.HandleFunc("GET", "/ping", func(r *overlay.Request) (*overlay.Response, error) {
        return &overlay.Response{StatusCode: http.StatusOK}, nil
    })

    // Peer management
    s.overlay.HandleFunc("POST", "/peer/register", func(r *overlay.Request) (*overlay.Response, error) {
        var req struct {
            ValidatorID string `json:"validator_id"`
        }
        if err := json.Unmarshal(r.Body, &req); err != nil {
            return &overlay.Response{
                StatusCode: http.StatusBadRequest,
                Body:      []byte(`{"error":"Invalid request body"}`),
            }, nil
        }

        s.quorumManager.RegisterValidator(req.ValidatorID)
        if err := s.rewardManager.CreateAccount(req.ValidatorID, 0); err != nil && err.Error() != "account already exists" {
            return &overlay.Response{
                StatusCode: http.StatusInternalServerError,
                Body:      []byte(fmt.Sprintf(`{"error":"Failed to create account: %v"}`, err)),
            }, nil
        }

        return &overlay.Response{StatusCode: http.StatusOK}, nil
    })

    s.overlay.HandleFunc("POST", "/peer/status", func(r *overlay.Request) (*overlay.Response, error) {
        var req struct {
            PeerID        string   `json:"peer_id"`
            AvailableZaps []string `json:"available_zaps"`
        }
        if err := json.Unmarshal(r.Body, &req); err != nil {
            return &overlay.Response{
                StatusCode: http.StatusBadRequest,
                Body:      []byte(`{"error":"Invalid request body"}`),
            }, nil
        }

        s.peerManager.UpdatePeer(req.PeerID, "", req.AvailableZaps) // Empty address since it's overlay network
        for _, zapID := range req.AvailableZaps {
            if err := s.registry.AddPeerToFile(zapID, req.PeerID); err != nil {
                log.Printf("Failed to update peer-file association: %v", err)
            }
        }

        return &overlay.Response{StatusCode: http.StatusOK}, nil
    })

    // Account management
    s.overlay.HandleFunc("POST", "/account/create", func(r *overlay.Request) (*overlay.Response, error) {
        var req struct {
            ID      string  `json:"id"`
            Balance float64 `json:"balance"`
        }
        if err := json.Unmarshal(r.Body, &req); err != nil {
            return &overlay.Response{
                StatusCode: http.StatusBadRequest,
                Body:      []byte(`{"error":"Invalid request body"}`),
            }, nil
        }

        if err := s.rewardManager.CreateAccount(req.ID, req.Balance); err != nil {
            return &overlay.Response{
                StatusCode: http.StatusInternalServerError,
                Body:      []byte(fmt.Sprintf(`{"error":"Failed to create account: %v"}`, err)),
            }, nil
        }

        return &overlay.Response{StatusCode: http.StatusCreated}, nil
    })

    s.overlay.HandleFunc("GET", "/account/balance", func(r *overlay.Request) (*overlay.Response, error) {
        id := r.Path[len("/account/balance?id="):]
        if id == "" {
            return &overlay.Response{
                StatusCode: http.StatusBadRequest,
                Body:      []byte(`{"error":"Missing account ID"}`),
            }, nil
        }

        balance, err := s.rewardManager.GetBalance(id)
        if err != nil {
            return &overlay.Response{
                StatusCode: http.StatusNotFound,
                Body:      []byte(fmt.Sprintf(`{"error":"Failed to get balance: %v"}`, err)),
            }, nil
        }

        resp, err := json.Marshal(map[string]float64{"balance": balance})
        if err != nil {
            return nil, err
        }

        return &overlay.Response{
            StatusCode: http.StatusOK,
            Body:      resp,
        }, nil
    })

    // File operations
	s.overlay.HandleFunc("POST", "/file/register", func(r *overlay.Request) (*overlay.Response, error) {
		var fileInfo registry.FileInfo
		if err := json.Unmarshal(r.Body, &fileInfo); err != nil {
			return &overlay.Response{
				StatusCode: http.StatusBadRequest,
				Body:       []byte(`{"error":"Invalid request body"}`),
			}, nil
		}

		if err := s.registry.RegisterFile(&fileInfo); err != nil {
			return &overlay.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       []byte(fmt.Sprintf(`{"error":"Failed to register file: %v"}`, err)),
			}, nil
		}

		peers := s.peerManager.GetAllPeers()
		availablePeers := make([]peer.Peer, 0)
		for _, p := range peers {
			availablePeers = append(availablePeers, *p)
		}

		resp, err := json.Marshal(map[string]interface{}{
			"status": "success",
			"peers":  availablePeers,
		})
		if err != nil {
			return nil, err
		}

		return &overlay.Response{
			StatusCode: http.StatusOK,
			Body:       resp,
		}, nil
	})

	// Key management
	s.overlay.HandleFunc("POST", "/key/request", func(r *overlay.Request) (*overlay.Response, error) {
		var req struct {
			FileID    string `json:"file_id"`
			ClientID  string `json:"client_id"`
			PublicKey []byte `json:"public_key"`
		}
		if err := json.Unmarshal(r.Body, &req); err != nil {
			return &overlay.Response{
				StatusCode: http.StatusBadRequest,
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
				StatusCode: http.StatusInternalServerError,
				Body:       []byte(fmt.Sprintf(`{"error":"Failed to register key request: %v"}`, err)),
			}, nil
		}

		if err := s.quorumManager.CreateVoteSession(req.FileID, req.ClientID); err != nil {
			return &overlay.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       []byte(fmt.Sprintf(`{"error":"Failed to create vote session: %v"}`, err)),
			}, nil
		}

		return &overlay.Response{
			StatusCode: http.StatusAccepted,
		}, nil
	})

	// Chunk operations
	s.overlay.HandleFunc("POST", "/chunks/register", func(r *overlay.Request) (*overlay.Response, error) {
		var req struct {
			PeerID   string   `json:"peer_id"`
			Address  string   `json:"address"`
			ChunkIDs []string `json:"chunk_ids"`
		}
		if err := json.Unmarshal(r.Body, &req); err != nil {
			return &overlay.Response{
				StatusCode: http.StatusBadRequest,
				Body:       []byte(`{"error":"Invalid request body"}`),
			}, nil
		}

		s.registry.RegisterPeerChunks(req.PeerID, req.Address, req.ChunkIDs)

		return &overlay.Response{
			StatusCode: http.StatusOK,
		}, nil
	})

	s.overlay.HandleFunc("GET", "/chunks/peers/{id}", func(r *overlay.Request) (*overlay.Response, error) {
		chunkID := r.Path[len("/chunks/peers/"):]
		if chunkID == "" {
			return &overlay.Response{
				StatusCode: http.StatusBadRequest,
				Body:       []byte(`{"error":"Missing chunk ID"}`),
			}, nil
		}

		peers := s.registry.GetPeersForChunk(chunkID)

		resp, err := json.Marshal(peers)
		if err != nil {
			return nil, err
		}

		return &overlay.Response{
			StatusCode: http.StatusOK,
			Body:       resp,
		}, nil
	})
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

// Close gracefully shuts down the server
func (s *Server) Close() error {
    s.cancel()
    if err := s.overlay.Close(); err != nil {
        return fmt.Errorf("failed to close overlay adapter: %v", err)
    }
    return nil
}

// ListenAndServe starts both HTTP and overlay networking servers
func (s *Server) ListenAndServe(addr string) error {
    log.Printf("Starting validator server on %s (HTTP) and overlay network (NodeID: %s)", addr, s.overlay.GetNodeID())

    // Set up overlay network handlers
    s.setupOverlayHandlers()

    // Add remaining key management handlers
    s.overlay.HandleFunc("POST", "/key/vote", func(r *overlay.Request) (*overlay.Response, error) {
        var req struct {
            FileID      string `json:"file_id"`
            ClientID    string `json:"client_id"`
            ValidatorID string `json:"validator_id"`
            Approved    bool   `json:"approved"`
        }
        if err := json.Unmarshal(r.Body, &req); err != nil {
            return &overlay.Response{
                StatusCode: http.StatusBadRequest,
                Body:      []byte(`{"error":"Invalid request body"}`),
            }, nil
        }

        if err := s.quorumManager.SubmitVote(req.FileID, req.ClientID, req.ValidatorID, req.Approved); err != nil {
            return &overlay.Response{
                StatusCode: http.StatusInternalServerError,
                Body:      []byte(fmt.Sprintf(`{"error":"Failed to submit vote: %v"}`, err)),
            }, nil
        }

        approved, err := s.quorumManager.CheckQuorum(req.FileID, req.ClientID)
        if err != nil {
            return &overlay.Response{
                StatusCode: http.StatusInternalServerError,
                Body:      []byte(fmt.Sprintf(`{"error":"Failed to check quorum: %v"}`, err)),
            }, nil
        }

        if approved {
            if err := s.rewardManager.RewardForValidation(req.ValidatorID); err != nil {
                log.Printf("Failed to process validator reward: %v", err)
            }
        }

        resp, err := json.Marshal(map[string]bool{"approved": approved})
        if err != nil {
            return nil, err
        }

        return &overlay.Response{
            StatusCode: http.StatusOK,
            Body:      resp,
        }, nil
    })

    s.overlay.HandleFunc("GET", "/key/share", func(r *overlay.Request) (*overlay.Response, error) {
        // Extract fileID and validatorID from query params
        fileID := r.Path[len("/key/share?file_id="):]
        validatorID := ""
        if idx := strings.Index(fileID, "&validator_id="); idx != -1 {
            validatorID = fileID[idx+len("&validator_id="):]
            fileID = fileID[:idx]
        }

        if fileID == "" || validatorID == "" {
            return &overlay.Response{
                StatusCode: http.StatusBadRequest,
                Body:      []byte(`{"error":"Missing file_id or validator_id"}`),
            }, nil
        }

        share, err := s.keyManager.GetKeyShare(fileID, validatorID)
        if err != nil {
            return &overlay.Response{
                StatusCode: http.StatusNotFound,
                Body:      []byte(fmt.Sprintf(`{"error":"Failed to get key share: %v"}`, err)),
            }, nil
        }

        resp, err := json.Marshal(share)
        if err != nil {
            return nil, err
        }

        return &overlay.Response{
            StatusCode: http.StatusOK,
            Body:      resp,
        }, nil
    })

    s.overlay.HandleFunc("GET", "/file/info/{name}", func(r *overlay.Request) (*overlay.Response, error) {
        name := r.Path[len("/file/info/"):]
        fileInfo, exists := s.registry.GetFileByName(name)
        if !exists {
            return &overlay.Response{
                StatusCode: http.StatusNotFound,
                Body:      []byte(`{"error":"File not found"}`),
            }, nil
        }

        peersWithFile := []*peer.Peer{}
        for _, peerID := range fileInfo.PeerIDs {
            if p, exists := s.peerManager.GetPeer(peerID); exists {
                peersWithFile = append(peersWithFile, p)
            }
        }

        resp, err := json.Marshal(map[string]interface{}{
            "file_info": fileInfo,
            "peers":     peersWithFile,
        })
        if err != nil {
            return nil, err
        }

        return &overlay.Response{
            StatusCode: http.StatusOK,
            Body:      resp,
        }, nil
    })

    // Create error channel for server errors
    errChan := make(chan error, 2)

    // Create server with shutdown timeout
    srv := &http.Server{
        Addr:    addr,
        Handler: s.router,
    }

    // Start HTTP server
    go func() {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            errChan <- fmt.Errorf("HTTP server error: %v", err)
        }
    }()

    // Wait for shutdown signal or error
    select {
    case err := <-errChan:
        s.Close()
        return err
    case <-s.ctx.Done():
        log.Println("Shutting down validator server...")
        if err := srv.Shutdown(context.Background()); err != nil {
            log.Printf("HTTP server shutdown error: %v", err)
        }
        return s.Close()
    }
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
