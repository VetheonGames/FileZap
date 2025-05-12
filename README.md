# 🔒 FileZap

**FileZap** is a decentralized, cryptographically secured file-splitting and sharing system. It lets users encrypt, divide, and distribute files into chunks—then reassemble them only with the appropriate authorization and after a proof-of-payment process. The system is designed to ensure that uploaded files are **impossible to lose** and **impossible to understand** until you’ve paid for and reconstructed them.

---

## 🚀 Why FileZap?

In a world of centralized control, surveillance, and unreliable data hosting, FileZap provides:

- **Upload once, lose never**: Chunked files are redundantly stored across a peer network.  
- **Privacy by design**: Files are encrypted at the chunk level and require quorum-approved access.  
- **Trustless value exchange**: No decryption key is issued until a fee is paid and validated by the network.  

It’s like torrenting, but anonymous, cryptographically enforced, and economically incentivized.

---

## ⚙️ How It Works

1. **Upload Phase**  
   - User chunks a file and submits the '.zap' metadata file to the Validator network.  
   - Validators check whether the file is already known (via a zero-knowledge method).  
   - If it’s new, the Validators assign storage peers and issue a reward for contributing a new file.  

2. **Distribution Phase**  
   - The client sends each chunk to the assigned peers.  
   - Validators monitor and require that all chunks be stored at least once before the file is 'accepted'.  
   - No one can download the file until full replication is confirmed.  

3. **Availability Phase**  
   - Once confirmed, the Validators allow download requests.  
   - Users requesting files must submit payment to the Validator quorum.  

4. **Retrieval Phase**  
   - Validators confirm payment and return a list of peer seeders for chunk download.  
   - The client downloads the chunks and requests a decryption key.  

5. **Decryption Phase**  
   - Upon final payment validation, the Validator quorum issues a time-limited decryption key.  
   - The client decrypts and reconstructs the file using the '.zap' file and downloaded chunks.  

6. **Post-Retrieval Rule**  
   - Once decrypted, the user cannot seed the file unless they delete their local copy.  
   - This ensures that only zero-knowledge nodes seed the network.  

---

## 🧩 Project Overview

| Component          | Description |
|--------------------|-------------|
| **Client**          | GUI app built with Fyne for file splitting, joining, and network interaction |
| **Divider**         | Handles file encryption and chunking, produces '.zap' metadata |
| **Reconstructor**   | Rebuilds files from chunks after integrity and payment validation |
| **Network Core**    | Manages communication between clients, Validators, and peers |
| **Validator Node**  | Coordinates file validation, storage quorum, consensus voting, and crypto-based payments |
| **Seeder Node**     | Stores encrypted file chunks; receives payment for serving chunks to clients |

---

## 🔐 Key Features

- End-to-end file encryption (AES-based)  
- Chunked file storage with redundancy  
- Zero-trust, decentralized validation model  
- Quorum-based payment and key issuance  
- Enforced privacy and anti-leeching design  
- Cross-platform GUI interface  
- Fully scriptable build system for all OSes  

---

## ⚙️ Building It

### Requirements

- Go 1.21+  
- Fyne UI dependencies  
- (Windows only) MinGW-w64, GCC, and CGO_ENABLED=1  

### Build Scripts

| OS      | Script         |
|---------|----------------|
| Windows | 'build.ps1'    |
| Linux   | 'build.sh'     |
| Mac     | 'build.sh'     |

Example:  
```bash  
./build.sh  
```

---

## 🎮 Getting Started

1. Launch the GUI client  
2. Use the **Split File** tab:  
   - Choose file, chunk size, and output directory  
3. Use the **Join File** tab:  
   - Select '.zap' file and reconstruct  
4. Use the **Network** tab to register files or request downloads via Validators  

---

## 🤝 How to Contribute

We’re looking for:

- 🧠 Cryptographers & protocol designers  
- 🌍 Peer-to-peer & decentralized systems engineers  
- 🧰 Go developers (especially familiar with Fyne, libp2p, or blockchain)  
- 🧪 Testers and file chaos agents  

Start here:

- Check issues labeled 'help wanted'  
- Read our 'CONTRIBUTING.md'  
- Message me on [Telegram](https://t.me/Vetheon)

---

## 📄 License

[GPLv3](https://www.gnu.org/licenses/gpl-3.0.en.html#license-text)

---

## 🌐 Project Status

> FileZap is currently in **pre-alpha**. The client and chunking logic are functional, and a stubbed Validator implementation is being tested. Core crypto logic, networking, and distributed storage enforcement are being iterated on weekly. Early contributors can help shape protocol rules, crypto flows, and incentive mechanics.
