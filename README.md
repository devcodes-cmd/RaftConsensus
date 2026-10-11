# Distributed Backup Engine with Raft Consensus

A learning project: a small cluster of nodes that agree on a shared backup
catalog using the **Raft** consensus algorithm, and store file data as
**deduplicated, content-addressed chunks**.

> Status: **Phase 0 (not started — concepts only).** Nothing here is
> benchmarked yet. Every number that appears in this README will come from
> a test run that can be re-run.

## What problem does this solve?

Imagine backing up 3 nearly identical 10 GB virtual machine images. Storing
them naively costs 30 GB. Most of the bytes are the same, so a smarter
system stores each unique piece once (**deduplication**), and keeps the
"which pieces make up which backup" list (the **catalog**) replicated on
several machines so that losing one machine loses nothing (**consensus**).

## How the repo is organized

| Path | What lives there |
|---|---|
| `PROJECT_GUIDE.md` | The phase-by-phase build plan (the "syllabus") |
| `docs/DESIGN.md` | The design document: architecture, decisions, rejected alternatives |
| `docs/concepts/` | Plain-language explainers, one per concept, written before the code |
| `docs/QUESTIONS_AND_ANSWERS.md` | Every check-yourself question with its answer |
| `docs/GO_FOR_JAVA_DEVELOPERS.md` | Each Go construct used in the code, next to its Java equivalent |
| `internal/...` | The Go packages (created phase by phase) |
| `cmd/...` | Runnable programs (node, client CLI) |

## Reading order for someone new

1. This README.
2. `docs/DESIGN.md` sections 1–3 (problem, goals, big-picture architecture).
3. `docs/concepts/` in numeric order. Each one matches a phase.
4. The code for that phase. Every package starts with a doc comment
   explaining what it does and why it exists.

## Progress

See the tracker in `PROJECT_GUIDE.md` section 5.
