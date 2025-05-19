package overlay

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
)

// RequestHandler is a function that handles network requests
type RequestHandler func(r *Request) (*Response, error)

// ServerAdapter adapts the overlay network for server use
type ServerAdapter struct {
    node     *OverlayNode
    handlers map[string]RequestHandler
    ctx      context.Context
    cancel   context.CancelFunc
}

// NewServerAdapter creates a new server adapter
func NewServerAdapter(ctx context.Context) (*ServerAdapter, error) {
    ctx, cancel := context.WithCancel(ctx)
    
    node, err := NewOverlayNode(ctx)
    if err != nil {
        cancel()
        return nil, fmt.Errorf("failed to create overlay node: %v", err)
    }

    adapter := &ServerAdapter{
        node:     node,
        handlers: make(map[string]RequestHandler),
        ctx:      ctx,
        cancel:   cancel,
    }

    // Set up message handler
    node.SetMessageHandler(adapter)

    return adapter, nil
}

// HandleMessage implements MessageHandler
func (s *ServerAdapter) HandleMessage(msg *Message) error {
    if msg.Type != MsgTypeValidatorRequest {
        return fmt.Errorf("unexpected message type: %s", msg.Type)
    }

    var req Request
    if err := json.Unmarshal(msg.Payload, &req); err != nil {
        return fmt.Errorf("failed to unmarshal request: %v", err)
    }

    // Find handler for path
    handler := s.findHandler(req.Method, req.Path)
    if handler == nil {
        // Send 404 response
        resp := &Response{
            StatusCode: http.StatusNotFound,
            Body:      []byte(`{"error":"not found"}`),
        }
        return s.sendResponse(msg.FromID, resp)
    }

    // Handle request
    resp, err := handler(&req)
    if err != nil {
        // Send error response
        resp = &Response{
            StatusCode: http.StatusInternalServerError,
            Body:      []byte(fmt.Sprintf(`{"error":"%s"}`, err.Error())),
        }
    }

    return s.sendResponse(msg.FromID, resp)
}

// HandleFunc registers a handler for a specific path pattern
func (s *ServerAdapter) HandleFunc(method string, pattern string, handler RequestHandler) {
    key := method + " " + pattern
    s.handlers[key] = handler
}

// GetNodeID returns the overlay node ID
func (s *ServerAdapter) GetNodeID() string {
    return s.node.nodeID
}

// Close shuts down the server adapter
func (s *ServerAdapter) Close() error {
    s.cancel()
    return s.node.Close()
}

// Internal methods

func (s *ServerAdapter) sendResponse(toID string, resp *Response) error {
    respData, err := json.Marshal(resp)
    if err != nil {
        return fmt.Errorf("failed to marshal response: %v", err)
    }

    if err := s.node.SendMessage(toID, MsgTypeValidatorResponse, respData); err != nil {
        return fmt.Errorf("failed to send response: %v", err)
    }

    return nil
}

func (s *ServerAdapter) findHandler(method string, path string) RequestHandler {
    // Check for exact match first
    if handler := s.handlers[method+" "+path]; handler != nil {
        return handler
    }

    // Check for pattern matches
    for pattern, handler := range s.handlers {
        parts := strings.Split(pattern, " ")
        if len(parts) != 2 {
            continue
        }

        if parts[0] != method {
            continue
        }

        // Convert pattern to regex
        regex := patternToRegex(parts[1])
        if regex.MatchString(path) {
            return handler
        }
    }

    return nil
}
