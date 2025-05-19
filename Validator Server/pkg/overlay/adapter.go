package overlay

import (
    "context"
    "fmt"
    "sync"

    networkoverlay "github.com/VetheonGames/FileZap/NetworkCore/pkg/overlay"
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
    Body       []byte
}

// RequestHandler is a function that handles overlay network requests
type RequestHandler func(r *Request) (*Response, error)

// Adapter wraps the network core's overlay adapter to provide validator-specific functionality
type Adapter struct {
    adapter *networkoverlay.ServerAdapter
    ctx     context.Context
    cancel  context.CancelFunc
    mu      sync.RWMutex
}

// NewAdapter creates a new overlay network adapter for the validator server
func NewAdapter(ctx context.Context) (*Adapter, error) {
    ctx, cancel := context.WithCancel(ctx)

    adapter, err := networkoverlay.NewServerAdapter(ctx)
    if err != nil {
        cancel()
        return nil, fmt.Errorf("failed to create network adapter: %v", err)
    }

    return &Adapter{
        adapter: adapter,
        ctx:     ctx,
        cancel:  cancel,
    }, nil
}

// Close shuts down the overlay adapter
func (a *Adapter) Close() error {
    a.cancel()
    return a.adapter.Close()
}

// GetNodeID returns the overlay node ID
func (a *Adapter) GetNodeID() string {
    return a.adapter.GetNodeID()
}

// HandleFunc registers a handler for a specific method and path
func (a *Adapter) HandleFunc(method, path string, handler RequestHandler) {
    overlayHandler := func(r *networkoverlay.Request) (*networkoverlay.Response, error) {
        req := &Request{
            Method: r.Method,
            Path:   r.Path,
            Body:   r.Body,
        }
        
        resp, err := handler(req)
        if err != nil {
            return nil, err
        }
        
        return &networkoverlay.Response{
            StatusCode: resp.StatusCode,
            Body:      resp.Body,
        }, nil
    }
    
    a.adapter.HandleFunc(method, path, overlayHandler)
}
