package validator

import (
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "testing"
    
    "github.com/VetheonGames/FileZap/NetworkCore/pkg/types"
    "github.com/stretchr/testify/assert"
)

func TestCustomChunkValidator(t *testing.T) {
// Test custom size limits
cv := NewChunkValidator(2048, 4096) // 2KB to 4KB

tests := []struct {
name        string
data        []byte
hash        string
shouldError bool
errType     error
}{
{
name:        "Within custom limits",
data:        make([]byte, 3072), // 3KB
hash:        calculateHash(make([]byte, 3072)),
shouldError: false,
},
{
name:        "Below custom minimum",
data:        make([]byte, 1024), // 1KB
hash:        "any",
shouldError: true,
errType:     ErrInvalidChunkSize,
},
{
name:        "Above custom maximum",
data:        make([]byte, 5120), // 5KB
hash:        "any",
shouldError: true,
errType:     ErrInvalidChunkSize,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
err := cv.ValidateChunk(tt.data, tt.hash)
if tt.shouldError {
assert.Error(t, err)
assert.ErrorIs(t, err, tt.errType)
} else {
assert.NoError(t, err)
}
})
}
}

func TestChunkValidator(t *testing.T) {
cv := NewChunkValidator(MinChunkSize, MaxChunkSize)

tests := []struct {
name        string
data        []byte
hash        string
shouldError bool
errType     error
}{
{
name:        "Valid chunk",
data:        make([]byte, MinChunkSize),
hash:        calculateHash(make([]byte, MinChunkSize)),
shouldError: false,
},
{
name:        "Too small chunk",
data:        make([]byte, MinChunkSize-1),
hash:        "any",
shouldError: true,
errType:     ErrInvalidChunkSize,
},
{
name:        "Too large chunk",
data:        make([]byte, MaxChunkSize+1),
hash:        "any",
shouldError: true,
errType:     ErrInvalidChunkSize,
},
{
name:        "Invalid hash",
data:        make([]byte, MinChunkSize),
hash:        "invalid_hash",
shouldError: true,
errType:     ErrInvalidChunkHash,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
err := cv.ValidateChunk(tt.data, tt.hash)
if tt.shouldError {
assert.Error(t, err)
assert.ErrorIs(t, err, tt.errType)
} else {
assert.NoError(t, err)
}
})
}
}

func TestPeerValidator(t *testing.T) {
pv := NewPeerValidator()

// Test peer ID validation
t.Run("Peer ID validation", func(t *testing.T) {
validID := "valid_peer_id"
assert.NoError(t, pv.ValidatePeerID(validID))

invalidID := ""
assert.ErrorIs(t, pv.ValidatePeerID(invalidID), ErrInvalidPeerID)
})

// Test address validation
addressTests := []struct {
name        string
address     string
shouldError bool
}{
{
name:        "Valid IPv4 address",
address:     "192.168.1.1:8080",
shouldError: false,
},
{
name:        "Invalid IP format",
address:     "256.256.256.256:8080",
shouldError: true,
},
{
name:        "Missing port",
address:     "192.168.1.1",
shouldError: true,
},
{
name:        "Invalid format",
address:     "not_an_address",
shouldError: true,
},
}

for _, tt := range addressTests {
t.Run(tt.name, func(t *testing.T) {
err := pv.ValidateAddress(tt.address)
if tt.shouldError {
assert.Error(t, err)
assert.ErrorIs(t, err, ErrInvalidAddress)
} else {
assert.NoError(t, err)
}
})
}
}

func TestMessageValidator(t *testing.T) {
validTypes := []string{"GET", "PUT", "DELETE"}
mv := NewMessageValidator(validTypes)

tests := []struct {
name        string
msgType     string
shouldError bool
}{
{
name:        "Valid message type",
msgType:     "GET",
shouldError: false,
},
{
name:        "Invalid message type",
msgType:     "INVALID",
shouldError: true,
},
{
name:        "Empty message type",
msgType:     "",
shouldError: true,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
err := mv.ValidateMessageType(tt.msgType)
if tt.shouldError {
assert.Error(t, err)
assert.ErrorIs(t, err, ErrInvalidMessageType)
} else {
assert.NoError(t, err)
}
})
}
}

