//go:build windows

package main

import (
	"os"
	"path/filepath"
)

func init() {
	// Add MSYS2 mingw64 bin to PATH if not already present
	msys2Path := `C:\msys64\mingw64\bin`
	path := os.Getenv("PATH")
	if path == "" {
		os.Setenv("PATH", msys2Path)
	} else {
		os.Setenv("PATH", msys2Path+string(filepath.ListSeparator)+path)
	}
}
