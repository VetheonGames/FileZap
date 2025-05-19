// Copyright (c) 2025 The FileZap developers

package types

import (
    "fmt"
    "io"
    "log"
    "os"
)

// Params defines the parameters unique to FileZap network.
type Params struct {
    // Net identifies the message start bytes unique to the FileZap network.
    Net uint32

    // Name defines a human-readable identifier for the network.
    Name string

    // DefaultPort defines the default peer-to-peer port for the network.
    DefaultPort string

    // DNSSeeds defines the DNS seed hostnames for the network.
    DNSSeeds []string

    // RelayNonStdTxs defines whether to relay non-standard transactions.
    RelayNonStdTxs bool

    // BlockHeight is the current block height.
    BlockHeight uint32

    // MinTokensForUpload is the minimum number of tokens required to upload a file.
    MinTokensForUpload uint64

    // MinTokensForStaking is the minimum number of tokens required for staking.
    MinTokensForStaking uint64

    // MinValidators is the minimum number of validators required.
    MinValidators int

    // MaxValidators is the maximum number of validators allowed.
    MaxValidators int

    // MinFileSize is the minimum size of a file that can be uploaded.
    MinFileSize int64

    // MaxFileSize is the maximum size of a file that can be uploaded.
    MaxFileSize int64

    // MinChunkSize is the minimum size of a file chunk.
    MinChunkSize int64

    // MaxChunkSize is the maximum size of a file chunk.
    MaxChunkSize int64
}

// Logger provides structured logging capabilities.
type Logger struct {
    *log.Logger
    subsystem string
}

// LogBackend handles the creation and management of loggers.
type LogBackend struct {
    output io.Writer
    loggers map[string]*Logger
}

// NewLogBackend creates a new logging backend that writes to standard output.
func NewLogBackend() *LogBackend {
    return &LogBackend{
        output: os.Stdout,
        loggers: make(map[string]*Logger),
    }
}

// NewSubLogger creates a new logger for a subsystem.
func NewSubLogger(subsystem string, backend *LogBackend) *Logger {
    if logger, exists := backend.loggers[subsystem]; exists {
        return logger
    }

    logger := &Logger{
        Logger: log.New(backend.output, fmt.Sprintf("[%s] ", subsystem), log.LstdFlags),
        subsystem: subsystem,
    }
    backend.loggers[subsystem] = logger
    return logger
}

// Info logs an informational message.
func (l *Logger) Info(args ...interface{}) {
    l.Printf("INFO: %s", fmt.Sprint(args...))
}

// Infof logs a formatted informational message.
func (l *Logger) Infof(format string, args ...interface{}) {
    l.Printf("INFO: "+format, args...)
}

// Error logs an error message.
func (l *Logger) Error(args ...interface{}) {
    l.Printf("ERROR: %s", fmt.Sprint(args...))
}

// Errorf logs a formatted error message.
func (l *Logger) Errorf(format string, args ...interface{}) {
    l.Printf("ERROR: "+format, args...)
}

// Parameter constants
const (
    // Mainnet parameters
    MainnetDefaultPort = "9333"
    MainnetName       = "mainnet"

    // Testnet parameters
    TestnetDefaultPort = "19333"
    TestnetName       = "testnet"

    // Network protocol constants
    DefaultMinChunkSize = 1 << 20    // 1 MB
    DefaultMaxChunkSize = 1 << 30    // 1 GB
    DefaultMinFileSize  = 1 << 10    // 1 KB
    DefaultMaxFileSize  = 1 << 40    // 1 TB

    // Economic constants
    DefaultMinTokensForUpload = 100   // 100 ZAP
    DefaultMinTokensForStaking = 1000 // 1000 ZAP

    // Consensus constants
    DefaultMinValidators = 3
    DefaultMaxValidators = 100
)

// MainnetParams defines the parameters for the main FileZap network.
var MainnetParams = &Params{
    Name:                MainnetName,
    Net:                 0xf11e2a0,  // Unique to FileZap
    DefaultPort:         MainnetDefaultPort,
    DNSSeeds: []string{
        "seed1.filezap.net",
        "seed2.filezap.net",
        "seed3.filezap.net",
    },
    RelayNonStdTxs:      false,
    MinTokensForUpload:   DefaultMinTokensForUpload,
    MinTokensForStaking:  DefaultMinTokensForStaking,
    MinValidators:        DefaultMinValidators,
    MaxValidators:        DefaultMaxValidators,
    MinFileSize:          DefaultMinFileSize,
    MaxFileSize:          DefaultMaxFileSize,
    MinChunkSize:        DefaultMinChunkSize,
    MaxChunkSize:        DefaultMaxChunkSize,
}

// TestnetParams defines the parameters for the FileZap test network.
var TestnetParams = &Params{
    Name:                TestnetName,
    Net:                 0xf11e2a1,  // Unique to FileZap testnet
    DefaultPort:         TestnetDefaultPort,
    DNSSeeds: []string{
        "testnet-seed1.filezap.net",
        "testnet-seed2.filezap.net",
    },
    RelayNonStdTxs:      true,
    MinTokensForUpload:   DefaultMinTokensForUpload / 10,
    MinTokensForStaking:  DefaultMinTokensForStaking / 10,
    MinValidators:        DefaultMinValidators,
    MaxValidators:        DefaultMaxValidators,
    MinFileSize:          DefaultMinFileSize,
    MaxFileSize:          DefaultMaxFileSize,
    MinChunkSize:        DefaultMinChunkSize,
    MaxChunkSize:        DefaultMaxChunkSize,
}
