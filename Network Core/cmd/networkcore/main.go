package main

import (
	"flag"
	"log"
	"net/http"
	"time"

    "github.com/VetheonGames/FileZap/NetworkCore/pkg/filemanager"
    "github.com/VetheonGames/FileZap/NetworkCore/pkg/peer"
    "github.com/VetheonGames/FileZap/NetworkCore/pkg/types"
    "github.com/VetheonGames/FileZap/NetworkCore/pkg/validator"
)

// Core represents the main Network Core instance coordinating all components
type Core struct {
	peerManager     *peer.Manager
	validatorClient *validator.Client
	fileManager     *filemanager.Manager
}

// HandleNewZapFile registers a new .zap file and its chunks
func (c *Core) HandleNewZapFile(fileName string, chunks []string) error {
	c.fileManager.RegisterZapFile(fileName, chunks)
	return nil
}

// RequestZapFile requests information about a .zap file from the validator
func (c *Core) RequestZapFile(fileName string) (*types.FileInfo, error) {
	return c.validatorClient.RequestZapFile(fileName)
}

// HandleChunkUpload processes a new chunk upload
func (c *Core) HandleChunkUpload(fileName string, chunkID string, data []byte) error {
	// TODO: Implement chunk storage
	return nil
}

// HandleChunkRequest processes a request for a chunk
func (c *Core) HandleChunkRequest(fileName string, chunkID string) ([]byte, error) {
	// TODO: Implement chunk retrieval
	return nil, nil
}

// Initialize sets up network listeners and starts handling requests
func (c *Core) Initialize() error {
	// Set up HTTP endpoints for peer-to-peer communication
	http.HandleFunc("/peer/chunk/upload", func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement chunk upload endpoint
	})

	http.HandleFunc("/peer/chunk/request", func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement chunk request endpoint
	})

	return http.ListenAndServe(":3000", nil)
}

func main() {
	log.Println("Starting Network Core...")

	// Parse command line flags
	validatorAddr := flag.String("validator", "localhost:8080", "Validator server address")
	peerTimeout := flag.Duration("peer-timeout", 1*time.Hour, "Time after which inactive peers are removed")
	flag.Parse()

	// Initialize components
	peerManager := peer.NewManager(*peerTimeout)
	validatorClient := validator.NewClient(*validatorAddr)
	fileManager := filemanager.NewManager()

	// Create core instance
	core := &Core{
		peerManager:     peerManager,
		validatorClient: validatorClient,
		fileManager:     fileManager,
	}

	// Start background tasks
	go validatorClient.MaintainConnection()

	// Start background task to update validator about our available files
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		for range ticker.C {
			if validatorClient.IsConnected() {
				availableZaps := fileManager.GetAvailableZaps()
				validatorClient.UpdateAvailableZaps(availableZaps)
			}
		}
	}()

	log.Printf("Network Core running, connected to validator at %s", *validatorAddr)

	// Initialize HTTP server and start handling requests
	if err := core.Initialize(); err != nil {
		log.Fatalf("Failed to initialize network core: %v", err)
	}
}
