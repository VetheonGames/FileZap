# 📅 June 21st, 2025
## 🚀 First Dev Journal Entry

Hey everyone!

**FileZap** is, without a doubt, the biggest and most ambitious project I’ve ever tackled. It’s been a rough road so far, but things are finally starting to take shape.

---

## 🧩 The Bootstrapping Challenge

The biggest hurdle has been solving the *bootstrapping problem*:  
**How do peers discover each other without any prior knowledge?**

Traditional approaches like DHTs rely on known bootstrap nodes or pre-configured peer lists — which directly conflicts with one of FileZap’s core principles: **Zero-Knowledge**.

After a lot of brainstorming, I landed on a solution:  
**Roll my own VPN layer.**

### 🔑 How It Works (Current Model)

1. **Client Startup:**  
   If it’s the first launch, the client:
   - Creates a TUN interface in the OS, using the same standard config as every other client.
   - Generates a deterministic **Peer ID** from the hash of a known global public key.

2. **IP Assignment:**  
   The client is assigned an IP address from FileZap’s VPN IP space.

3. **Peer Discovery:**  
   It announces its presence over the TUN interface to find initial peers.

4. **Network Join:**  
   It uses those peers to bootstrap into the DHT via pubsub and gossip.

5. **Begin Operations:**  
   Once connected, normal operations commence.

It’s not perfect — and it’s admittedly messy — but logically, it should work. 🤞

---

## 🌍 Scalability Concerns

One concern is **IPv4 exhaustion** as the network grows. We may run out of addresses, but that’s a bridge we’ll cross when we get there.

---

## 🤬 Why I’m Building FileZap

I’ve always hated how BitTorrent handles trust and resilience. Don’t get me wrong — torrenting is user-friendly, censorship-resistant, and generally robust. But it has glaring issues:

- **Malware:**  
  You can download a file that appears legit but secretly has malware bundled inside. There’s no way to warn the swarm or block its spread.

- **No Governance:**  
  BitTorrent has no built-in way to moderate malicious or corrupted torrents. If even one peer keeps seeding, the bad file stays alive.

- **Data Loss:**  
  When seeds vanish (due to takedowns or people going offline), torrents can be lost forever — even valuable open-source software or obscure media.

---

## ✅ How FileZap Solves This

FileZap is designed to tackle these flaws with **democratic governance** and **redundancy by design**:

- **Quorum Voting:**  
  If a file is malicious, anyone can report it. Once enough votes confirm it’s bad, the quorum instructs all **Storers** to stop distributing it and to purge it from the network.

- **Zero-Knowledge Storage:**  
  To prevent targeting, Storers never know exactly what they’re storing. They only know encrypted chunk hashes. So even if someone wanted to take down specific files:
  1. They’d need to match encrypted hashes to actual content.
  2. They’d need to locate every peer storing those chunks — nearly impossible due to the private VPN layer.
  3. Even if they succeed, the quorum detects missing peers and spins up new redundancy to replace them.

In short: **Once a file is uploaded and chunked, removing it is practically impossible.**  
Taking down a single storer achieves nothing — the network heals itself automatically.

---

## ♾️ Built to Last

As long as there are enough peers with enough storage, files in FileZap are **permanent** and always retrievable — no matter how many peers churn in and out.

Think of it like torrenting, but:
- No seeds required
- No permanent file loss
- Censorship-resistant
- Impossible to fully shut down

If even *one* FileZap client survives in the wild, the entire network can rebuild from scratch.

---

## ✍️ Looking Ahead

It’s late, and I’ve made a lot of progress tonight — so I’ll wrap it up here.  
I plan to write one of these journal entries every week after big update commits, as we push toward the end of the initial dev cycle and gear up for a public alpha.

Thanks for following along — and remember: **stay secure out there!**

💙 ~ Veth
