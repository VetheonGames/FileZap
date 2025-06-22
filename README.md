# 🔒 FileZap

> **About the Developer:**  
> Hi! I’m a new dad with a full-time job, so I mostly work on FileZap during weekends (when life allows). Please be patient with replies on IRC, Telegram, or GitHub Issues — I appreciate your understanding!

---

## 📚 What is FileZap?

**FileZap** is a decentralized, cryptographically secure file-splitting and sharing system.  
It encrypts, chunks, and distributes files redundantly across a private peer-to-peer network — ensuring that files are never lost due to offline peers and are unintelligible without proper authorization.

📖 **For more context and technical deep-dives, see the [Developer Journal](https://github.com/VetheonGames/FileZap/blob/main/Developer%20Journal.md).**

---

## 🚀 Why FileZap?

In a world of centralization, surveillance, and unreliable file hosting, FileZap aims to be:

✅ **Upload once, lose never:**  
Chunks are stored with intentional redundancy to survive peer churn.

✅ **Privacy by design:**  
Files are encrypted at the chunk level — no single peer ever holds the whole file.

✅ **Trustless sharing:**  
A zero-trust model ensures that reconstruction is only possible with valid manifests and quorum metadata.

💡 Think of it like torrenting, but with no leechers, no lost seeds, and no file decay.

---

## ⚙️ How It Works (High-Level)

### 1️⃣ **Upload Phase**

- The user splits a file into encrypted chunks and generates a `.zap` manifest.
- The manifest contains chunk hashes, redundancy rules, encryption keys, and metadata for reassembly.
- The manifest is replicated across multiple peers via the quorum to prevent loss.

---

### 2️⃣ **Distribution Phase**

- Chunks are sent to peers over encrypted QUIC connections with NAT traversal.
- Redundant chunks are generated via replication and erasure coding.
- Any client can opt in as a **Storer**, contributing storage space:
  - Storers define a storage directory and quota.
  - Storers never hold an entire file — only a fraction of encrypted chunks, decided automatically by the quorum.
  - Files are encrypted *before* distribution, so Storers can’t inspect their chunks.
- If malicious content (e.g., malware, CSAM) is found, it can be reported:
  - The quorum votes on its removal.
  - If confirmed malicious, Storers purge the chunks automatically.

---

### 3️⃣ **Swarm Coordination**

- FileZap has no master nodes — swarm health and chunk availability are governed by a decentralized **peer quorum**.
- If a Storer goes offline, the quorum reassigns redundant peers to cover missing chunks.
- Only one Storer actively distributes each chunk at a time — chosen based on reputation, speed, and uptime.
- Redundant Storers hold backup copies and activate on demand, keeping bandwidth use efficient.

---

### 4️⃣ **Retrieval Phase**

- Users can request a file by submitting its `.zap` manifest to the quorum or via the planned Manifest Explorer (coming soon).
- Chunks are fetched from the swarm and verified for integrity.

---

### 5️⃣ **Decryption Phase**

- Once all chunks are downloaded, the client decrypts and reconstructs the original file using the manifest.

---

## 🧩 Project Components

| Component | Description |
|-----------|--------------|
| **Client GUI** | Built with Fyne; handles file splitting, joining, and network interaction |
| **Divider** | Encrypts and chunks files; generates `.zap` manifests |
| **Reconstructor** | Rebuilds files from chunks with integrity checks |
| **Network Core** | Peer-to-peer transport over QUIC with UDP hole punching |
| **VPN Layer** | Creates a private network for peers to discover each other securely, without pre-known bootstrap nodes — a core FileZap concept |

---

## 🔐 Key Features

- End-to-end AES encryption  
- Chunked storage with automatic redundancy  
- Zero-trust, decentralized storage model  
- Cross-platform GUI

---

## ⚙️ Building FileZap

### Requirements

- **Go** 1.21+
- **Fyne** UI dependencies
- *(Windows only)* MinGW-w64, GCC, and `CGO_ENABLED=1`

### ⚠️ Build Scripts (Deprecated)

| OS | Script |
|----|--------|
| ~~Windows~~ | ~~`build.ps1`~~ |
| ~~Linux~~ | ~~`build.sh`~~ |
| ~~Mac~~ | ~~`build.sh`~~ |

---

## 🤝 How to Contribute

**We welcome help from:**

- 🧠 Protocol designers & systems thinkers  
- 🌍 P2P, NAT traversal, and transport engineers  
- 🧰 Go developers (especially Fyne, QUIC, libp2p, STUN/ICE experience)  
- 🧪 Testers and file chaos agents

**Start here:**

1. Check open issues  
2. Read `CONTRIBUTING.md`  
3. Message me directly on [Telegram](https://t.me/Vetheon)

---

## 📄 License

Licensed under **GPLv3** — see `LICENSE` for details.

---

## 🚧 Project Status

> ⚠️ **Pre-Alpha Warning:**  
> FileZap is in an early **pre-alpha** stage. Chunking logic and the client are functional; the networking stack is under active development.
> NAT punching and QUIC peer transport are being prototyped. Crypto incentives ***MAY*** be optional plugins later — for now, it’s pure distributed storage.  
> The GUI is rough and integration is incomplete. Chunk retrieval and robust swarm logic are not yet functional. **FileZap should not be used for real file distribution at this time.**

---

✨ **Stay tuned — and stay secure!**
