// Copyright (c) 2025 The FileZap developers

package indexers

import (
    "fmt"

    "github.com/VetheonGames/FileZap/filezap-chain/database"
)

// Indexer provides a generic interface for blockchain index management.
type Indexer interface {
    // Name returns the human-readable name of the index.
    Name() string

    // Create creates the initial index state.
    Create(tx database.Transaction) error

    // Init initializes the index manager with the provided database.
    Init() error

    // Connect notifies the indexer that a new block has been connected to
    // the main chain.
    Connect(tx database.Transaction) error

    // Disconnect notifies the indexer that a block has been disconnected
    // from the main chain.
    Disconnect(tx database.Transaction) error
}

// dropIndex drops the specified index from the database. The idx parameter
// MUST be the key for the index bucket, idxName should be the human-readable
// name for the index, and the interrupt parameter provides a mechanism to
// cancel the operation. 
func dropIndex(db database.DB, idx []byte, idxName string, interrupt <-chan struct{}) error {
    // Nothing to do if the index doesn't already exist.
    exists := false
    err := db.View(func(tx database.Transaction) error {
        exists = (tx.Bucket(idx) != nil)
        return nil
    })
    if err != nil {
        return err
    }
    if !exists {
        return nil
    }

    // Remove the index bucket.
    err = db.Update(func(tx database.Transaction) error {
        return tx.DeleteBucket(idx)
    })
    if err != nil {
        return fmt.Errorf("failed to delete %s: %v", idxName, err)
    }

    return nil
}

// interruptRequested returns true when the provided channel has been closed.
// This is used to provide clean shutdown handling.
func interruptRequested(interrupted <-chan struct{}) bool {
    select {
    case <-interrupted:
        return true
    default:
    }

    return false
}
