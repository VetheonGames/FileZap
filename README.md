# 🔒 FileZap

> Note about the Developer: I am a new dad, and work full time. So I only work on coding projects on the weekend (when I have nothing else going on) please be patient with getting replies on IRC/Telegram/Issues!



FileZap is a decentralized, cryptographically secured file-splitting and sharing system. It lets users encrypt, divide, and distribute files into redundant chunks across a peer-to-peer network. The goal is to ensure that files uploaded to the network are never lost, even if individual peers go offline, and are impossible to understand without proper authorization.


---

## 🚀 Why FileZap?

In a world of centralized control, surveillance, and unreliable data hosting, FileZap provides:

**Upload once, lose never:** Chunked files are stored with intentional redundancy to survive peer churn.

**Privacy by design:** Files are encrypted at the chunk level with no single peer having access to the full file.

**Trustless sharing:** Zero-trust model ensures that reconstruction is only possible with proper manifests and quorum metadata.


It’s like torrenting, but without leechers, missing seeds, or file decay.


---

## ⚙️ How It Works (Simplified Model)

1. **Upload Phase**

The user splits a file into encrypted chunks and creates a .zap manifest.

The manifest describes chunk hashes, redundancy parameters, decryption keys, and peer locations.



2. **Distribution Phase**

Chunks are sent to available peers using encrypted QUIC connections with NAT traversal.

Redundant chunks are generated using replication and erasure coding then distributed across the swarm.



3. **Swarm Coordination**

Master nodes (small, easy-to-run servers) keep track of chunk distribution and swarm health.

Master nodes help reassign chunks if peers disappear and maintain availability through passive health checks.



4. **Retrieval Phase**

Clients pull manifest files from master nodes or local storage.

Chunks are fetched from the active swarm and reassembled locally with integrity checks.



5. **Decryption Phase**

Once chunks are retrieved, the client decrypts and reassembles the file using the manifest.


---

## 🧩 Project Overview

### Component	Description

**Client	GUI** - app built with Fyne for file splitting, joining, and network interaction
Divider	Handles file encryption and chunking, produces .zap metadata

**Reconstructor** -	Rebuilds files from chunks after verifying integrity

**Network Core** -	Manages peer-to-peer transport using QUIC and UDP hole punching

**Master Node** - lightweight node that coordinates metadata and ensures swarm health
Seeder Node	Stores encrypted file chunks and serves them to requesting peers



---

## 🔐 Key Features

End-to-end file encryption (AES-based)

Chunked file storage with automatic redundancy

Zero-trust, decentralized storage model

Master-server-assisted swarm healing

Cross-platform GUI interface

Fully scriptable build system for all OSes



---

## ⚙️ Building It

Requirements

Go 1.21+

Fyne UI dependencies

(Windows only) MinGW-w64, GCC, and CGO_ENABLED=1


## Build Scripts (DEPRECATED)

### OS	Script

**Windows**	build.ps1
**Linux**	build.sh
**Mac**	build.sh


Example:

```bash
./build.sh
```


---

## 🎮 Getting Started

1. **Launch the GUI client**


2. **Use the Split File tab:**

Choose file, chunk size, and output directory



3. **Use the Join File tab:**

Select .zap file and reconstruct



4. **Use the Network tab to distribute or retrieve files via the peer swarm**




---

## 🤝 How to Contribute

We’re looking for:

**🧠 Protocol designers & systems thinkers**

**🌍 P2P, NAT traversal, and transport layer engineers**

**🧰 Go developers (especially with Fyne, QUIC, libp2p, or STUN/ICE experience)**

**🧪 Testers and file chaos agents**


### Start here:

Check issues labeled 'help wanted'

Read our 'CONTRIBUTING.md'

Message me on Telegram



---

📄 License

GPLv3 (See LICENSE file)


---

## 🌐 Project Status

> FileZap is currently in very very early pre-alpha. The client and chunking logic are functional, and a rough networking stack is under active development. A simplified master-server model is being tested for swarm coordination, and NAT-punching + QUIC-based peer transport is being prototyped. Incentives and crypto requirements are being removed to be reintroduced later as an optionak plugin — for now, just pure distributed file storage.



> The GUI is still in rough shape, and the logic is not yet fully integrated. Chunk storage and retrieval are not functional yet, and the master node logic is still part of the main app binary. Eventually, master nodes will be standalone. In its current state, the network is non-functional, and FileZap should not be used for real file distribution or retrieval at this time.




---

## TODO

1. **Swarm Redundancy System**

Integrate chunk replication and/or Reed–Solomon encoding

Build chunk health monitoring into master nodes

Implement swarm-wide chunk repair on peer dropoff



2. **Master Server Implementation**

Split master coordination logic into standalone binary

Add REST or gRPC API for peer coordination

Support peer health tracking and manifest registry

Build basic federation model for multi-master support



3. **Transport Enhancements**

Use QUIC for encrypted peer-to-peer chunk transfer

Implement NAT traversal via UDP hole punching

Add relay fallback for peers behind symmetric NAT



4. **File Integrity & Security**

Integrate Merkle tree-based chunk validation

Improve encryption and key handling

Add support for manifest signing and tamper detection



5. **GUI Enhancements**

Build in swarm health and file availability indicators

Integrate master server discovery and selection

Polish split/join UX and link directly to peeparameters