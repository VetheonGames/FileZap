package overlay

import (
    "context"
    "encoding/json"
    "fmt"
)

// NetworkAdapter wraps the overlay network for use by other components
type NetworkAdapter struct {
    node    *Node
    ctx     context.Context
    msgChan chan *Message
}

// MessageType constants
const (
    MsgTypeValidatorRequest  = "validator_request"
    MsgTypeValidatorResponse = "validator_response"
)

// Request represents a network request
type Request struct {
    Method  string          `json:"method"`
    Path    string          `json:"path"`
    Body    json.RawMessage `json:"body"`
    pattern string          // internal field for routing
}

// Response represents a network response
type Response struct {
    StatusCode int             `json:"status_code"`
    Body       json.RawMessage `json:"body"`
}

// NewNetworkAdapter creates a new network adapter
func NewNetworkAdapter(ctx context.Context) (*NetworkAdapter, error) {
    node, err := NewNode(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to create overlay node: %v", err)
    }

    adapter := &NetworkAdapter{
        node:    node,
        ctx:     ctx,
        msgChan: make(chan *Message, 100),
    }

    // Set up message handler
    node.SetMessageHandler(adapter)

    return adapter, nil
}

// SendRequest sends a request to a peer
func (a *NetworkAdapter) SendRequest(peerID string, method string, path string, body interface{}) (*Response, error) {
    // Marshal request
    reqBody, err := json.Marshal(body)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request body: %v", err)
    }

    req := Request{
        Method: method,
        Path:   path,
        Body:   reqBody,
    }

    reqData, err := json.Marshal(req)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %v", err)
    }

    // Send message and wait for response
    if err := a.node.SendMessage(peerID, MsgTypeValidatorRequest, reqData); err != nil {
        return nil, fmt.Errorf("failed to send message: %v", err)
    }

    // Wait for response
    select {
    case msg := <-a.msgChan:
        if msg.Type != MsgTypeValidatorResponse {
            return nil, fmt.Errorf("unexpected message type: %s", msg.Type)
        }

        var resp Response
        if err := json.Unmarshal(msg.Payload, &resp); err != nil {
            return nil, fmt.Errorf("failed to unmarshal response: %v", err)
        }

        return &resp, nil

    case <-a.ctx.Done():
        return nil, fmt.Errorf("context cancelled")
    }
}

// HandleMessage implements MessageHandler
func (a *NetworkAdapter) HandleMessage(msg *Message) error {
    // For requests, process and send response
    if msg.Type == MsgTypeValidatorRequest {
        var req Request
        if err := json.Unmarshal(msg.Payload, &req); err != nil {
            return fmt.Errorf("failed to unmarshal request: %v", err)
        }

        // Process request (to be implemented by validator server)
        resp := &Response{
            StatusCode: 200,
            Body:       []byte(`{"status":"ok"}`),
        }

        respData, err := json.Marshal(resp)
        if err != nil {
            return fmt.Errorf("failed to marshal response: %v", err)
        }

        // Send response
        if err := a.node.SendMessage(msg.FromID, MsgTypeValidatorResponse, respData); err != nil {
            return fmt.Errorf("failed to send response: %v", err)
        }

        return nil
    }

    // For responses, forward to channel
    a.msgChan <- msg
    return nil
}

// Close closes the network adapter
func (a *NetworkAdapter) Close() error {
    close(a.msgChan)
    return a.node.Close()
}

// GetNodeID returns the overlay node ID
func (a *NetworkAdapter) GetNodeID() string {
    return a.node.nodeID
}
