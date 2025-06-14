package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type BuildTarget struct {
	name string
	dir  string
}

func main() {
	targets := []BuildTarget{
		{"Client", "./Client"},
		{"Divider", "./Divider"},
		{"NetworkCore", "./Network Core"},
		{"Reconstructor", "./Reconstructor"},
	}

	var wg sync.WaitGroup
	errors := make(chan error, len(targets))

	// Create bin directory if it doesn't exist
	if err := os.MkdirAll("bin", 0755); err != nil {
		log.Fatalf("Failed to create bin directory: %v", err)
	}

	for _, target := range targets {
		wg.Add(1)
		go func(t BuildTarget) {
			defer wg.Done()

			fmt.Printf("Building %s...\n", t.name)

			// Determine the command directory name based on the target
			cmdDir := "cmd"
			if t.name == "NetworkCore" {
				cmdDir = filepath.Join(cmdDir, "networkcore")
			} else {
				cmdDir = filepath.Join(cmdDir, strings.ToLower(t.name))
			}

			cmd := exec.Command("go", "build", "-o", filepath.Join("bin", t.name+".exe"), cmdDir)
			cmd.Dir = t.dir
			if output, err := cmd.CombinedOutput(); err != nil {
				errors <- fmt.Errorf("Failed to build %s:\n%s\n%v", t.name, string(output), err)
				return
			}

			fmt.Printf("Successfully built %s\n", t.name)
		}(target)
	}

	wg.Wait()
	close(errors)

	// Check for any build errors
	failed := false
	for err := range errors {
		failed = true
		fmt.Fprintln(os.Stderr, err)
	}

	if failed {
		os.Exit(1)
	}

	fmt.Println("\nAll components built successfully! Binaries are in the bin directory.")
}
