package validator

import (
"crypto/sha256"
"encoding/hex"
"errors"
"fmt"
"regexp"

"github.com/VetheonGames/FileZap/NetworkCore/pkg/types"
)

var (
ErrInvalidChunkSize    = errors.New("invalid chunk size")
ErrInvalidChunkHash    = errors.New("invalid chunk hash")
ErrInvalidPeerID       = errors.New("invalid peer ID")
ErrInvalidAddress      = errors.New("invalid network address")
ErrInvalidMessageType  = errors.New("invalid message type")
ErrInvalidManifest     = errors.New("invalid manifest format")
ErrManifestIncomplete  = errors.New("manifest missing required fields")
ErrUnsupportedVersion  = errors.New("unsupported manifest version")
)

const (
MinChunkSize = 1024              // 1KB
MaxChunkSize = 100 * 1024 * 1024 // 100MB
)

// ChunkValidator handles validation of file chunks
type ChunkValidator struct {
minSize int64
maxSize int64
}

// NewChunkValidator creates a new chunk validator with custom size limits
func NewChunkValidator(minSize, maxSize int64) *ChunkValidator {
return &ChunkValidator{
minSize: minSize,
maxSize: maxSize,
}
}

// ValidateChunk checks if a chunk is valid
func (cv *ChunkValidator) ValidateChunk(data []byte, expectedHash string) error {
size := int64(len(data))

if size < cv.minSize || size > cv.maxSize {
return fmt.Errorf("%w: size %d not between %d and %d", ErrInvalidChunkSize, size, cv.minSize, cv.maxSize)
}

hash := sha256.Sum256(data)
actualHash := hex.EncodeToString(hash[:])
if actualHash != expectedHash {
return fmt.Errorf("%w: expected %s, got %s", ErrInvalidChunkHash, expectedHash, actualHash)
}

return nil
}

// PeerValidator handles validation of peer information
type PeerValidator struct {
addressRegex *regexp.Regexp
}

// NewPeerValidator creates a new peer validator
func NewPeerValidator() *PeerValidator {
return &PeerValidator{
addressRegex: regexp.MustCompile(`^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5]):[0-9]+$`),
}
}

// ValidatePeerID checks if a peer ID is valid
func (pv *PeerValidator) ValidatePeerID(id string) error {
if len(id) == 0 {
return ErrInvalidPeerID
}
return nil
}

// ValidateAddress checks if a network address is valid
func (pv *PeerValidator) ValidateAddress(addr string) error {
if !pv.addressRegex.MatchString(addr) {
return fmt.Errorf("%w: %s", ErrInvalidAddress, addr)
}
return nil
}

// MessageValidator handles validation of network messages
type MessageValidator struct {
validTypes map[string]bool
}

// NewMessageValidator creates a new message validator
func NewMessageValidator(validTypes []string) *MessageValidator {
typeMap := make(map[string]bool)
for _, t := range validTypes {
typeMap[t] = true
}
return &MessageValidator{validTypes: typeMap}
}

// ValidateMessageType checks if a message type is valid
func (mv *MessageValidator) ValidateMessageType(msgType string) error {
if !mv.validTypes[msgType] {
return fmt.Errorf("%w: %s", ErrInvalidMessageType, msgType)
}
return nil
}

// ManifestValidator handles validation of ZAP file manifests
type ManifestValidator struct {
supportedVersions map[string]bool
}

// NewManifestValidator creates a new manifest validator
func NewManifestValidator(supportedVersions []string) *ManifestValidator {
versions := make(map[string]bool)
for _, v := range supportedVersions {
versions[v] = true
}
return &ManifestValidator{supportedVersions: versions}
}

// ValidateManifest checks if a manifest is valid
func (mv *ManifestValidator) ValidateManifest(manifest *types.FileInfo) error {
// Check required fields
if manifest.Name == "" {
return fmt.Errorf("%w: missing name", ErrManifestIncomplete)
}

if len(manifest.ChunkIDs) == 0 {
return fmt.Errorf("%w: missing chunk IDs", ErrManifestIncomplete)
}

// Validate each chunk ID format
for _, chunkID := range manifest.ChunkIDs {
if len(chunkID) != 64 { // SHA-256 hex string length
return fmt.Errorf("%w: invalid chunk ID format", ErrInvalidManifest)
}
}

return nil
}
