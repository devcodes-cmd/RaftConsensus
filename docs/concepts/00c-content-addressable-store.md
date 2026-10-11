# Concept 00c — The Content-Addressable Chunk Store

**Phase:** 0, step c · **Code:** `internal/store/store.go`

## 1. Normal storage vs content-addressed storage

A normal key-value store lets **you** pick the key: `put("report.pdf", bytes)`.
A content-addressed store picks the key **for** you. The key *is* a hash of
the bytes:

```
Put("hello")  →  key = SHA-256("hello") = 2cf24dba5fb0a30e...
Put("hello")  →  same key; already stored, write nothing   ← deduplication
Get(2cf24d…)  →  "hello"
```

That one rule gives us three properties for free:

| Property | Why it follows |
|---|---|
| **Deduplication** | Same bytes give the same key, so "already stored?" is a single file-exists check. |
| **Integrity** | On read, re-hash the bytes. If the hash doesn't equal the key, the data is corrupt. |
| **Immutability** | Changing the bytes changes the key, so a stored chunk can never be "updated", only added. This becomes important when chunks are replicated between nodes in Phase 3: there are no conflicting versions to reconcile. |

## 2. Why SHA-256 and not CRC32?

Both turn bytes into a short fingerprint, but they are designed for
different jobs.

- **CRC32** detects *accidental* corruption, such as a few flipped bits on
  a wire. Its output is 32 bits.
- **SHA-256** is a *cryptographic* hash. Nobody knows how to find two inputs
  with the same output. Its output is 256 bits.

The problem with using CRC32 as a *name* is the **birthday paradox**. With
2^32 possible names, the chance that some two different chunks share a name
reaches 50% after only about 1.18 × √(2^32) ≈ **77,000 chunks**. At our
roughly 10 KiB average chunk size, that's about **750 MB of data**. Back up a
single 1 GB file and you'll probably get a collision. The store would then
think a new chunk is "already stored", skip it, and the restore would
silently contain the wrong bytes.

With SHA-256, the same 50% point is around 2^128 chunks, far more than all
the data on Earth. CRC is also *linear*, so someone can deliberately craft
a colliding chunk in microseconds. With SHA-256 nobody can.

| | CRC32 | SHA-256 | BLAKE3 |
|---|---|---|---|
| Output size | 32 bits | 256 bits | 256 bits |
| Safe as a content name? | **No** (collides at ~77k chunks) | Yes | Yes |
| In Go's standard library? | Yes | Yes (hardware-accelerated on ARM64 and x86) | No (third-party module) |

**Chosen:** SHA-256. **Rejected:** CRC32, which is unsafe as a name.
BLAKE3 is faster but adds a dependency, and SHA-256 is fast enough at our
scale. We can measure that claim later instead of assuming it.

## 3. On-disk layout

Each chunk is one file, named by its hash in hex, inside a sub-directory
named after the first two hex characters:

```
data/chunks/2c/2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
data/chunks/9f/9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08
```

Why the `2c/` sub-directory (called **fan-out**)? Many filesystems slow down
when one directory holds millions of entries. Two hex characters give 256
buckets, which spreads 10 million chunks to about 40,000 per directory.
Git stores its objects the same way (`.git/objects/2c/...`).

**Rejected for now:** *pack files*, where many chunks are appended into one
big file with a separate index of offsets. They're more efficient: no
per-file overhead, and fewer files for the OS to track. But they need an
index format, and its crash consistency is exactly the Phase 1 problem.
One file per chunk is simple enough to reason about today.

## 4. Check yourself

1. Two different users back up the same 10 KiB chunk. How many copies end
   up on disk, and how does `Put` know?
2. Someone flips one bit inside a chunk file on disk. What does `Get`
   return, and why can't it return wrong bytes silently?