func TestConcurrentValidation(t *testing.T) {
cv := NewChunkValidator(MinChunkSize, MaxChunkSize)
pv := NewPeerValidator()
mv := NewMessageValidator([]string{"GET", "PUT"})

const numGoroutines = 100
done := make(chan bool)

// Test concurrent chunk validation
go func() {
data := make([]byte, MinChunkSize)
hash := calculateHash(data)
for i := 0; i < numGoroutines; i++ {
go func() {
cv.ValidateChunk(data, hash)
done <- true
}()
}
}()

// Test concurrent peer validation
go func() {
for i := 0; i < numGoroutines; i++ {
go func(i int) {
pv.ValidatePeerID(fmt.Sprintf("peer-%d", i))
pv.ValidateAddress("192.168.1.1:8080")
done <- true
}(i)
}
}()

// Test concurrent message validation
go func() {
for i := 0; i < numGoroutines; i++ {
go func(i int) {
if i%2 == 0 {
mv.ValidateMessageType("GET")
} else {
mv.ValidateMessageType("PUT")
}
done <- true
}(i)
}
}()

// Wait for all operations to complete
for i := 0; i < numGoroutines*3; i++ {
<-done
}
}

func TestEdgeCases(t *testing.T) {
cv := NewChunkValidator(MinChunkSize, MaxChunkSize)
pv := NewPeerValidator()

// Test nil/empty inputs
t.Run("Nil chunk data", func(t *testing.T) {
err := cv.ValidateChunk(nil, "hash")
assert.Error(t, err)
assert.ErrorIs(t, err, ErrInvalidChunkSize)
})

t.Run("Empty peer address", func(t *testing.T) {
err := pv.ValidateAddress("")
assert.Error(t, err)
assert.ErrorIs(t, err, ErrInvalidAddress)
})

t.Run("Malformed addresses", func(t *testing.T) {
malformedAddrs := []string{
"256.0.0.1:8080",    // Invalid IP
"192.168.1.1:",      // Missing port
":8080",             // Missing IP
"192.168.1.1:-1",    // Invalid port
"192.168.1.1:65536", // Port out of range
}

for _, addr := range malformedAddrs {
err := pv.ValidateAddress(addr)
assert.Error(t, err)
assert.ErrorIs(t, err, ErrInvalidAddress)
}
})

// Test boundary conditions
t.Run("Chunk size boundaries", func(t *testing.T) {
// Exactly minimum size
data := make([]byte, MinChunkSize)
hash := calculateHash(data)
assert.NoError(t, cv.ValidateChunk(data, hash))

// Exactly maximum size
data = make([]byte, MaxChunkSize)
hash = calculateHash(data)
assert.NoError(t, cv.ValidateChunk(data, hash))
})
}

func TestManifestValidator(t *testing.T) {
mv := NewManifestValidator([]string{"1.0.0", "1.1.0"})

validChunkID := calculateHash([]byte("test"))
tests := []struct {
name        string
manifest    *types.FileInfo
shouldError bool
errType     error
}{
{
name: "Valid manifest",
manifest: &types.FileInfo{
Name:     "test.txt",
ChunkIDs: []string{validChunkID},
},
shouldError: false,
},
{
name: "Missing name",
manifest: &types.FileInfo{
ChunkIDs: []string{validChunkID},
},
shouldError: true,
errType:     ErrManifestIncomplete,
},
{
name: "Empty chunk IDs",
manifest: &types.FileInfo{
Name:     "test.txt",
ChunkIDs: []string{},
},
shouldError: true,
errType:     ErrManifestIncomplete,
},
{
name: "Invalid chunk ID format",
manifest: &types.FileInfo{
Name:     "test.txt",
ChunkIDs: []string{"invalid_chunk_id"},
},
shouldError: true,
errType:     ErrInvalidManifest,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
err := mv.ValidateManifest(tt.manifest)
if tt.shouldError {
assert.Error(t, err)
assert.ErrorIs(t, err, tt.errType)
} else {
assert.NoError(t, err)
}
})
}
}

// Helper function to calculate SHA-256 hash
func calculateHash(data []byte) string {
hash := sha256.Sum256(data)
return hex.EncodeToString(hash[:])
}
