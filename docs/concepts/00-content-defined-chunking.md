# Concept 00 — Content-Defined Chunking and Deduplication

**Phase:** 0 · **Code that will implement this:** `internal/chunker`, `internal/store`

## 1. The problem: storing near-duplicate files cheaply

Deduplication means splitting a file into pieces (**chunks**), naming each
chunk by a hash of its bytes, and storing each unique chunk once. When a
second file shares a chunk, we store only a reference to it.

Everything depends on **where the chunk boundaries fall**. If two files are
mostly the same, we want them split in the same places, so their chunks
turn out byte-identical and deduplicate.

## 2. Why fixed-size chunking fails

Fixed-size chunking cuts every N bytes. Take N = 4:

```
Original:   A B C D | E F G H | I J K L
```

Now insert one byte, `X`, at the very start:

```
Modified:   X A B C | D E F G | H I J K | L
```

Every boundary moved one byte to the right, so **no chunk matches** the
original. One inserted byte destroyed all deduplication. This is called the
**boundary-shift problem**, and real files hit it constantly: one row added
near the top of a database dump shifts every byte after it.

## 3. The fix: let the content decide where to cut

Instead of cutting at fixed positions, we cut wherever the **content** looks
a certain way. Toy rule: "cut right after any `D` or `H`."

```
Original:   A B C D | E F G H | I J K L
Modified:   X A B C D | E F G H | I J K L
```

The first chunk changed (it now contains `X`), but chunks 2 and 3 are
**byte-identical to before**, so they deduplicate. The damage from an edit
stays local to the chunk where the edit happened, because the boundaries
travel with the content.

### Why the real rule uses a hash of a window, not one byte

"Cut after D" is too crude. A file full of `D`s would be cut at every byte,
and a file with no `D`s would never be cut at all. The real rule is:

> Look at the last *w* bytes (the **window**, e.g. 48 bytes). Compute a hash
> of the window. If the hash's lowest *k* bits are all zero, cut here.

A good hash looks random, so the chance a given position is a boundary is
1 / 2^k. With k = 13 that is 1/8192, so chunks average about **8 KiB**.

We also enforce a **minimum** and **maximum** chunk size:
- **Min** stops tiny chunks (each one costs metadata).
- **Max** forces a cut in content that never triggers the rule, such as a
  file of all zero bytes. That's why the test list includes "every byte
  identical" and "no natural boundaries".

## 4. The rolling hash: why this is fast

Naively, re-hashing the 48-byte window at every position costs 48 steps per
byte. A **rolling hash** updates the previous window's hash in **constant
time**: subtract the byte leaving the window, add the byte entering it.

Decimal example with a window of 3 digits and base 10, over the stream
`1 2 3 4`:

```
Window [1 2 3]  →  hash = 1·100 + 2·10 + 3 = 123

Slide one step: 1 leaves, 4 enters.
  remove outgoing:   123 − 1·100 = 23
  shift left:         23 · 10    = 230
  add incoming:      230 + 4     = 234   ✓  (same as hashing [2 3 4] directly)
```

Three arithmetic operations, no matter how big the window is. Rabin
fingerprinting is the same idea, done with polynomial arithmetic modulo an
irreducible polynomial, which gives good mixing properties.

## 5. Content addressing

Each chunk's **name** is the SHA-256 of its bytes. Consequences:
- The same bytes always get the same name, so deduplication is just
  "does this name already exist in the store?"
- Different bytes get different names. For SHA-256 this is true in
  practice: no collision has ever been found.
- A chunk can't be silently corrupted. Re-hash it on read and compare.

## 6. What Phase 0 will build, in order

| Step | Piece | Size (approx.) |
|---|---|---|
| 0a | Rolling hash (just the window math, with tests) | small |
| 0b | Chunker: uses the rolling hash plus min/avg/max to emit chunks | medium |
| 0c | Chunk store: `Put(bytes) → hash`, `Get(hash) → bytes` on disk | small |
| 0d | Merkle tree over a file's list of chunk hashes | small |
| 0e | CLI: chunk + store + restore a file, report dedup ratio | small |

## 7. Decisions we will make (and record in DESIGN.md when we make them)

- **Rolling hash algorithm:** Rabin (polynomial over GF(2)), Buzhash, or
  Gear/FastCDC. The guide suggests Rabin; we'll compare before choosing.
- **Chunk sizes (min / avg / max):** smaller chunks give better dedup but
  more metadata per byte. Proposed starting point: 2 KiB / 8 KiB / 64 KiB.
- **Content hash:** SHA-256 vs CRC32 vs BLAKE3.
- **On-disk layout:** one file per chunk vs packing many chunks into
  large files.

## 8. Check yourself before we write code

1. In the fixed-size example, why does inserting a byte at the **end** of
   the file cause no damage, but inserting at the **start** breaks
   everything?
2. With k = 13, why is the *average* chunk about 8 KiB, and what happens to
   the average if we switch to k = 10?
3. Why do we need a maximum chunk size at all?
