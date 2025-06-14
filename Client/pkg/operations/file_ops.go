package operations

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strconv"
    "time"

    "github.com/VetheonGames/FileZap/Client/pkg/server"
)

const defaultBufferSize = 32 * 1024 // 32KB buffer

// FileOperations handles file splitting and joining operations
type FileOperations struct {
    server *server.IntegratedServer
}

// NewFileOperations creates a new FileOperations instance
func NewFileOperations(server *server.IntegratedServer) *FileOperations {
    return &FileOperations{
        server: server,
    }
}

// SplitFile splits a file into chunks and generates a .zap file
func (f *FileOperations) SplitFile(inputPath, outputPath, chunkSizeStr string) error {
    // Parse chunk size
    chunkSize, err := strconv.ParseInt(chunkSizeStr, 10, 64)
    if err != nil {
        return fmt.Errorf("invalid chunk size: %v", err)
    }

    // Create output directory if it doesn't exist
    if err := os.MkdirAll(outputPath, 0755); err != nil {
        return fmt.Errorf("failed to create output directory: %v", err)
    }

    // Open input file
    file, err := os.Open(inputPath)
    if err != nil {
        return fmt.Errorf("failed to open input file: %v", err)
    }
    defer file.Close()

    // Get file info
    fileInfo, err := file.Stat()
    if err != nil {
        return fmt.Errorf("failed to get file info: %v", err)
    }

    // Create FileInfo structure
    info := &server.FileInfo{
        Name:      fileInfo.Name(),
        ID:        generateFileID(inputPath),
        ChunkDir:  outputPath,
        TotalSize: fileInfo.Size(),
    }

    // Create buffer for reading
    buffer := make([]byte, chunkSize)
    chunkIndex := 0

    for {
        n, err := file.Read(buffer)
        if err == io.EOF {
            break
        }
        if err != nil {
            return fmt.Errorf("error reading file: %v", err)
        }

        // Generate chunk ID
        chunkData := buffer[:n]
        chunkID := generateChunkID(chunkData)

        // Save chunk to file
        chunkPath := filepath.Join(outputPath, chunkID)
        if err := os.WriteFile(chunkPath, chunkData, 0644); err != nil {
            return fmt.Errorf("failed to write chunk %s: %v", chunkID, err)
        }

        // Add chunk info
        info.Chunks = append(info.Chunks, server.ChunkInfo{
            ID:    chunkID,
            Size:  int64(n),
            Hash:  chunkID,
            Index: chunkIndex,
        })

        chunkIndex++
    }

    // Register with server
    if err := f.server.RegisterFile(info); err != nil {
        return fmt.Errorf("failed to register file: %v", err)
    }

    // Save manifest
    manifestPath := filepath.Join(outputPath, info.Name+".zap")
    if err := saveManifest(manifestPath, info); err != nil {
        return fmt.Errorf("failed to save manifest: %v", err)
    }

    return nil
}

// JoinFile joins chunks back into the original file using a .zap file
func (f *FileOperations) JoinFile(zapPath, outputPath string) error {
    // Load manifest
    info, err := loadManifest(zapPath)
    if err != nil {
        return fmt.Errorf("failed to load manifest: %v", err)
    }

    // Create output directory if it doesn't exist
    if err := os.MkdirAll(outputPath, 0755); err != nil {
        return fmt.Errorf("failed to create output directory: %v", err)
    }

    // Get chunks from network if needed
    peers := f.server.GetPeersWithFile(info.ID)
    if len(peers) > 0 {
        if err := f.server.FetchChunks(info, peers[0]); err != nil {
            return fmt.Errorf("failed to fetch chunks: %v", err)
        }
    }

    // Create output file
    outputFile := filepath.Join(outputPath, info.Name)
    out, err := os.Create(outputFile)
    if err != nil {
        return fmt.Errorf("failed to create output file: %v", err)
    }
    defer out.Close()

    // Write chunks in order
    for i := range info.Chunks {
        chunkPath := filepath.Join(info.ChunkDir, info.Chunks[i].ID)
        chunkData, err := os.ReadFile(chunkPath)
        if err != nil {
            return fmt.Errorf("failed to read chunk %s: %v", info.Chunks[i].ID, err)
        }

        if _, err := out.Write(chunkData); err != nil {
            return fmt.Errorf("failed to write chunk data: %v", err)
        }
    }

    return nil
}

// Helper functions

func generateFileID(path string) string {
    data := []byte(path + strconv.FormatInt(time.Now().UnixNano(), 10))
    hash := sha256.Sum256(data)
    return hex.EncodeToString(hash[:])
}

func generateChunkID(data []byte) string {
    hash := sha256.Sum256(data)
    return hex.EncodeToString(hash[:])
}

func saveManifest(path string, info *server.FileInfo) error {
    data, err := json.Marshal(info)
    if err != nil {
        return err
    }
    return os.WriteFile(path, data, 0644)
}

func loadManifest(path string) (*server.FileInfo, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    var info server.FileInfo
    if err := json.Unmarshal(data, &info); err != nil {
        return nil, err
    }

    return &info, nil
}
