// Copyright (c) 2025 The FileZap developers

package indexers

import (
    "bytes"
    "encoding/binary"
    "fmt"

    "github.com/VetheonGames/FileZap/filezap-chain/database"
)

const (
    // contentIndexName is the name of the content hash index.
    contentIndexName = "content hash index"

    // similarityThreshold defines the minimum similarity required for
    // content to be considered a duplicate.
    similarityThreshold = 0.95
)

var (
    // contentIndexKey is the key of the content index bucket.
    contentIndexKey = []byte("contentidx")

    // HashBucket is the bucket used to store content hash entries.
    HashBucket = []byte("contenthash")

    // MetadataBucket stores metadata about content hashes.
    MetadataBucket = []byte("contentmeta")
)

// ContentEntry represents an indexed content hash entry.
type ContentEntry struct {
    Hash       []byte
    Size       int64
    Timestamp  int64
    Submitter  []byte
    ChunkCount int
    Signature  []byte
}

// ContentIndex implements a content hash index for FileZap's proof-of-uniqueness.
type ContentIndex struct {
    db          database.DB
    chainParams interface{} // Will be defined based on chain parameters
}

// DropContentIndex drops the content hash index.
func DropContentIndex(db database.DB, interrupt <-chan struct{}) error {
    return dropIndex(db, contentIndexKey, contentIndexName, interrupt)
}

// NewContentIndex creates a new content hash index.
func NewContentIndex(db database.DB, chainParams interface{}) (*ContentIndex, error) {
    // Create the index structure
    idx := &ContentIndex{
        db:          db,
        chainParams: chainParams,
    }

    // Initialize the index buckets
    err := idx.Init()
    if err != nil {
        return nil, err
    }

    return idx, nil
}

// Create creates initial index buckets and state.
func (idx *ContentIndex) Create(tx database.Transaction) error {
    // Create content hash bucket
    _, err := tx.CreateBucket(HashBucket)
    if err != nil {
        if dbErr, ok := err.(database.Error); !ok || dbErr.ErrorCode != database.ErrBucketExists {
            return err
        }
    }

    // Create metadata bucket
    _, err = tx.CreateBucket(MetadataBucket)
    if err != nil {
        if dbErr, ok := err.(database.Error); !ok || dbErr.ErrorCode != database.ErrBucketExists {
            return err
        }
    }

    return nil
}

// Init initializes the index.
func (idx *ContentIndex) Init() error {
    // Create buckets in a write transaction
    err := idx.db.Update(func(tx database.Transaction) error {
        return idx.Create(tx)
    })
    return err
}

// Connect implements the Indexer interface
func (idx *ContentIndex) Connect(tx database.Transaction) error {
    // Nothing to do for content index on block connection
    return nil
}

// Disconnect implements the Indexer interface
func (idx *ContentIndex) Disconnect(tx database.Transaction) error {
    // Nothing to do for content index on block disconnection
    return nil
}

// AddContent adds a new content entry to the index.
func (idx *ContentIndex) AddContent(entry *ContentEntry) error {
    if entry == nil || len(entry.Hash) == 0 {
        return fmt.Errorf("invalid content entry")
    }

    return idx.db.Update(func(tx database.Transaction) error {
        bucket := tx.Bucket(HashBucket)
        if bucket == nil {
            return fmt.Errorf("content hash bucket not found")
        }

        // Check if content already exists
        if bucket.Get(entry.Hash) != nil {
            return fmt.Errorf("content hash already exists")
        }

        // Serialize the entry
        var buf bytes.Buffer
        if err := binary.Write(&buf, binary.LittleEndian, entry.Size); err != nil {
            return err
        }
        if err := binary.Write(&buf, binary.LittleEndian, entry.Timestamp); err != nil {
            return err
        }
        if err := binary.Write(&buf, binary.LittleEndian, entry.ChunkCount); err != nil {
            return err
        }
        buf.Write(entry.Submitter)
        buf.Write(entry.Signature)

        // Store the entry
        return bucket.Put(entry.Hash, buf.Bytes())
    })
}

// GetContent retrieves a content entry by its hash.
func (idx *ContentIndex) GetContent(hash []byte) (*ContentEntry, error) {
    var entry *ContentEntry

    err := idx.db.View(func(tx database.Transaction) error {
        bucket := tx.Bucket(HashBucket)
        if bucket == nil {
            return fmt.Errorf("content hash bucket not found")
        }

        data := bucket.Get(hash)
        if data == nil {
            return fmt.Errorf("content hash not found")
        }

        // Deserialize the entry
        buf := bytes.NewReader(data)
        entry = &ContentEntry{
            Hash: hash,
        }

        if err := binary.Read(buf, binary.LittleEndian, &entry.Size); err != nil {
            return err
        }
        if err := binary.Read(buf, binary.LittleEndian, &entry.Timestamp); err != nil {
            return err
        }
        if err := binary.Read(buf, binary.LittleEndian, &entry.ChunkCount); err != nil {
            return err
        }

        submitterSize := (len(data) - buf.Len() - 64) // 64 is signature size
        entry.Submitter = make([]byte, submitterSize)
        if _, err := buf.Read(entry.Submitter); err != nil {
            return err
        }

        entry.Signature = make([]byte, 64)
        if _, err := buf.Read(entry.Signature); err != nil {
            return err
        }

        return nil
    })

    if err != nil {
        return nil, err
    }

    return entry, nil
}

// CheckSimilarity verifies if content is sufficiently different from existing entries.
// Returns nil if content is unique, error if too similar to existing content.
func (idx *ContentIndex) CheckSimilarity(hash []byte, metadata []byte) error {
    // This would implement the actual similarity checking algorithm
    // For now, we just check for exact matches
    var exists bool
    err := idx.db.View(func(tx database.Transaction) error {
        bucket := tx.Bucket(HashBucket)
        if bucket == nil {
            return fmt.Errorf("content hash bucket not found")
        }

        exists = bucket.Get(hash) != nil
        return nil
    })

    if err != nil {
        return err
    }

    if exists {
        return fmt.Errorf("identical content already exists")
    }

    return nil
}

// Name returns the name of the index.
func (idx *ContentIndex) Name() string {
    return contentIndexName
}
