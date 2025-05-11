package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/VetheonGames/FileZap/Validator-Server/pkg/server"
)

func main() {
	// Parse command line flags
	addr := flag.String("addr", ":8080", "HTTP service address")
	dataDir := flag.String("data", "./data", "Directory to store registry data")
	flag.Parse()

	// Ensure data directory exists
	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Convert relative path to absolute
	absDataDir, err := filepath.Abs(*dataDir)
	if err != nil {
		log.Fatalf("Failed to resolve data directory path: %v", err)
	}

	// Create and start the server
	srv, err := server.NewServer(absDataDir)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	log.Printf("Starting Validator Server with data directory: %s", absDataDir)
	if err := srv.ListenAndServe(*addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
