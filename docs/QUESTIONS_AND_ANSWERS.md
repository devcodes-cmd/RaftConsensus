# Questions and Answers

Every check-yourself question asked during the build, with its answer.
Try answering each one before reading the answer. They're grouped by phase
and step, and the same questions come up in interviews.

---

## Phase 0 — Content-defined chunking

### Concept (before any code)

**Q1. In fixed-size chunking, why does inserting a byte at the end of a
file cause no damage, but inserting at the start breaks everything?**

Fixed-size boundaries sit at absolute positions (4, 8, 12, ...) counted from
the start of the file. A byte appended at the end comes after every existing
boundary, so all earlier chunks are unchanged. A byte inserted at the start
shifts every later byte by one position, so every chunk's contents change and
none match the original.

**Q2. With k = 13, why is the average chunk about 8 KiB? What if k = 10?**

If the hash behaves like a random number, each position has a 1 in 2^13 =
1 in 8192 chance that its k selected bits are all zero. On average there is
one boundary every 8192 bytes, which is 8 KiB. With k = 10 that becomes
1 in 1024, so chunks average about 1 KiB. You get 8× as many chunks: finer
deduplication, but 8× more metadata.

*Caveat:* the minimum chunk size shifts the real average upward. No boundary
is checked until `MinSize` bytes have been read, so the expected size is
about `MinSize + 2^k`. See step 0b.

**Q3. Why do we need a maximum chunk size?**

Some content never triggers the boundary rule. In a file of all zero bytes,
every window is identical, so every position gives the same hash. If that
hash isn't a boundary, no position ever is, and the whole file becomes one
giant chunk. A maximum size forces a cut, which bounds memory use and keeps
deduplication working.

### Step 0a — Rolling hash (`internal/chunker/rolling.go`)

**Q4. Why must `defaultBase` be odd?**

Multiplying by an even number shifts in a zero at the bottom, just as
multiplying by 10 appends a zero in decimal. With an even base, each step
pushes every earlier byte's contribution one bit higher, and after 64 steps
it falls off the top of the `uint64` completely. Two bad effects follow:
- The lowest bit of the hash depends only on the newest byte, the next bit
  on the newest two bytes, and so on. The low bits are badly mixed.
- For windows longer than 64 bytes, the oldest bytes have no effect at all.

An odd base has an inverse mod 2^64, so multiplying by it never throws
information away.

**Q5. Before the window is full, `Roll` skips the subtraction. Why is that
still correct?**

Nothing has left the window yet, so there's nothing to remove. The hash of a
partly filled window is the polynomial hash of all bytes seen so far. That is
exactly the same as the hash of a full window padded with leading zero bytes,
because a zero byte contributes `0 · base^i = 0`. The buffer slots start at
zero too, so doing the subtraction anyway would subtract 0 and give the same
result. The `full` flag just makes this explicit.

### Step 0b — Chunker (`internal/chunker/chunker.go`)

**Q6. 200 KiB of zero bytes produced 100 chunks of exactly 2048 bytes
(MinSize), not chunks of MaxSize. Why?**

The hash of 48 zero bytes is `0·base^47 + ... + 0 = 0`. Shifting 0 right
gives 0, so the boundary test passes at every position. The only thing
stopping a cut is `MinSize`, so every chunk is cut as soon as it's allowed.
A run of some other repeated byte `v` hashes to a fixed non-zero constant.
If that constant fails the test, the run is cut at `MaxSize` instead.

**Q7. The measured average chunk was 9903 bytes, not 8192. Why?**

No boundary is checked during the first `MinSize` (2048) bytes of a chunk.
After that, the chance of a cut at each byte is 1/8192, so on average another
8192 bytes pass before one happens. The expected size is 2048 + 8192 = 10240
bytes. 9903 is close; it's slightly lower because `MaxSize` caps the rare
very long chunks.

**Q8. After inserting 'X' at the start, why did 99 of 100 chunks stay
identical? Why does the chunker "resynchronize" after the first chunk?**

Whether a position is a boundary depends only on the 48 bytes ending there.
Inserting 'X' moves every original byte one position right, but the *content*
of each 48-byte window is unchanged, so each original boundary reappears one
position later. From the first boundary onward, the chunks are identical.
Only the first chunk contains 'X', so only it differs.

**Q9. Why is `Chunker` built on `io.Reader` instead of taking a `[]byte`?**

`io.Reader` means "anything you can read bytes from": a file, a network
socket, or an in-memory buffer (as in the tests, via `bytes.NewReader`). The
chunker only ever holds one chunk (at most `MaxSize` bytes) in memory, so it
can process a 100 GB VM image with 64 KiB of buffer. A `[]byte` API would
need the whole file loaded into memory first.

**Q10. If the boundary test used the low bits (`h & 0x1FFF == 0`), which
bytes would the lowest bit of the hash depend on?**

Only on the lowest bit of each byte in the window. Mod 2, every power of an
odd base is 1, so `hash mod 2 = (b0 + b1 + ... + b47) mod 2`. That is the
XOR of the bytes' lowest bits. More generally, bit *j* of the hash depends
only on bits 0..*j* of the bytes, because carries in addition and
multiplication only flow upward. Example: if every byte is even, bit 0 is
always 0, so one of the 13 "random" bits is stuck and boundaries become
twice as likely. The top bits receive carries from every bit of every byte,
which is why D0.3 tests the top bits.

**Q11. Why does the rolling hash never need resetting between chunks? What
would change if `MinSize` were smaller than the 48-byte window?**

Once the window is full, the hash depends only on the last 48 bytes, because
everything older has been subtracted out. The first boundary test in a chunk
happens at `MinSize` (2048) bytes, and by then the window contains only
bytes of the current chunk. A reset would give exactly the same value, so
it would be wasted work.

If `MinSize` were below 48, the two choices would diverge:
- *No reset:* the window reaches back into the previous chunk. The boundary
  decision is still purely "the 48 bytes ending here".
- *Reset:* the hash covers fewer bytes, so the decision depends on where the
  *previous* cut happened.

No reset is the correct choice either way. The resync property (Q8) relies
on boundaries depending only on local content, not on cut history.

### Step 0c — Chunk store (`internal/store/store.go`)

**Q12. Two different users back up the same 10 KiB chunk. How many copies
end up on disk, and how does `Put` know?**

One copy. Both users' `Put` computes the same SHA-256 and therefore the same
file path. The second `Put` calls `os.Stat` on that path, finds the file
already there, and returns `stored = false` without writing anything. No
lookup table is needed: the filesystem path *is* the index.

**Q13. Someone flips one bit inside a chunk file on disk. What does `Get`
return, and why can't it return wrong bytes silently?**

`Get` re-hashes the bytes it read and compares the result with the name it
was asked for. A single flipped bit changes the SHA-256 completely (the
*avalanche effect*), so the comparison fails and `Get` returns `ErrCorrupt`.
To get wrong bytes past this check, someone would need different bytes with
the same SHA-256, which nobody knows how to produce.

**Q14. Why can a crash during `Put` leave a chunk permanently broken, even
though `Get` detects the damage?** (See risk R0.1 in DESIGN.md.)

`os.WriteFile` creates the file under its final name *before* all the bytes
are written. A crash in between leaves a truncated file at the correct path.
Every later `Put` of that chunk sees "file exists" and skips the write, so
the good bytes are never written again. Detection works, but repair never
happens. The standard fix is to write to a temporary name, fsync, then
rename it into place. Rename is atomic, so the final name only ever points
at a complete file.
