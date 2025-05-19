// Copyright (c) 2025 The FileZap developers

package database

import (
    "fmt"
)

// DB defines the interface for a FileZap database.
type DB interface {
    // Type returns the database driver type.
    Type() string

    // Begin starts a transaction.
    Begin(writable bool) (Transaction, error)

    // View executes a function within a read-only transaction.
    View(fn func(tx Transaction) error) error

    // Update executes a function within a writable transaction.
    Update(fn func(tx Transaction) error) error

    // Close cleanly shuts down the database.
    Close() error
}

// Transaction represents a database transaction.
type Transaction interface {
    // Bucket returns the bucket with the given name.
    Bucket(name []byte) Bucket

    // CreateBucket creates a new bucket with the given name.
    CreateBucket(name []byte) (Bucket, error)

    // DeleteBucket deletes the bucket with the given name.
    DeleteBucket(name []byte) error

    // Commit commits the transaction.
    Commit() error

    // Rollback undoes all changes made within the transaction.
    Rollback() error
}

// Bucket represents a collection of key/value pairs.
type Bucket interface {
    // Get returns the value for the given key.
    Get(key []byte) []byte

    // Put sets the value for the given key.
    Put(key []byte, value []byte) error

    // Delete removes the given key.
    Delete(key []byte) error

    // ForEach calls the given function for each key/value pair.
    ForEach(fn func(k, v []byte) error) error
}

// Error represents a database error.
type Error struct {
    ErrorCode ErrorCode
    Description string
}

// ErrorCode identifies a kind of error.
type ErrorCode int

// Error codes.
const (
    // ErrDbTypeRegistered indicates a database type is already registered.
    ErrDbTypeRegistered ErrorCode = iota

    // ErrDbUnknownType indicates an unknown database type.
    ErrDbUnknownType

    // ErrDbDoesNotExist indicates a database does not exist.
    ErrDbDoesNotExist

    // ErrDbExists indicates a database already exists.
    ErrDbExists

    // ErrDbNotOpen indicates a database is not open.
    ErrDbNotOpen

    // ErrDbAlreadyOpen indicates a database is already open.
    ErrDbAlreadyOpen

    // ErrInvalid indicates invalid parameters to database operation.
    ErrInvalid
)

// Error satisfies the error interface and prints human-readable errors.
func (e Error) Error() string {
    return fmt.Sprintf("database error: %s", e.Description)
}

// driver defines a database driver.
type driver struct {
    DbType string
    Create func(path string, net uint32) (DB, error)
    Open   func(path string, net uint32) (DB, error)
}

var (
    drivers = make(map[string]*driver)
)

// RegisterDriver registers a database driver.
func RegisterDriver(dbType string, create func(path string, net uint32) (DB, error),
    open func(path string, net uint32) (DB, error)) error {

    if _, exists := drivers[dbType]; exists {
        str := fmt.Sprintf("driver %q is already registered", dbType)
        return Error{ErrorCode: ErrDbTypeRegistered, Description: str}
    }

    drivers[dbType] = &driver{
        DbType: dbType,
        Create: create,
        Open:   open,
    }

    return nil
}

// Create initializes and opens a database.
func Create(dbType string, path string, net uint32) (DB, error) {
    drv, exists := drivers[dbType]
    if !exists {
        str := fmt.Sprintf("driver %q is not registered", dbType)
        return nil, Error{ErrorCode: ErrDbUnknownType, Description: str}
    }

    return drv.Create(path, net)
}

// Open opens an existing database.
func Open(dbType string, path string, net uint32) (DB, error) {
    drv, exists := drivers[dbType]
    if !exists {
        str := fmt.Sprintf("driver %q is not registered", dbType)
        return nil, Error{ErrorCode: ErrDbUnknownType, Description: str}
    }

    return drv.Open(path, net)
}
