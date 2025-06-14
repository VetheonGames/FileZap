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

// HandlerFunc defines the handler function type for overlay requests
type HandlerFunc func(*Request) (*Response, error)

// Adapter defines the interface for overlay network operations
type Adapter interface {
    // Basic operations
    GetNodeID() string
    Close() error

    // Request handling
    HandleFunc(method, path string, handler HandlerFunc)
    HandleRequest(*Request) (*Response, error)

    // Network operations
    ConnectTo(context.Context, string) error
    Peers() []string
    SendMessage(context.Context, string, *Request) (*Response, error)
    StartDiscovery() error
    NotifyPeer(peerID string, action string, data map[string]string) error
}


// NewAdapter creates a new overlay network adapter
func NewAdapter(ctx context.Context) (Adapter, error) {
    return NewBasicAdapter(ctx)
}
