package overlay

import (
"bytes"
"context"
"testing"
"time"

"github.com/libp2p/go-libp2p/core/network"
"github.com/stretchr/testify/mock"
)

// Mock stream for testing
type mockStream struct {
	network.Stream
	readBuf  *bytes.Buffer
	writeBuf *bytes.Buffer
}

func newMockStream() *mockStream {
	return &mockStream{
		readBuf:  new(bytes.Buffer),
		writeBuf: new(bytes.Buffer),
	}
}

func (m *mockStream) Read(p []byte) (n int, err error)  { return m.readBuf.Read(p) }
func (m *mockStream) Write(p []byte) (n int, err error) { return m.writeBuf.Write(p) }
func (m *mockStream) Close() error                      { return nil }

// Mock message handler
type mockMessageHandler struct {
	mock.Mock
}

func (m *mockMessageHandler) HandleMessage(msg *Message) error {
	args := m.Called(msg)
	return args.Error(0)
}

func TestMessageSerialization(t *testing.T) {
	tests := []struct {
		name    string
		msg     *Message
		wantErr bool
	}{
		{
			name: "Basic message",
			msg: &Message{
				FromID:  "sender123",
				ToID:    "receiver456",
				Type:    "test",
				Payload: []byte("test payload"),
				IsLAN:   false,
			},
			wantErr: false,
		},
		{
			name: "Empty payload",
			msg: &Message{
				FromID:  "sender123",
				ToID:    "receiver456",
				Type:    "empty",
				Payload: []byte{},
				IsLAN:   true,
			},
			wantErr: false,
		},
		{
			name: "Binary payload",
			msg: &Message{
				FromID:  "sender123",
				ToID:    "receiver456",
				Type:    "binary",
				Payload: []byte{0x00, 0x01, 0x02, 0x03},
				IsLAN:   false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream := newMockStream()

			// Write message
			err := WriteMessage(stream, tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("WriteMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Copy write buffer to read buffer for reading
			stream.readBuf.Write(stream.writeBuf.Bytes())

			// Read message
			got, err := ReadMessage(stream)
			if err != nil {
				t.Errorf("ReadMessage() error = %v", err)
				return
			}

			// Compare messages
			if got.FromID != tt.msg.FromID || got.ToID != tt.msg.ToID || got.Type != tt.msg.Type ||
				!bytes.Equal(got.Payload, tt.msg.Payload) || got.IsLAN != tt.msg.IsLAN {
				t.Errorf("ReadMessage() = %v, want %v", got, tt.msg)
			}
		})
	}
}

func TestUint64Encoding(t *testing.T) {
	tests := []struct {
		name  string
		value uint64
	}{
		{"Zero", 0},
		{"Small number", 42},
		{"Large number", 1<<32 - 1},
		{"Max uint64", ^uint64(0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)

			// Write value
			err := writeUint64(buf, tt.value)
			if err != nil {
				t.Errorf("writeUint64() error = %v", err)
				return
			}

			// Read value
			got, err := readUint64(buf)
			if err != nil {
				t.Errorf("readUint64() error = %v", err)
				return
			}

			if got != tt.value {
				t.Errorf("readUint64() = %v, want %v", got, tt.value)
			}
		})
	}
}

func TestMessageHandling(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	node, err := NewNode(ctx)
	if err != nil {
		t.Fatalf("NewNode() error = %v", err)
	}
	defer node.Close()

	handler := &mockMessageHandler{}
	node.SetMessageHandler(handler)

	testMsg := &Message{
		FromID:  "sender123",
		ToID:    node.nodeID,
		Type:    "test",
		Payload: []byte("test payload"),
		IsLAN:   false,
	}

	handler.On("HandleMessage", testMsg).Return(nil)

	stream := newMockStream()
	if err := WriteMessage(stream, testMsg); err != nil {
		t.Fatalf("WriteMessage() error = %v", err)
	}

	// Copy write buffer to read buffer for the handler
	stream.readBuf.Write(stream.writeBuf.Bytes())

	// Handle the stream
	node.handleIncomingStream(stream)

	handler.AssertExpectations(t)
}

func TestLANDiscovery(t *testing.T) {
t.Skip("LAN discovery tests need to be run in isolation due to port binding")
}

func TestMessageDelivery(t *testing.T) {
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Create two nodes
node1, err := NewNode(ctx)
if err != nil {
t.Fatalf("Failed to create node1: %v", err)
}
defer node1.Close()

node2, err := NewNode(ctx)
if err != nil {
t.Fatalf("Failed to create node2: %v", err)
}
defer node2.Close()

// Set up message handlers
handler1 := &mockMessageHandler{}
handler2 := &mockMessageHandler{}
node1.SetMessageHandler(handler1)
node2.SetMessageHandler(handler2)


// Set up expectation for handler2
handler2.On("HandleMessage", mock.MatchedBy(func(msg *Message) bool {
return msg.FromID == node1.nodeID && msg.Type == "test"
})).Return(nil)

// Store node2 as a LAN peer for node1
node1.lanPeers.Store(node2.nodeID, PeerInfo{
ID:       node2.host.ID(),
IsLAN:    true,
LastSeen: time.Now(),
})

// Send message
err = node1.SendMessage(node2.nodeID, "test", []byte("test payload"))
if err != nil {
t.Errorf("SendMessage() error = %v", err)
}

// Allow time for message processing
time.Sleep(100 * time.Millisecond)

handler2.AssertExpectations(t)


// Expect error for nonexistent peer
err = node1.SendMessage("nonexistent-peer", "overlay-test", []byte("overlay test"))
if err == nil {
t.Error("Expected error when sending to nonexistent peer")
}
}

func TestStreamErrors(t *testing.T) {
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

node, err := NewNode(ctx)
if err != nil {
t.Fatalf("NewNode() error = %v", err)
}
defer node.Close()

// Test corrupt message length
stream := newMockStream()
stream.writeBuf.Write([]byte{0xFF, 0xFF}) // Invalid length prefix

_, err = ReadMessage(stream)
if err == nil {
t.Error("Expected error when reading corrupt message")
}

// Test truncated message
stream = newMockStream()
err = writeUint64(stream, 100) // Length larger than actual data
if err != nil {
t.Fatalf("writeUint64() error = %v", err)
}
stream.readBuf.Write(stream.writeBuf.Bytes())
stream.readBuf.Write([]byte("truncated"))

_, err = ReadMessage(stream)
if err == nil {
t.Error("Expected error when reading truncated message")
}

// Test invalid JSON
stream = newMockStream()
err = writeUint64(stream, 10)
if err != nil {
t.Fatalf("writeUint64() error = %v", err)
}
stream.writeBuf.Write([]byte("invalid json"))
stream.readBuf.Write(stream.writeBuf.Bytes())

_, err = ReadMessage(stream)
if err == nil {
t.Error("Expected error when reading invalid JSON")
}
}

func TestPeerAnnouncement(t *testing.T) {
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

node, err := NewNode(ctx)
if err != nil {
t.Fatalf("NewNode() error = %v", err)
}
defer node.Close()

// Test self announcement
node.handlePeerAnnouncement([]byte(node.nodeID))
var count int
node.lanPeers.Range(func(key, value interface{}) bool {
count++
return true
})
if count > 0 {
t.Error("Self announcement should be ignored")
}

// Test valid peer announcement
peerID := "test-peer-id"
node.handlePeerAnnouncement([]byte(peerID))
var foundPeer bool
node.lanPeers.Range(func(key, value interface{}) bool {
if key.(string) == peerID {
foundPeer = true
info := value.(PeerInfo)
if !info.IsLAN {
t.Error("Peer should be marked as LAN peer")
}
}
return true
})
if !foundPeer {
t.Error("Peer announcement not properly stored")
}

// Test empty announcement
node.handlePeerAnnouncement([]byte{})
node.lanPeers.Range(func(key, value interface{}) bool {
if key.(string) == "" {
t.Error("Empty peer announcement should not be stored")
}
return true
})
}

func TestDHTBootstrap(t *testing.T) {
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

node, err := NewNode(ctx)
if err != nil {
t.Fatalf("NewNode() error = %v", err)
}
defer node.Close()

// Test bootstrap timeout
timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 1*time.Millisecond)
defer timeoutCancel()

err = node.dht.Bootstrap(timeoutCtx)
if err == nil {
t.Error("Expected bootstrap to timeout")
}

// Verify DHT is still functional after timeout
if node.dht == nil {
t.Error("DHT should still be initialized after timeout")
}
}

func TestNodeCreation(t *testing.T) {
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Create node
node, err := NewNode(ctx)
if err != nil {
t.Fatalf("NewNode() error = %v", err)
}
defer node.Close()

// Verify node ID is not empty
if node.nodeID == "" {
t.Error("Node ID should not be empty")
}

// Verify host is initialized
if node.host == nil {
t.Error("Host should be initialized")
}

// Verify DHT is initialized
if node.dht == nil {
t.Error("DHT should be initialized")
}

// Test setting message handler
handler := &mockMessageHandler{}
node.SetMessageHandler(handler)
if node.msgHandler != handler {
t.Error("Message handler not set correctly")
}
}

func TestNodeClosing(t *testing.T) {
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Create node
node, err := NewNode(ctx)
if err != nil {
t.Fatalf("NewNode() error = %v", err)
}

// Close node
if err := node.Close(); err != nil {
t.Errorf("Close() error = %v", err)
}

// Verify context is cancelled
select {
case <-node.ctx.Done():
// Context should be cancelled
default:
t.Error("Context should be cancelled after closing")
}
}
