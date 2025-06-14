package overlay

import (
    "context"
    "encoding/json"
    "fmt"
    "sync"
)

type basicAdapter struct {
    ctx      context.Context
    handlers map[string]map[string]HandlerFunc // method -> path -> handler
    nodeID   string
    mu       sync.RWMutex
}

func NewBasicAdapter(ctx context.Context) (Adapter, error) {
    a := &basicAdapter{
        ctx:      ctx,
        handlers: make(map[string]map[string]HandlerFunc),
        nodeID:   "local-node", // For testing, would be replaced with real node ID
    }
    return a, nil
}

func (a *basicAdapter) GetNodeID() string {
    return a.nodeID
}

func (a *basicAdapter) Close() error {
    return nil
}

func (a *basicAdapter) HandleFunc(method string, path string, handler HandlerFunc) {
    a.mu.Lock()
    defer a.mu.Unlock()

    if a.handlers[method] == nil {
        a.handlers[method] = make(map[string]HandlerFunc)
    }
    a.handlers[method][path] = handler
}

func (a *basicAdapter) HandleRequest(req *Request) (*Response, error) {
    a.mu.RLock()
    defer a.mu.RUnlock()

    if handlers, ok := a.handlers[req.Method]; ok {
        if handler, ok := handlers[req.Path]; ok {
            return handler(req)
        }
    }
    return &Response{
        StatusCode: 404,
        Body:      []byte(`{"error": "Not found"}`),
    }, nil
}

func (a *basicAdapter) ConnectTo(_ context.Context, _ string) error {
    // Stub implementation
    return nil
}

func (a *basicAdapter) Peers() []string {
    // Stub implementation
    return []string{}
}

func (a *basicAdapter) SendMessage(_ context.Context, _ string, _ *Request) (*Response, error) {
    // Stub implementation
    return nil, fmt.Errorf("not implemented")
}

func (a *basicAdapter) StartDiscovery() error {
    // Stub implementation
    return nil
}

func (a *basicAdapter) NotifyPeer(peerID string, action string, data map[string]string) error {
    // Stub implementation
    return nil
}

func (r *Request) UnmarshalJSON(v interface{}) error {
    return json.Unmarshal(r.Body, v)
}

func MarshalJSON(v interface{}) ([]byte, error) {
    return json.Marshal(v)
}

func (r *Request) PathParam(name string) string {
    // Stub implementation
    return ""
}

func (r *Request) QueryParam(name string) string {
    // Stub implementation
    return ""
}
