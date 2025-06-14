package keymanager

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"fmt"
	"sync"
)

// KeyShare represents a portion of a decryption key
type KeyShare struct {
	PeerID    string
	ShareData []byte
}

// KeyRequest represents a client's request for a decryption key
type KeyRequest struct {
	FileID      string
	ClientID    string
	PublicKey   []byte
	RequestTime int64
}

// KeyManager handles secure key distribution
type KeyManager struct {
	shares    map[string][]KeyShare // map[fileID][]KeyShare
	requests  map[string]*KeyRequest
	threshold int // minimum shares needed for key reconstruction
	mu        sync.RWMutex
}

// NewKeyManager creates a new key manager instance
func NewKeyManager(threshold int) *KeyManager {
	return &KeyManager{
		shares:    make(map[string][]KeyShare),
		requests:  make(map[string]*KeyRequest),
		threshold: threshold,
	}
}

// GenerateKeyShares splits a decryption key into shares
func (km *KeyManager) GenerateKeyShares(fileID string, key []byte, peerCount int) ([]KeyShare, error) {
	if peerCount < km.threshold {
		return nil, fmt.Errorf("peer count must be >= threshold")
	}

	// Generate random shares that XOR together to form the key
	shares := make([]KeyShare, peerCount)
	finalShare := make([]byte, len(key))
	copy(finalShare, key)

	// Generate random shares except for the last one
	for i := 0; i < peerCount-1; i++ {
		shareData := make([]byte, len(key))
		if _, err := rand.Read(shareData); err != nil {
			return nil, fmt.Errorf("failed to generate random share: %v", err)
		}

		shares[i] = KeyShare{
			ShareData: shareData,
		}

		// XOR this share with the running total
		for j := range finalShare {
			finalShare[j] ^= shareData[j]
		}
	}

	// Last share is what's needed to reconstruct the key
	shares[peerCount-1] = KeyShare{
		ShareData: finalShare,
	}

	km.mu.Lock()
	km.shares[fileID] = shares
	km.mu.Unlock()

	return shares, nil
}

// RegisterKeyRequest registers a client's request for a decryption key
func (km *KeyManager) RegisterKeyRequest(req *KeyRequest) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	// Store the request
	km.requests[req.FileID] = req
	return nil
}

// GetKeyShare returns a peer's key share for a file
func (km *KeyManager) GetKeyShare(fileID, peerID string) (*KeyShare, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	shares, exists := km.shares[fileID]
	if !exists {
		return nil, fmt.Errorf("no shares found for file")
	}

	// Find the share assigned to this peer
	for _, share := range shares {
		if share.PeerID == peerID {
			return &share, nil
		}
	}

	return nil, fmt.Errorf("no share found for peer")
}

// RecombineKeyShares reconstructs the original key from shares
func (km *KeyManager) RecombineKeyShares(fileID string, shares []KeyShare) ([]byte, error) {
	if len(shares) < km.threshold {
		return nil, fmt.Errorf("insufficient shares for key reconstruction")
	}

	// Verify all shares belong to the same file
	key := make([]byte, len(shares[0].ShareData))
	copy(key, shares[0].ShareData)

	// XOR all shares together
	for i := 1; i < len(shares); i++ {
		for j := range key {
			key[j] ^= shares[i].ShareData[j]
		}
	}

	return key, nil
}

// EncryptKeyShare encrypts a key share for a specific client
func (km *KeyManager) EncryptKeyShare(share []byte, publicKey []byte) ([]byte, error) {
	// Parse the public key
	pub, err := x509.ParsePKIXPublicKey(publicKey)
	if err != nil {
		return nil, fmt.Errorf("invalid public key: %v", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not RSA")
	}

	// Encrypt the share
	encrypted, err := rsa.EncryptOAEP(
		sha256.New(),
		rand.Reader,
		rsaPub,
		share,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt share: %v", err)
	}

	return encrypted, nil
}
