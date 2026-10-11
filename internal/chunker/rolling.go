// Package chunker splits a byte stream into variable-size chunks whose
// boundaries are chosen by the content itself (content-defined chunking).
// Because boundaries travel with the content, inserting or deleting bytes
// only changes the chunks near the edit, so the rest still deduplicate.
//
// Background: docs/concepts/00-content-defined-chunking.md
package chunker

// defaultBase is the multiplier of the polynomial hash. It must be odd:
// multiplying by an even number shifts in a zero bit, so with an even base
// each byte's contribution loses one low bit per step and vanishes entirely
// after 64 steps. Low hash bits would then depend only on the newest bytes.
// Odd numbers are invertible mod 2^64, so no byte's influence is ever lost.
const defaultBase uint64 = 0x100000001b3

// RollingHash is a Rabin-Karp polynomial hash over the last windowSize
// bytes pushed into it. For a window [b0 b1 ... b(w-1)] the hash is
//
//	b0*base^(w-1) + b1*base^(w-2) + ... + b(w-1)*base^0   (mod 2^64)
//
// The "mod 2^64" is free: uint64 arithmetic in Go wraps around on overflow.
type RollingHash struct {
	base      uint64
	window    []byte // circular buffer holding the last len(window) bytes
	next      int    // index in window where the next byte will be written
	full      bool   // true once window has been filled at least once
	hash      uint64
	outFactor uint64 // base^(windowSize-1): the weight of the oldest byte
}

// NewRollingHash returns a rolling hash over a window of windowSize bytes.
func NewRollingHash(windowSize int) *RollingHash {
	return newRollingHashWithBase(windowSize, defaultBase)
}

// newRollingHashWithBase lets tests use a small base (e.g. 10) so results
// can be checked by hand.
func newRollingHashWithBase(windowSize int, base uint64) *RollingHash {
	outFactor := uint64(1)
	for i := 0; i < windowSize-1; i++ {
		outFactor *= base
	}
	return &RollingHash{
		base:      base,
		window:    make([]byte, windowSize),
		outFactor: outFactor,
	}
}

// Roll pushes one byte into the window and returns the hash of the
// window's current contents. Once the window is full, the oldest byte is
// evicted. Each call is O(1) regardless of window size.
func (r *RollingHash) Roll(in byte) uint64 {
	if r.full {
		out := r.window[r.next] // the oldest byte sits where we're about to write
		r.hash -= uint64(out) * r.outFactor
	}

	r.hash = r.hash*r.base + uint64(in)

	r.window[r.next] = in
	r.next++
	if r.next == len(r.window) {
		r.next = 0
		r.full = true
	}
	return r.hash
}
