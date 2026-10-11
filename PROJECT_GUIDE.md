# Distributed Backup Engine with Raft Consensus — Build & Learn Guide

**Goal:** Build a real, defensible distributed systems project — a Raft-consensus
cluster managing a content-addressable dedup/backup engine, with a live dashboard
— while actually learning Go, Raft, and systems-testing along the way. Not a
tutorial clone. Not copy-pasted code you can't explain.

This file is meant to live at the root of your repo and be fed to Cursor as
context. See "How to use this file with Cursor" below.

---

## 0. How to use this file with Cursor

1. Create your repo, put this file at the root as `PROJECT_GUIDE.md`.
2. Copy the block in **Section 1 (Cursor Rules)** into a `.cursorrules` file at
   your repo root (or `.cursor/rules/learning.mdc` if your Cursor version uses
   the newer Project Rules format — either works, pick whichever your version
   picks up).
3. When starting a session, open Cursor Chat and say something like: *"Read
   PROJECT_GUIDE.md, we're starting Phase 0. Walk me through the concept
   first before writing any code."*
4. Work phase by phase, in order. Don't let Cursor jump ahead — the phases are
   sequenced so each one teaches you something the next one depends on.
5. Commit at the end of every phase with a message referencing it (e.g. `git commit -m "Phase 0: content-defined chunker + local store"`)
   so your git history itself becomes a visible record of incremental,
   understood progress — this is something a tech lead may actually look at.

---

## 1. Cursor Rules (paste into `.cursorrules`)

<!-- ```
# Role
You are a patient distributed-systems mentor pair-programming with someone
new to Go and distributed systems. Prioritize my understanding along with speed.
This is a learning project — I need to be able to defend every design
decision and every line of nontrivial code in an interview.

# Rules
1. Before implementing any new concept, explain it first in plain language
   with a concrete example. Wait for me to confirm I understand before
   writing code.
2. Never write more than ~50-80 lines of new logic without pausing to
   explain what you just wrote and why. 
3. After implementing a nontrivial piece (a function, a state transition,
   a test), ask me 1-2 questions to check my understanding. Don't just say
   "let me know if you have questions" — actually quiz me.
4. When you make a design decision (e.g. "I'll use a mutex here" or "I'll
   model this as a separate goroutine"), explicitly name the alternative
   you rejected and why.
5. Do not silently add error handling, retries, or edge-case handling I
   didn't ask about — flag it and explain why it's needed first.
6. When I ask "why does this work" or "what happens if X", answer
   conceptually first, then point to the exact lines of code.
7. Prefer explicit, readable code over clever one-liners — this is a
   learning project, not a code-golf exercise.
8. When introducing a new Go idiom (channels, goroutines, interfaces,
   context, select) for the first time, briefly explain the idiom itself,
   not just what this specific usage does.
9. At the end of each phase, generate a short quiz (3-5 questions) from
   that phase's "Be ready to explain" list in PROJECT_GUIDE.md, and flag it
   if I try to move on without being able to answer them.
10. Never fabricate benchmark numbers, test results, or claims. Actually
    run the benchmark/test and show me real output before either of us
    writes a number into README.md, DESIGN.md, or my resume.
``` -->

---

## 2. Prerequisites (skip what you already know)

You said you're new to this — spend real time here before Phase 0, it pays
for itself later.

