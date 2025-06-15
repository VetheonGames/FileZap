package overlay

import (
    "context"
    "fmt"
    "sync"
    "time"
)

// mockAdapter provides a mock implementation of the Adapter interface for testing
type mockAdapter struct {
    nodeID     string
    handlers   map[string]map[string]HandlerFunc
    closed     bool
    peers      []string
    peersMutex sync.RWMutex
    handlerMu  sync.RWMutex
}

// NewMockAdapter creates a mock adapter for testing
func NewMockAdapter() Adapter {
    return &mockAdapter{
        nodeID:   fmt.Sprintf("mock-%d", time.Now().UnixNano()),
        handlers: make(map[string]map[string]HandlerFunc),
        peers:    make([]string, 0),
        closed:   false,
    }
}

// NewAdapter is the package-level constructor for creating a new adapter
func NewAdapter(ctx context.Context) (Adapter, error) {
    if ctx.Err() != nil {
        return nil, fmt.Errorf("context already cancelled")
    }

    // For now, we just return a mock adapter for development
    mock := NewMockAdapter()
    return mock, nil
}

func (m *mockAdapter) GetNodeID() string {
    return m.nodeID
}

func (m *mockAdapter) Close() error {
    m.handlerMu.Lock()
    m.closed = true
    m.handlerMu.Unlock()
    return nil
}

func (m *mockAdapter) HandleFunc(method, path string, handler HandlerFunc) {
    m.handlerMu.Lock()
    defer m.handlerMu.Unlock()

    if m.closed {
        return
    }

    if m.handlers[method] == nil {
        m.handlers[method] = make(map[string]HandlerFunc)
    }
    m.handlers[method][path] = handler
}

func (m *mockAdapter) HandleRequest(r *Request) (*Response, error) {
    m.handlerMu.RLock()
    if m.closed {
        m.handlerMu.RUnlock()
        return &Response{StatusCode: 503}, nil // Service Unavailable
    }

    methodHandlers, exists := m.handlers[r.Method]
    if !exists {
        m.handlerMu.RUnlock()
        return &Response{StatusCode: 405}, nil // Method Not Allowed
    }

    handler, exists := methodHandlers[r.Path]
    m.handlerMu.RUnlock()

    if !exists {
        return &Response{StatusCode: 404}, nil // Not Found
    }

    resp, err := handler(r)
    if err != nil {
        return &Response{StatusCode: 500}, nil // Internal Server Error
    }
    return resp, nil
}

func (m *mockAdapter) ConnectTo(ctx context.Context, peerID string) error {
    if ctx.Err() != nil {
        return ctx.Err()
    }

    m.peersMutex.Lock()
    defer m.peersMutex.Unlock()

    if m.closed {
        return fmt.Errorf("adapter is closed")
    }

    for _, p := range m.peers {
        if p == peerID {
            return nil // Already connected
        }
    }
    m.peers = append(m.peers, peerID)
    return nil
}

func (m *mockAdapter) Peers() []string {
    m.peersMutex.RLock()
    defer m.peersMutex.RUnlock()

    peers := make([]string, len(m.peers))
    copy(peers, m.peers)
    return peers
}

func (m *mockAdapter) SendMessage(ctx context.Context, peerID string, req *Request) (*Response, error) {
    if ctx.Err() != nil {
        return nil, ctx.Err()
    }

    m.peersMutex.RLock()
    if m.closed {
        m.peersMutex.RUnlock()
        return nil, fmt.Errorf("adapter is closed")
    }

    connected := false
    for _, p := range m.peers {
        if p == peerID {
            connected = true
            break
        }
    }
    m.peersMutex.RUnlock()

    if !connected {
        return nil, fmt.Errorf("not connected to peer %s", peerID)
    }

    // In a real implementation, this would send the message over the network
    // For testing, we just process it locally
    return m.HandleRequest(req)
}

// SetClosed allows explicitly setting the closed state for testing
func (m *mockAdapter) SetClosed(closed bool) {
    m.handlerMu.Lock()
    m.closed = closed
    m.handlerMu.Unlock()
}
