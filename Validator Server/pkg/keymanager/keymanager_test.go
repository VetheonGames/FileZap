package keymanager

import (
    "fmt"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestKeyRequestHandling(t *testing.T) {
    km := NewKeyManager(3) // Require 3 shares for reconstruction

    t.Run("register valid request", func(t *testing.T) {
        req := &KeyRequest{
            FileID:      "test123",
            ClientID:    "client1",
            PublicKey:   []byte("testkey"),
            RequestTime: time.Now().Unix(),
        }

        err := km.RegisterKeyRequest(req)
        require.NoError(t, err)

        // Verify request was stored
        stored, err := km.GetKeyRequest("test123", "client1")
        assert.NoError(t, err)
        assert.Equal(t, req.FileID, stored.FileID)
        assert.Equal(t, req.ClientID, stored.ClientID)
        assert.Equal(t, req.PublicKey, stored.PublicKey)
    })

    t.Run("duplicate request", func(t *testing.T) {
        req := &KeyRequest{
            FileID:      "test123",
            ClientID:    "client1",
            PublicKey:   []byte("newkey"),
            RequestTime: time.Now().Unix(),
        }

        err := km.RegisterKeyRequest(req)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "already exists")
    })

    t.Run("invalid request", func(t *testing.T) {
        req := &KeyRequest{} // Empty request
        err := km.RegisterKeyRequest(req)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "invalid request")
    })
}

func TestKeyShareManagement(t *testing.T) {
    km := NewKeyManager(3)

    // Register initial key request
    req := &KeyRequest{
        FileID:      "test123",
        ClientID:    "client1",
        PublicKey:   []byte("testkey"),
        RequestTime: time.Now().Unix(),
    }
    err := km.RegisterKeyRequest(req)
    require.NoError(t, err)

    t.Run("register key share", func(t *testing.T) {
        share := &KeyShare{
            FileID:      "test123",
            ValidatorID: "validator1",
            Share:      []byte("share1"),
        }

        err := km.RegisterKeyShare(share)
        require.NoError(t, err)

        // Verify share was stored
        stored, err := km.GetKeyShare("test123", "validator1")
        assert.NoError(t, err)
        assert.Equal(t, share.Share, stored.Share)
    })

    t.Run("duplicate share", func(t *testing.T) {
        share := &KeyShare{
            FileID:      "test123",
            ValidatorID: "validator1",
            Share:      []byte("different_share"),
        }

        err := km.RegisterKeyShare(share)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "already exists")
    })

    t.Run("share without request", func(t *testing.T) {
        share := &KeyShare{
            FileID:      "nonexistent",
            ValidatorID: "validator1",
            Share:      []byte("share"),
        }

        err := km.RegisterKeyShare(share)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "no request found")
    })
}

func TestKeyReconstruction(t *testing.T) {
    km := NewKeyManager(3)

    // Register key request
    req := &KeyRequest{
        FileID:      "test123",
        ClientID:    "client1",
        PublicKey:   []byte("testkey"),
        RequestTime: time.Now().Unix(),
    }
    err := km.RegisterKeyRequest(req)
    require.NoError(t, err)

    t.Run("insufficient shares", func(t *testing.T) {
        // Add only 2 shares when 3 are required
        shares := []*KeyShare{
            {
                FileID:      "test123",
                ValidatorID: "validator1",
                Share:      []byte("share1"),
            },
            {
                FileID:      "test123",
                ValidatorID: "validator2",
                Share:      []byte("share2"),
            },
        }

        for _, share := range shares {
            err := km.RegisterKeyShare(share)
            require.NoError(t, err)
        }

        _, err := km.ReconstructKey("test123", "client1")
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "insufficient shares")
    })

    t.Run("successful reconstruction", func(t *testing.T) {
        // Add third share to meet threshold
        share := &KeyShare{
            FileID:      "test123",
            ValidatorID: "validator3",
            Share:      []byte("share3"),
        }
        err := km.RegisterKeyShare(share)
        require.NoError(t, err)

        key, err := km.ReconstructKey("test123", "client1")
        assert.NoError(t, err)
        assert.NotNil(t, key)
    })

    t.Run("nonexistent request", func(t *testing.T) {
        _, err := km.ReconstructKey("nonexistent", "client1")
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "no request found")
    })
}

func TestShareExpiration(t *testing.T) {
    km := NewKeyManager(3)

    // Create request with expired timestamp
    expiredReq := &KeyRequest{
        FileID:      "expired",
        ClientID:    "client1",
        PublicKey:   []byte("testkey"),
        RequestTime: time.Now().Add(-24 * time.Hour).Unix(), // 1 day old
    }
    err := km.RegisterKeyRequest(expiredReq)
    require.NoError(t, err)

    // Create fresh request
    freshReq := &KeyRequest{
        FileID:      "fresh",
        ClientID:    "client1",
        PublicKey:   []byte("testkey"),
        RequestTime: time.Now().Unix(),
    }
    err = km.RegisterKeyRequest(freshReq)
    require.NoError(t, err)

    t.Run("expired request", func(t *testing.T) {
        share := &KeyShare{
            FileID:      "expired",
            ValidatorID: "validator1",
            Share:      []byte("share1"),
        }

        err := km.RegisterKeyShare(share)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "request expired")
    })

    t.Run("fresh request", func(t *testing.T) {
        share := &KeyShare{
            FileID:      "fresh",
            ValidatorID: "validator1",
            Share:      []byte("share1"),
        }

        err := km.RegisterKeyShare(share)
        assert.NoError(t, err)
    })
}

func TestConcurrentKeyOperations(t *testing.T) {
    km := NewKeyManager(3)

    // Register base request
    req := &KeyRequest{
        FileID:      "concurrent",
        ClientID:    "client1",
        PublicKey:   []byte("testkey"),
        RequestTime: time.Now().Unix(),
    }
    err := km.RegisterKeyRequest(req)
    require.NoError(t, err)

    t.Run("concurrent share submissions", func(t *testing.T) {
        done := make(chan bool)
        for i := 0; i < 5; i++ {
            go func(id int) {
                share := &KeyShare{
                    FileID:      "concurrent",
                    ValidatorID: fmt.Sprintf("validator%d", id),
                    Share:      []byte(fmt.Sprintf("share%d", id)),
                }
                err := km.RegisterKeyShare(share)
                assert.NoError(t, err)
                done <- true
            }(i)
        }

        // Wait for all shares to be registered
        for i := 0; i < 5; i++ {
            <-done
        }

        // Try reconstruction
        key, err := km.ReconstructKey("concurrent", "client1")
        assert.NoError(t, err)
        assert.NotNil(t, key)
    })
}

func TestInvalidKeyOperations(t *testing.T) {
    km := NewKeyManager(3)

    t.Run("invalid threshold", func(t *testing.T) {
        invalidKM := NewKeyManager(0)
        assert.Nil(t, invalidKM)
    })

    t.Run("invalid share data", func(t *testing.T) {
        // Register valid request
        req := &KeyRequest{
            FileID:      "test",
            ClientID:    "client1",
            PublicKey:   []byte("testkey"),
            RequestTime: time.Now().Unix(),
        }
        err := km.RegisterKeyRequest(req)
        require.NoError(t, err)

        // Try to register invalid share
        share := &KeyShare{
            FileID:      "test",
            ValidatorID: "validator1",
            Share:      []byte{}, // Empty share
        }
        err = km.RegisterKeyShare(share)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "invalid share")
    })
}
