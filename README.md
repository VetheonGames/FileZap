# 🔒 FileZap

> Note about the Developer: I am a new dad, and work full time. So I only work on coding projects on the weekend (when I have nothing else going on) please be patient with getting replies on IRC/Telegram/Issues!


FileZap is a decentralized, cryptographically secured file-splitting and sharing system. It lets users encrypt, divide, and distribute files into redundant chunks across a peer-to-peer private network. The goal is to ensure that files uploaded to the network are never lost, even if individual peers go offline, and are impossible to understand without proper authorization.

## See the Developer Journal.md file for more nuanced information about FileZap, it's purpose, and why it is being created
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

That .zap manifest is then submitted to the quorum, where it is replicated across many peers to prevent manifest loss.

The manifest describes chunk hashes, redundancy parameters, decryption keys, and peer locations among other metadata needed for file reconstruction post-download



2. **Distribution Phase**

Chunks are sent to available peers using encrypted QUIC connections with NAT traversal.

Redundant chunks are generated using replication and erasure coding then distributed across the swarm.

Each client has the ability to opt-in to being a "Storer". Storers are granted heavier weight to their votes in the Quorum, as they are directly contributing to network health and file redundancy.

Storers are able to define a storage directory as well as a storage quota, to ensure that they aren't going to get their entire local machine filled with chunks by the network.

Storers are also unable to have every chunk of a file at the same time. How many chunks a storer is allowed to have of any one given file is decided by the quorum automatically, but will never be a complete file. Since encryption happens per chunk, on the local machine creating the chunks, before the distribution phase, Storers are also unable to have any knowledge of the chunks to which they are storing.

However, should a file be deemed malicious (e.g CSAM, Viruses, etc) a downloading peer has the ability to report the file to the quorum, which will then vote on its removal from the network. Once a file has been deleted from the network, all Storers that have chunks of that file will automatically delete them, preventing network poisoning, and bad actors.



3. **Swarm Coordination**

~~Master nodes (small, easy-to-run servers) keep track of chunk distribution and swarm health.~~

~~Master nodes help reassign chunks if peers disappear and maintain availability through passive health checks.~~

Swarm Coordination is handled via decentralized governance via the peer quorum. Should a Storer go offline that holds critical chunks of a file, the Quorum will assign a redundancy peer to compensate. Redundancy peers are Storers who hold copies of chunks, but do not distribute them unless requested to by the quorum. This allows for chunk redundancy, without wasting data by having multiple Storers attempt to serve the same chunk at the same time. The Storer who is chosen to distribute a chunk during the Retrieval Phase is decided by the quorum based upon (but not limited to) Reputation, Network Speed, and total amount of time in the swarm minus offline time.



4. **Retrieval Phase**

~~Clients pull manifest files from master nodes or local storage.~~

Clients request a file from the Quorum by either uploading the .zap manifest in a retrieval request, or by using the GUI manifest explorer. The Manifest Explorer is something that will come later, as it is yet to be architected to allow for proper zero-trust and zero-knowledge while still letting users find manifests to initiate retrievals without having to first have the manifest locally.

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

**VPN** - Creates a Virtual Private Network that allows all peers to communicate directly and privately with the DHT and Overlay. It also provides the backbone for self-bootstrapping without the need for dedicated bootstrapping nodes, or any pre-known peers. A joining client is able to connect to the network, find peers, and start operations without the need for previous knowledge. This is one of the core concepts of FileZap. The primary objective of the custom VPN network is to allow peers to find each other with zero previous knowledge of each other, and without making ISPs angry about random UDP packets with seemingly no destination. Since only FileZap clients will be on the VPN, we can send announcements out into the void without garnering negative attention from local ISPs.


---

## 🔐 Key Features

End-to-end file encryption (AES-based)

Chunked file storage with automatic redundancy

Zero-trust, decentralized storage model

Cross-platform GUI interface


---

## ⚙️ Building It

Requirements

Go 1.21+

Fyne UI dependencies

(Windows only) MinGW-w64, GCC, and CGO_ENABLED=1


## Build Scripts (DEPRECATED)

### OS	Script

~~**Windows**	build.ps1~~
~~**Linux**	build.sh~~
~~**Mac**	build.sh~~

---

## 🤝 How to Contribute

We’re looking for:

**🧠 Protocol designers & systems thinkers**

**🌍 P2P, NAT traversal, and transport layer engineers**

**🧰 Go developers (especially with Fyne, QUIC, libp2p, or STUN/ICE experience)**

**🧪 Testers and file chaos agents**


### Start here:

Check issues

Read our 'CONTRIBUTING.md'

Message me on Telegram

---

📄 License

GPLv3 (See LICENSE file)


---

## 🌐 Project Status

> FileZap is currently in very very early pre-alpha. The client and chunking logic are functional, and a rough networking stack is under active development. A simplified master-server model is being tested for swarm coordination, and NAT-punching + QUIC-based peer transport is being prototyped. Incentives and crypto requirements are being removed to be reintroduced later as an optionak plugin — for now, just pure distributed file storage.



> The GUI is still in rough shape, and the logic is not yet fully integrated. Chunk storage and retrieval are not functional yet, and the master node logic is still part of the main app binary. Eventually, master nodes will be standalone. In its current state, the network is non-functional, and FileZap should not be used for real file distribution or retrieval at this time.
