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
        adapter := NewMockAdapter()
        require.NotNil(t, adapter)

        nodeID := adapter.GetNodeID()
        assert.NotEmpty(t, nodeID)

        err := adapter.Close()
        assert.NoError(t, err)
    })

    t.Run("create production adapter", func(t *testing.T) {
        ctx := context.Background()
        adapter, err := NewAdapter(ctx)
        require.NoError(t, err)
        require.NotNil(t, adapter)

        err = adapter.Close()
        assert.NoError(t, err)
    })
}

func TestRequestHandling(t *testing.T) {
    adapter := NewMockAdapter()
    defer adapter.Close()

    t.Run("register and handle request", func(t *testing.T) {
        adapter.HandleFunc("GET", "/test", func(r *Request) (*Response, error) {
            return &Response{
                StatusCode: 200,
                Body:      []byte("success"),
            }, nil
        })

        req := &Request{
            Method: "GET",
            Path:   "/test",
            Body:   []byte("test data"),
        }

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
}

func TestNetworkOperations(t *testing.T) {
    ctx := context.Background()
    adapter1 := NewMockAdapter()
    adapter2 := NewMockAdapter()
    defer adapter1.Close()
    defer adapter2.Close()

    t.Run("connect peers", func(t *testing.T) {
        nodeID2 := adapter2.GetNodeID()
        err := adapter1.ConnectTo(ctx, nodeID2)
        require.NoError(t, err)

        peers := adapter1.Peers()
        assert.Contains(t, peers, nodeID2)
    })

    t.Run("message exchange", func(t *testing.T) {
        received := make(chan []byte, 1)
        adapter2.HandleFunc("POST", "/msg", func(r *Request) (*Response, error) {
            received <- r.Body
            return &Response{StatusCode: 200}, nil
        })

        nodeID2 := adapter2.GetNodeID()
        message := []byte("test message")
        resp, err := adapter1.SendMessage(ctx, nodeID2, &Request{
            Method: "POST",
            Path:   "/msg",
            Body:   message,
        })
        require.NoError(t, err)
        assert.Equal(t, 200, resp.StatusCode)

        select {
        case receivedMsg := <-received:
            assert.Equal(t, message, receivedMsg)
        case <-time.After(time.Second):
            t.Fatal("timeout waiting for message")
        }
    })

    t.Run("send to unconnected peer", func(t *testing.T) {
        adapter3 := NewMockAdapter()
        defer adapter3.Close()

        _, err := adapter1.SendMessage(ctx, adapter3.GetNodeID(), &Request{
            Method: "POST",
            Path:   "/msg",
            Body:   []byte("test"),
        })
        assert.Error(t, err)
    })
}

func TestConcurrentOperations(t *testing.T) {
    adapter := NewMockAdapter()
    defer adapter.Close()

    t.Run("concurrent message handling", func(t *testing.T) {
        adapter.HandleFunc("POST", "/concurrent", func(r *Request) (*Response, error) {
            var data map[string]string
            json.Unmarshal(r.Body, &data)
            return &Response{
                StatusCode: 200,
                Body:      []byte(fmt.Sprintf("received: %s", data["id"])),
            }, nil
        })

        const numRequests = 10
        done := make(chan bool)
        responses := make(chan *Response, numRequests)

        for i := 0; i < numRequests; i++ {
            go func(id int) {
                msg := map[string]string{"id": fmt.Sprintf("req-%d", id)}
                body, _ := json.Marshal(msg)
                resp, err := adapter.HandleRequest(&Request{
                    Method: "POST",
                    Path:   "/concurrent",
                    Body:   body,
                })
                require.NoError(t, err)
                responses <- resp
                done <- true
            }(i)
        }

        for i := 0; i < numRequests; i++ {
            <-done
        }
        close(responses)

        count := 0
        for resp := range responses {
            assert.Equal(t, 200, resp.StatusCode)
            count++
        }
        assert.Equal(t, numRequests, count)
    })
}
