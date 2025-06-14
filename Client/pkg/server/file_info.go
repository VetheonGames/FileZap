package server

// ChunkInfo represents information about a file chunk
type ChunkInfo struct {
    ID      string
    Size    int64
    Hash    string
    Index   int
}

// FileInfo represents information about a split file
type FileInfo struct {
    ID       string
    Name     string
    ChunkDir string
    Chunks   []ChunkInfo
    Metadata []byte
    TotalSize int64
}

// GetChunkIDs returns the IDs of all chunks in the file
func (f *FileInfo) GetChunkIDs() []string {
    ids := make([]string, len(f.Chunks))
    for i, chunk := range f.Chunks {
        ids[i] = chunk.ID
    }
    return ids
}
