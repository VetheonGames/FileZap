// Copyright (c) 2025 The FileZap developers

package config

import (
    "fmt"
    "net"
    "os"
    "path/filepath"
    "strconv"

    "github.com/VetheonGames/FileZap/filezap-chain/types"
    "github.com/jessevdk/go-flags"
)

const (
    defaultConfigFilename     = "filezap.conf"
    defaultDataDirname       = "data"
    defaultLogLevel          = "info"
    defaultMaxPeers          = 125
    defaultBanDuration       = "24h"
    defaultBanThreshold      = 100
    defaultMaxFileSize       = 1073741824 // 1GB
    defaultMaxChunkSize      = 1048576    // 1MB
    defaultMinTokensRequired = 100        // 100 ZAP tokens
)

var (
    defaultDataDir    = filepath.Join(os.Getenv("HOME"), ".filezap")
    defaultConfigFile = filepath.Join(defaultDataDir, defaultConfigFilename)
    defaultLogDir     = filepath.Join(defaultDataDir, "logs")
)

// Config defines the configuration options for filezapd.
type Config struct {
    // General application behavior
    ConfigFile  string   `short:"C" long:"configfile" description:"Path to configuration file"`
    DataDir     string   `short:"b" long:"datadir" description:"Directory to store data"`
    LogDir      string   `long:"logdir" description:"Directory to log output"`
    LogLevel    string   `long:"loglevel" description:"Logging level for all subsystems"`
    Profile     string   `long:"profile" description:"Enable HTTP profiling on given port"`
    DebugLevel  string   `long:"debuglevel" description:"Debug level for subsystems"`

    // Memory profiling options
    MemoryProfile string `long:"memprofile" description:"Write memory profile to the specified file"`
    CPUProfile   string `long:"cpuprofile" description:"Write CPU profile to the specified file"`
    TraceProfile string `long:"traceprofile" description:"Write execution trace to specified file"`

    // Network settings
    Listen         []string `long:"listen" description:"Add interfaces/ports to listen on"`
    MaxPeers      int      `long:"maxpeers" description:"Max number of inbound and outbound peers"`
    ExternalIP    string   `long:"externalip" description:"External IP address to announce"`
    NoListen      bool     `long:"nolisten" description:"Disable listening for incoming connections"`
    Upnp          bool     `long:"upnp" description:"Use UPnP to map listening port"`
    AgentBlacklist []string `long:"agentblacklist" description:"A list of user-agent substrings to reject"`
    AgentWhitelist []string `long:"agentwhitelist" description:"A list of user-agent substrings to allow"`

    // Protocol settings
    TestNet    bool   `long:"testnet" description:"Use the test network"`
    Network    string `long:"network" description:"Network to connect to (mainnet, testnet)"`
    DbType     string `long:"dbtype" description:"Database backend to use"`
    NoFilters  bool   `long:"nofilters" description:"Disable compact filtering"`

    // File storage settings
    MaxFileSize    int64 `long:"maxfilesize" description:"Maximum file size in bytes"`
    MaxChunkSize   int64 `long:"maxchunksize" description:"Maximum chunk size in bytes"`
    MinTokens      int64 `long:"mintokens" description:"Minimum tokens required for operations"`
    StorageDir     string `long:"storagedir" description:"Directory to store file chunks"`
    ReplicationFactor int  `long:"replication" description:"Number of replicas to maintain"`

    // Validator settings
    ValidatorEnabled bool   `long:"validator" description:"Enable validator mode"`
    ValidatorKey     string `long:"validatorkey" description:"Path to validator private key"`
    StakeAmount      int64  `long:"stake" description:"Amount of tokens to stake"`

    // Active network parameters
    ActiveNet *types.Params
}

// LoadConfig initializes and parses the config using command line options.
func LoadConfig() (*Config, []string, error) {
    // Default config
    cfg := Config{
        ConfigFile:    defaultConfigFile,
        DataDir:       defaultDataDir,
        LogDir:        defaultLogDir,
        LogLevel:      defaultLogLevel,
        MaxPeers:      defaultMaxPeers,
        MaxFileSize:   defaultMaxFileSize,
        MaxChunkSize:  defaultMaxChunkSize,
        MinTokens:     defaultMinTokensRequired,
        DbType:        "leveldb",
        Listeners:     []string{},
    }

    // Parse command line options
    parser := flags.NewParser(&cfg, flags.Default)
    remainingArgs, err := parser.Parse()
    if err != nil {
        if e, ok := err.(*flags.Error); !ok || e.Type != flags.ErrHelp {
            parser.WriteHelp(os.Stderr)
        }
        return nil, nil, err
    }

    // Multiple networks can't be selected simultaneously.
    numNets := 0
    if cfg.TestNet {
        numNets++
        cfg.ActiveNet = types.TestnetParams
    }
    if numNets > 1 {
        str := "Multiple network flags specified. Please specify only one"
        return nil, nil, fmt.Errorf(str)
    }

    // Set default network if none specified
    if numNets == 0 {
        cfg.ActiveNet = types.MainnetParams
    }

    // Validate network-specific options
    if err := validateNetworkOptions(&cfg); err != nil {
        return nil, nil, err
    }

    // Validate file storage options
    if err := validateStorageOptions(&cfg); err != nil {
        return nil, nil, err
    }

    // Ensure paths exist
    if err := ensurePaths(&cfg); err != nil {
        return nil, nil, err
    }

    return &cfg, remainingArgs, nil
}

// validateNetworkOptions checks network-specific options for validity.
func validateNetworkOptions(cfg *Config) error {
    // Port ranges
    for _, addr := range cfg.Listen {
        host, portStr, err := net.SplitHostPort(addr)
        if err != nil {
            return err
        }

        if len(host) == 0 {
            return fmt.Errorf("listen interfaces can't be empty")
        }

        port, err := strconv.Atoi(portStr)
        if err != nil || port < 1 || port > 65535 {
            return fmt.Errorf("invalid port %q", portStr)
        }
    }

    return nil
}

// validateStorageOptions checks storage-related options for validity.
func validateStorageOptions(cfg *Config) error {
    if cfg.MaxFileSize < cfg.MaxChunkSize {
        return fmt.Errorf("max file size must be greater than max chunk size")
    }

    if cfg.ReplicationFactor < 1 {
        return fmt.Errorf("replication factor must be at least 1")
    }

    return nil
}

// ensurePaths creates necessary directories if they don't exist.
func ensurePaths(cfg *Config) error {
    // Create data directory if it doesn't exist
    if err := os.MkdirAll(cfg.DataDir, 0700); err != nil {
        return fmt.Errorf("failed to create data directory: %v", err)
    }

    // Create log directory if it doesn't exist
    if err := os.MkdirAll(cfg.LogDir, 0700); err != nil {
        return fmt.Errorf("failed to create log directory: %v", err)
    }

    if cfg.StorageDir != "" {
        // Create storage directory if it doesn't exist
        if err := os.MkdirAll(cfg.StorageDir, 0700); err != nil {
            return fmt.Errorf("failed to create storage directory: %v", err)
        }
    }

    return nil
}
