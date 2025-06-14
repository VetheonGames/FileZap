package overlay

import (
    "context"
    "encoding/json"
    "fmt"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestOverlayAdapter(t *testing.T) {
    t.Run("create adapter", func(t *testing.T) {
        ctx := context.Background()
        adapter, err := NewAdapter(ctx)
        require.NoError(t, err)
        require.NotNil(t, adapter)

        nodeID := adapter.GetNodeID()
        assert.NotEmpty(t, nodeID)

        err = adapter.Close()
        assert.NoError(t, err)
    })

    t.Run("invalid context", func(t *testing.T) {
        ctx, cancel := context.WithCancel(context.Background())
        cancel()

        adapter, err := NewAdapter(ctx)
        assert.Error(t, err)
        assert.Nil(t, adapter)
    })
}

func TestRequestHandling(t *testing.T) {
    ctx := context.Background()
    adapter, err := NewAdapter(ctx)
    require.NoError(t, err)
    defer adapter.Close()

    t.Run("register and handle request", func(t *testing.T) {
        // Register handler
        adapter.HandleFunc("GET", "/test", func(r *Request) (*Response, error) {
            return &Response{
                StatusCode: 200,
                Body:      []byte("success"),
            }, nil
        })

        // Create test request
        req := &Request{
            Method: "GET",
            Path:   "/test",
            Body:   []byte("test data"),
        }

        // Process request through the handler
        resp, err := adapter.HandleRequest(req)
        require.NoError(t, err)
        assert.Equal(t, 200, resp.StatusCode)
        assert.Equal(t, []byte("success"), resp.Body)
    })

    t.Run("method not allowed", func(t *testing.T) {
        req := &Request{
            Method: "POST",
            Path:   "/test",
            Body:   []byte("test data"),
        }

        resp, err := adapter.HandleRequest(req)
        require.NoError(t, err)
        assert.Equal(t, 405, resp.StatusCode)
    })

    t.Run("path not found", func(t *testing.T) {
        req := &Request{
            Method: "GET",
            Path:   "/nonexistent",
            Body:   []byte("test data"),
        }

        resp, err := adapter.HandleRequest(req)
        require.NoError(t, err)
        assert.Equal(t, 404, resp.StatusCode)
    })

    t.Run("handler error", func(t *testing.T) {
        adapter.HandleFunc("GET", "/error", func(r *Request) (*Response, error) {
            return nil, fmt.Errorf("test error")
        })

        req := &Request{
            Method: "GET",
            Path:   "/error",
            Body:   []byte("test data"),
        }

        resp, err := adapter.HandleRequest(req)
        require.NoError(t, err)
        assert.Equal(t, 500, resp.StatusCode)
    })
}

func TestMessageHandling(t *testing.T) {
    ctx := context.Background()
    adapter, err := NewAdapter(ctx)
    require.NoError(t, err)
    defer adapter.Close()

    t.Run("handle valid message", func(t *testing.T) {
        // Register test handler
        adapter.HandleFunc("GET", "/test", func(r *Request) (*Response, error) {
            var data map[string]string
            err := json.Unmarshal(r.Body, &data)
            require.NoError(t, err)

            return &Response{
                StatusCode: 200,
                Body:      []byte(fmt.Sprintf("received: %s", data["message"])),
            }, nil
        })

        // Create test message
        msg := map[string]string{"message": "hello"}
        body, err := json.Marshal(msg)
        require.NoError(t, err)

        req := &Request{
            Method: "GET",
            Path:   "/test",
            Body:   body,
        }

        resp, err := adapter.HandleRequest(req)
        require.NoError(t, err)
        assert.Equal(t, 200, resp.StatusCode)
        assert.Equal(t, []byte("received: hello"), resp.Body)
    })

    t.Run("invalid message format", func(t *testing.T) {
        adapter.HandleFunc("GET", "/test", func(r *Request) (*Response, error) {
            var data map[string]string
            err := json.Unmarshal(r.Body, &data)
            if err != nil {
                return &Response{
                    StatusCode: 400,
                    Body:      []byte("invalid json"),
                }, nil
            }
            return &Response{StatusCode: 200}, nil
        })

        req := &Request{
            Method: "GET",
            Path:   "/test",
            Body:   []byte("invalid json"),
        }

        resp, err := adapter.HandleRequest(req)
        require.NoError(t, err)
        assert.Equal(t, 400, resp.StatusCode)
    })
}

func TestConcurrentRequests(t *testing.T) {
    ctx := context.Background()
    adapter, err := NewAdapter(ctx)
    require.NoError(t, err)
    defer adapter.Close()

    // Register test handler
    adapter.HandleFunc("GET", "/concurrent", func(r *Request) (*Response, error) {
        time.Sleep(10 * time.Millisecond) // Simulate work
        return &Response{
            StatusCode: 200,
            Body:      []byte("success"),
        }, nil
    })

    t.Run("concurrent requests", func(t *testing.T) {
        const numRequests = 10
        done := make(chan bool)
        responses := make(chan *Response, numRequests)

        // Send concurrent requests
        for i := 0; i < numRequests; i++ {
            go func(id int) {
                req := &Request{
                    Method: "GET",
                    Path:   "/concurrent",
                    Body:   []byte(fmt.Sprintf("request %d", id)),
                }

        resp, err := adapter.HandleRequest(req)
                require.NoError(t, err)
                responses <- resp
                done <- true
            }(i)
        }

        // Wait for all requests
        for i := 0; i < numRequests; i++ {
            <-done
        }
        close(responses)

        // Verify responses
        count := 0
        for resp := range responses {
            assert.Equal(t, 200, resp.StatusCode)
            assert.Equal(t, []byte("success"), resp.Body)
            count++
        }
        assert.Equal(t, numRequests, count)
    })
}

func TestNetworkIntegration(t *testing.T) {
    // Create two adapters to simulate network communication
    ctx := context.Background()
    
    adapter1, err := NewAdapter(ctx)
    require.NoError(t, err)
    defer adapter1.Close()

    adapter2, err := NewAdapter(ctx)
    require.NoError(t, err)
    defer adapter2.Close()

    t.Run("connect peers", func(t *testing.T) {
        // Get peer addresses
        nodeID2 := adapter2.GetNodeID()

        // Connect peers via DHT
        err := adapter1.ConnectTo(ctx, nodeID2)
        require.NoError(t, err)

        // Verify connection
        peers := adapter1.Peers()
        assert.Contains(t, peers, adapter2.GetNodeID())

        peers = adapter2.Peers()
        assert.Contains(t, peers, adapter1.GetNodeID())
    })

    t.Run("message exchange", func(t *testing.T) {
        // Set up message handler on adapter2
        received := make(chan []byte, 1)
        adapter2.HandleFunc("POST", "/message", func(r *Request) (*Response, error) {
            received <- r.Body
            return &Response{StatusCode: 200}, nil
        })

        // Send message from adapter1 to adapter2
        message := []byte("test message")
        resp, err := adapter1.SendMessage(ctx, adapter2.GetNodeID(), &Request{
            Method: "POST",
            Path:   "/message",
            Body:   message,
        })
        require.NoError(t, err)
        assert.Equal(t, 200, resp.StatusCode)

        // Verify message was received
        select {
        case receivedMsg := <-received:
            assert.Equal(t, message, receivedMsg)
        case <-time.After(time.Second):
            t.Fatal("timeout waiting for message")
        }
    })
}
