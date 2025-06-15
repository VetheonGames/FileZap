package overlay

import (
    "context"
)

// Request represents an overlay network request
type Request struct {
    Method string
    Path   string
    Body   []byte
}

// Response represents an overlay network response
type Response struct {
    StatusCode int
    Body      []byte
}

// HandlerFunc defines the function type for request handlers
type HandlerFunc func(*Request) (*Response, error)

// Adapter defines the interface for overlay network operations
type Adapter interface {
    // Basic operations
    GetNodeID() string
    Close() error

    // Request handling
    HandleFunc(method, path string, handler HandlerFunc)
    HandleRequest(req *Request) (*Response, error)

    // Network operations
    ConnectTo(ctx context.Context, peerID string) error
    Peers() []string
    SendMessage(ctx context.Context, peerID string, req *Request) (*Response, error)
}

// Option defines a function type for configuring adapters
type Option func(*adapterConfig)

// adapterConfig holds configuration for adapters
type adapterConfig struct {
    BootstrapPeers []string
}

// WithBootstrapPeers specifies a list of peers to connect to on startup
func WithBootstrapPeers(peers []string) Option {
    return func(cfg *adapterConfig) {
        cfg.BootstrapPeers = peers
    }
}

// defaultConfig returns the default adapter configuration
func defaultConfig() *adapterConfig {
    return &adapterConfig{
        BootstrapPeers: []string{},
    }
}
