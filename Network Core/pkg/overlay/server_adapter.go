package overlay

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"
)

// ServerAdapter wraps the overlay network for HTTP-like server functionality
type ServerAdapter struct {
    node    *Node
    ctx     context.Context
    routes  map[string]map[string]HandlerFunc // method -> path -> handler
    msgChan chan *Message
}

// HandlerFunc handles HTTP-like requests over the overlay network
type HandlerFunc func(r *Request) (*Response, error)

// NewServerAdapter creates a new server adapter
func NewServerAdapter(ctx context.Context) (*ServerAdapter, error) {
    node, err := NewNode(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to create overlay node: %v", err)
    }

    adapter := &ServerAdapter{
        node:    node,
        ctx:     ctx,
        routes:  make(map[string]map[string]HandlerFunc),
        msgChan: make(chan *Message, 100),
    }

    // Set up message handler
    node.SetMessageHandler(adapter)

    return adapter, nil
}

// HandleFunc registers a handler for a specific method and path
func (s *ServerAdapter) HandleFunc(method string, path string, handler HandlerFunc) {
    if s.routes[method] == nil {
        s.routes[method] = make(map[string]HandlerFunc)
    }
    s.routes[method][path] = handler
}

// HandleMessage implements MessageHandler
func (s *ServerAdapter) HandleMessage(msg *Message) error {
    if msg.Type == MsgTypeValidatorRequest {
        var req Request
        if err := json.Unmarshal(msg.Payload, &req); err != nil {
            return fmt.Errorf("failed to unmarshal request: %v", err)
        }

        // Find handler
        handlers, ok := s.routes[req.Method]
        if !ok {
            return s.sendError(msg.FromID, 405, "method not allowed")
        }

        handler, pattern := s.matchRoute(handlers, req.Path)
        if handler == nil {
            return s.sendError(msg.FromID, 404, "not found")
        }

        // Update request with pattern info
        req.pattern = pattern

        // Call handler
        resp, err := handler(&req)
        if err != nil {
            return s.sendError(msg.FromID, 500, err.Error())
        }

        // Send response
        respData, err := json.Marshal(resp)
        if err != nil {
            return s.sendError(msg.FromID, 500, "failed to marshal response")
        }

        if err := s.node.SendMessage(msg.FromID, MsgTypeValidatorResponse, respData); err != nil {
            return fmt.Errorf("failed to send response: %v", err)
        }

        return nil
    }

    // Forward responses to channel
    s.msgChan <- msg
    return nil
}

func (s *ServerAdapter) sendError(peerID string, status int, message string) error {
    resp := &Response{
        StatusCode: status,
        Body:       []byte(fmt.Sprintf(`{"error":"%s"}`, message)),
    }

    respData, err := json.Marshal(resp)
    if err != nil {
        return fmt.Errorf("failed to marshal error response: %v", err)
    }

    return s.node.SendMessage(peerID, MsgTypeValidatorResponse, respData)
}

func (s *ServerAdapter) matchRoute(routes map[string]HandlerFunc, path string) (HandlerFunc, string) {
    // Try exact match first
    if handler, ok := routes[path]; ok {
        return handler, path
    }

    // Try pattern matching
    for pattern, handler := range routes {
        if isPatternMatch(pattern, path) {
            return handler, pattern
        }
    }

    return nil, ""
}

// GetNodeID returns the node's ID
func (s *ServerAdapter) GetNodeID() string {
    return s.node.nodeID
}

// Close shuts down the server adapter
func (s *ServerAdapter) Close() error {
    close(s.msgChan)
    return s.node.Close()
}