- **Go basics** (if new to Go): [A Tour of Go](https://go.dev/tour/) (~1 day),
  then [Effective Go](https://go.dev/doc/effective_go) (skim, ~half day).
- **Concurrency primitives**: goroutines, channels, `select`, `sync.Mutex`,
  `context.Context`. Go by Example's concurrency section is enough to start —
  you'll deepen this naturally while debugging Raft races.
- **Raft, conceptually**: watch [The Secret Lives of Data — Raft](http://thesecretlivesofdata.com/raft/)
  (interactive visualization, ~30 min) *before* reading the paper. It'll make
  the paper click much faster.
- **Raft, properly**: read *"In Search of an Understandable Consensus
  Algorithm (Extended Version)"* by Ongaro & Ousterhout. Read it twice —
  once for the overview, once slowly for Figure 2 (the state machine spec),
  which you'll basically be implementing line by line.
- **Optional but excellent for test-design ideas**: skim MIT's 6.5840
  (formerly 6.824) Raft lab writeup. Do **not** copy its code or API
  structure directly into your repo — build your own design — but its test
  cases (network partition scenarios, leader-kill scenarios) are genuinely
  excellent references for what a rigorous Raft test suite should cover.

Ask Cursor to quiz you on the Raft paper's Figure 2 before writing any Raft
code — if you can't explain what each RPC does and why each field exists,
you're not ready for Phase 2 yet.

---

## 3. Repo structure (set this up in Phase 0, grow into it)

```
distributed-backup-raft/
├── cmd/
│   ├── node/              # main entrypoint for a cluster node
│   └── client/            # CLI client to trigger backup/restore
├── internal/
│   ├── chunker/           # Phase 0: content-defined chunking (Rabin)
│   ├── store/             # Phase 0/1: content-addressable chunk store + WAL
│   ├── merkle/            # Phase 0: Merkle tree integrity
│   ├── raft/              # Phase 2: consensus implementation
│   ├── network/           # Phase 2: simulated network for fault injection
│   ├── catalog/           # Phase 3: metadata replicated via Raft
│   ├── coordinator/       # Phase 3: job scheduling/leasing
│   └── api/               # Phase 3/5: HTTP/gRPC + WebSocket streaming
├── web/                   # Phase 5: dashboard frontend
│   ├── src/
│   └── package.json
├── test/
│   ├── faultinjection/    # Phase 4: system-level chaos tests
│   └── linearizability/   # Porcupine harness
├── docs/
│   └── DESIGN.md          # design doc — tradeoffs, rejected alternatives
├── go.mod
└── README.md
```

Create `docs/DESIGN.md` in Phase 0 and add to it after every phase — a
running log of decisions made and alternatives rejected. This document,
more than the code, is what makes the project defensible in an interview.

---

## 4. Phase-by-phase plan

### Phase 0 — Content-defined chunking + local dedup store 

**Learning objectives:** rolling hashes, content-addressable storage, why
fixed-size chunking fails for dedup.

**Cursor starter prompt:**
> "Explain content-defined chunking vs fixed-size chunking using a concrete
> byte example — show me why inserting one byte at the start of a file
> breaks every fixed-size chunk boundary but not a rolling-hash one. Then
> let's implement a Rabin fingerprint chunker in Go together, one piece at
> a time."

**Build:**
- Rabin-fingerprint rolling hash chunker (`internal/chunker`)
- Content-addressable chunk store keyed by SHA-256 of chunk content
  (`internal/store`)
- Merkle tree over a snapshot's chunk list (`internal/merkle`)
- CLI to chunk a file, store it, and reconstruct it losslessly

**Definition of done:**
- Chunking + reassembly round-trips real files (test with VM disk images or
  DB dumps, not synthetic zero-filled files) with byte-perfect restore.
- Dedup ratio measured and logged on a real dataset (e.g. take 2-3 similar
  VM snapshots, measure how much is shared).
- Unit tests cover: empty file, single-byte file, file with no chunk
  boundaries at all, file where every byte is identical.

**Be ready to explain:**
- Why does a rolling hash let you find chunk boundaries without re-scanning
  the whole file on every shift?
- Why SHA-256 for content addressing and not something faster like a CRC?
- What's your Merkle tree actually protecting you against that a single
  whole-file checksum wouldn't?

**Common pitfall:** picking a naive average chunk size without reasoning
about the tradeoff (smaller chunks = better dedup, more metadata overhead).
Pick a size, justify it in DESIGN.md, and say so out loud to Cursor.

---

### Phase 1 — Crash-consistent catalog (WAL) 

**Learning objectives:** write-ahead logging, fsync semantics, crash
recovery.

**Cursor starter prompt:**
> "Explain why an application-level WAL is needed even though the
> filesystem itself has journaling — what specifically can still go wrong?
> Then let's design a WAL format for my backup catalog together before
> writing code."

**Build:**
- Append-only WAL for catalog operations (new snapshot, chunk manifest
  written, snapshot finalized)
- Recovery logic that replays the WAL on startup
- A fault-injection test harness that kills the process at a random byte
  offset mid-write and asserts recovery never loses a *committed* (fsynced)
  entry and never exposes a torn write

**Definition of done:**
- 100+ randomized kill-and-recover test runs pass with zero corruption and
  zero silently-lost committed entries.
- You can state your actual durability guarantee in one sentence (e.g. "any
  write acknowledged to the caller survives a crash; unacknowledged writes
  may be lost").

**Be ready to explain:**
- What's the difference between `write()` returning and `fsync()`
  returning, and why does it matter here?
- How do you detect a torn write (a write that was interrupted mid-record)
  on recovery, and what do you do with it?

**Common pitfall:** testing crash recovery by just calling your own "close"
function instead of actually killing the process (`os.Exit`, SIGKILL, or a
process-level kill in your test harness). A graceful shutdown test proves
nothing about crash safety.

---

### Phase 2 — Raft from scratch (the long pole)

**Learning objectives:** leader election, log replication, safety
properties, and — the actually hard part — testing distributed systems
correctly.

**Cursor starter prompt:**
> "We're implementing Raft leader election first, nothing else. Walk me
> through Figure 2 of the Raft paper's rules for RequestVote before writing
> any code. Then implement just election — no log replication yet — with
> a simulated in-memory network so we can control message delivery in
> tests."

**Build, in this sub-order (don't skip ahead):**
1. Simulated network layer (`internal/network`) — lets tests inject
   partitions, delay, message drops, and reordering deterministically.
   Build this *before* Raft itself; you can't test Raft properly without it.
2. Leader election only (RequestVote RPC, terms, timeouts). Test: a cluster
   always converges on exactly one leader per term, even with randomized
   timeouts and simulated delay.
3. Log replication (AppendEntries RPC, commit index advancement). Test:
   entries committed on a majority are never lost, even under repeated
   leader failures.
4. Snapshotting (compact the Raft log itself — this is a different
   snapshot than your backup snapshots; name things carefully so you don't
   confuse yourself, e.g. `RaftSnapshot` vs `BackupSnapshot`).
5. Dynamic membership changes (adding/removing nodes safely).
6. Linearizability verification: wire up the
   [Porcupine](https://github.com/anishathalye/porcupine) checker against
   your test harness's operation history. Run it under injected partitions,
   not just steady state.

**Definition of done:**
- 200+ randomized test runs with partitions, leader kills, and reordering,
  all passing under `go test -race`.
- Porcupine reports linearizable histories across all of the above.
- Real latency/throughput numbers logged under (a) steady state, (b) during
  a partition, (c) during leader failover — not just happy-path numbers.

**Be ready to explain:**
- What happens if a network partition occurs during a membership change —
  how does your implementation avoid split-brain (two leaders in two
  terms simultaneously committing conflicting entries)?
- How do you compact the log without losing entries that are committed but
  not yet reflected in the latest snapshot?
- Why is your commit index safe to advance even while a leader change is
  concurrently in progress? Walk through the actual invariant.

**Common pitfall:** testing with real `time.Sleep`-based timing and real
goroutines talking over real sockets from the start. This makes tests slow
and flaky and hides bugs. Build the simulated network first so your tests
are deterministic and fast — this is the single biggest quality
differentiator between a toy Raft and a real one.

---

### Phase 3 — Wire the catalog and data plane together 

**Learning objectives:** system integration, job coordination, fencing.

**Cursor starter prompt:**
> "Now let's route catalog operations — new snapshot, chunk manifest
> commit, snapshot finalize — through the Raft log we built in Phase 2,
> while keeping bulk chunk data replication on a separate, simpler quorum
> path. Explain why we're splitting these two paths before we write any
> integration code."

**Build:**
- Catalog operations become Raft log entries; catalog state is a state
  machine applied from the committed log (this is the standard Raft
  "replicated state machine" pattern — make sure you can explain it, not
  just implement it)
- Chunk data replication via a simpler quorum-write path (not through Raft
  — justify this split in DESIGN.md)
- Job leasing: when the leader assigns a backup job to a node, use a
  fencing token so a stale/zombie leader can't cause double execution
- `backup(source) -> snapshot_id`, `restore(snapshot_id, target)`,
  `list_snapshots()` API, backed by the above

**Definition of done:**
- Killing the leader mid-backup-job results in exactly one node completing
  the job, verified with a test that would fail if double-execution
  occurred.
- Catalog state is identical across all surviving nodes after any single
  node failure.

**Be ready to explain:**
- Why is a fencing token necessary here — what specifically goes wrong
  without one?
- Why does the catalog go through Raft but bulk chunk data doesn't? What
  would break if you did it the other way around?

---

### Phase 4 — System-level fault injection & benchmarking 

**Learning objectives:** chaos testing methodology, honest benchmarking.

**Cursor starter prompt:**
> "Let's build a chaos test suite that kills random nodes during backup,
> during restore, during leader election, and during job handoff, in
> randomized combinations. What invariants should we assert after each
> scenario?"

**Build:**
- `test/faultinjection` suite covering: node death during backup, during
  restore, during election, during job handoff, and partition-then-heal
  scenarios
- A benchmark harness reporting throughput/latency in steady state, during
  partition, and during leader failover (use Go's built-in benchmarking,
  `testing.B`, plus your own scenario harness for the failure-mode numbers)

**Definition of done:**
- Every number that ends up in your README or resume was actually printed
  by a test run you can re-run and reproduce.
- At least one chaos scenario you initially expected to pass but that
  actually caught a real bug — document it in DESIGN.md. (This happens to
  everyone doing this properly; if it doesn't happen to you, your fault
  injection probably isn't aggressive enough.)

---

### Phase 5 — Dashboard (build right after Phase 2, useful as a debugging tool too) 

**Learning objectives:** exposing internal state safely, basic real-time
frontend.

**Cursor starter prompt:**
> "Let's add a WebSocket endpoint that streams cluster state changes —
> current leader, term, node roles — so we can build a live dashboard.
> Keep this endpoint read-only and separate from the client API."

**Build:**
- `internal/api`: HTTP endpoints for `backup`/`restore`/`list_snapshots`
  (you likely already have the logic from Phase 3, this just exposes it)
  plus a WebSocket/SSE endpoint streaming cluster state
- `web/`: a small React (or plain JS + SVG) single-page app showing:
  1. Live cluster topology graph (leader highlighted, term number, node
     roles) — build this first, it's genuinely useful for debugging Raft
     bugs visually, not just for demos
  2. Fault injection buttons (kill node, partition, heal)
  3. Backup/restore trigger + live chunking/dedup progress
  4. Snapshot browser table

**Definition of done:**
- You can kill the leader from the dashboard and watch a new one get
  elected within your failover-latency budget, live, without refreshing.
- Keep it visually simple — this is an observability tool, not a product.
  Resist scope creep here; the backend is what's being evaluated.

---

## 5. Progress tracker

```
- [ ] Phase 0 — chunker + local store + Merkle tree
- [ ] Phase 1 — WAL + crash recovery tests
- [ ] Phase 2a — simulated network
- [ ] Phase 2b — leader election
- [ ] Phase 2c — log replication
- [ ] Phase 2d — snapshotting
- [ ] Phase 2e — membership changes
- [ ] Phase 2f — Porcupine linearizability harness
- [ ] Phase 3 — catalog over Raft + job fencing + API
- [ ] Phase 4 — chaos suite + real benchmarks
- [ ] Phase 5 — dashboard
- [ ] DESIGN.md complete with rejected alternatives documented
- [ ] README.md with real, reproducible numbers
```

<!-- ## 6. Time budget (part-time, ~8-12 hrs/week)

| Phase | Weeks |
|---|---|
| Prerequisites | 1-2 |
| 0 — chunking/dedup | 2-3 |
| 1 — crash-consistent WAL | 2 |
| 2 — Raft from scratch | 5-7 |
| 3 — integration | 3-4 |
| 4 — fault injection + benchmarking | 2-3 |
| 5 — dashboard | 2-2.5 |
| **Total** | **~17-23 weeks** |

Full-time-equivalent hours compress this to roughly 7-10 weeks. There's no
honest shortcut through Phase 2's testing — that's where the real learning
(and the real interview-defensibility) lives. -->

## 6. Resume mapping (fill in once numbers are real, never before)

Keep this section as a scratchpad — copy verified numbers here as you get
them, then move them into your actual resume/LaTeX once confirmed.

- Dedup ratio on real data: `___`
- Leader failover latency (measured, not assumed): `___`
- Linearizability test scale (# operations, # partition scenarios): `___`
- Fault-injection runs survived with zero corruption: `___`
