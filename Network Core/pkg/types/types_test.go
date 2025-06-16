package types

import (
"encoding/json"
"reflect"
"testing"
)

func TestPeerChunkInfoJSON(t *testing.T) {
tests := []struct {
name     string
info     PeerChunkInfo
wantJSON string
}{
{
name: "Full peer info",
info: PeerChunkInfo{
PeerID:    "peer1",
ChunkIDs:  []string{"chunk1", "chunk2"},
Address:   "127.0.0.1:8080",
Available: true,
},
wantJSON: `{"peer_id":"peer1","chunk_ids":["chunk1","chunk2"],"address":"127.0.0.1:8080","available":true}`,
},
{
name: "Empty chunk list",
info: PeerChunkInfo{
PeerID:    "peer2",
ChunkIDs:  []string{},
Address:   "127.0.0.1:8081",
Available: false,
},
wantJSON: `{"peer_id":"peer2","chunk_ids":[],"address":"127.0.0.1:8081","available":false}`,
},
{
name: "Nil chunk list",
info: PeerChunkInfo{
PeerID:    "peer3",
ChunkIDs:  nil,
Address:   "127.0.0.1:8082",
Available: true,
},
wantJSON: `{"peer_id":"peer3","chunk_ids":null,"address":"127.0.0.1:8082","available":true}`,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
// Test marshaling
got, err := json.Marshal(tt.info)
if err != nil {
t.Errorf("json.Marshal() error = %v", err)
return
}
if string(got) != tt.wantJSON {
t.Errorf("json.Marshal() = %v, want %v", string(got), tt.wantJSON)
}

// Test unmarshaling
var info PeerChunkInfo
err = json.Unmarshal([]byte(tt.wantJSON), &info)
if err != nil {
t.Errorf("json.Unmarshal() error = %v", err)
return
}
if !reflect.DeepEqual(info, tt.info) {
t.Errorf("json.Unmarshal() = %v, want %v", info, tt.info)
}
})
}
}

func TestFileInfoJSON(t *testing.T) {
tests := []struct {
name     string
info     FileInfo
wantJSON string
}{
{
name: "Complete file info",
info: FileInfo{
Name:      "test.txt",
ChunkIDs:  []string{"chunk1", "chunk2"},
Available: true,
Peers: []PeerChunkInfo{
{
PeerID:    "peer1",
ChunkIDs:  []string{"chunk1"},
Address:   "127.0.0.1:8080",
Available: true,
},
{
PeerID:    "peer2",
ChunkIDs:  []string{"chunk2"},
Address:   "127.0.0.1:8081",
Available: true,
},
},
},
wantJSON: `{"name":"test.txt","chunk_ids":["chunk1","chunk2"],"available":true,"peers":[{"peer_id":"peer1","chunk_ids":["chunk1"],"address":"127.0.0.1:8080","available":true},{"peer_id":"peer2","chunk_ids":["chunk2"],"address":"127.0.0.1:8081","available":true}]}`,
},
{
name: "Empty peers list",
info: FileInfo{
Name:      "empty.txt",
ChunkIDs:  []string{"chunk1"},
Available: false,
Peers:     []PeerChunkInfo{},
},
wantJSON: `{"name":"empty.txt","chunk_ids":["chunk1"],"available":false,"peers":[]}`,
},
{
name: "Nil values",
info: FileInfo{
Name:      "nil.txt",
ChunkIDs:  nil,
Available: false,
Peers:     nil,
},
wantJSON: `{"name":"nil.txt","chunk_ids":null,"available":false,"peers":null}`,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
// Test marshaling
got, err := json.Marshal(tt.info)
if err != nil {
t.Errorf("json.Marshal() error = %v", err)
return
}
if string(got) != tt.wantJSON {
t.Errorf("json.Marshal() = %v, want %v", string(got), tt.wantJSON)
}

// Test unmarshaling
var info FileInfo
err = json.Unmarshal([]byte(tt.wantJSON), &info)
if err != nil {
t.Errorf("json.Unmarshal() error = %v", err)
return
}
if !reflect.DeepEqual(info, tt.info) {
t.Errorf("json.Unmarshal() = %v, want %v", info, tt.info)
}
})
}
}

func TestChunkStorageConfigJSON(t *testing.T) {
tests := []struct {
name     string
config   ChunkStorageConfig
wantJSON string
}{
{
name: "Complete config",
config: ChunkStorageConfig{
StorageDir:     "/tmp/chunks",
MaxStorageSize: 1024 * 1024 * 1024, // 1GB
AutoCleanup:    true,
},
wantJSON: `{"storage_dir":"/tmp/chunks","max_storage_size":1073741824,"auto_cleanup":true}`,
},
{
name: "Empty storage dir",
config: ChunkStorageConfig{
StorageDir:     "",
MaxStorageSize: 1024 * 1024, // 1MB
AutoCleanup:    false,
},
wantJSON: `{"storage_dir":"","max_storage_size":1048576,"auto_cleanup":false}`,
},
{
name: "Zero max storage size",
config: ChunkStorageConfig{
StorageDir:     "/chunks",
MaxStorageSize: 0,
AutoCleanup:    true,
},
wantJSON: `{"storage_dir":"/chunks","max_storage_size":0,"auto_cleanup":true}`,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
// Test marshaling
got, err := json.Marshal(tt.config)
if err != nil {
t.Errorf("json.Marshal() error = %v", err)
return
}
if string(got) != tt.wantJSON {
t.Errorf("json.Marshal() = %v, want %v", string(got), tt.wantJSON)
}

// Test unmarshaling
var config ChunkStorageConfig
err = json.Unmarshal([]byte(tt.wantJSON), &config)
if err != nil {
t.Errorf("json.Unmarshal() error = %v", err)
return
}
if !reflect.DeepEqual(config, tt.config) {
t.Errorf("json.Unmarshal() = %v, want %v", config, tt.config)
}
})
}
}
