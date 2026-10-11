package chunker

import (
	"math/rand"
	"testing"
)

// hashFromScratch computes the polynomial hash of window directly, with no
// rolling. It is the reference the rolling version must agree with.
func hashFromScratch(window []byte, base uint64) uint64 {
	var h uint64
	for _, b := range window {
		h = h*base + uint64(b)
	}
	return h
}

// The decimal example from the concept doc: window 3, base 10.
func TestRollingHashDecimalExample(t *testing.T) {
	r := newRollingHashWithBase(3, 10)

	r.Roll(1)
	r.Roll(2)
	if got := r.Roll(3); got != 123 {
		t.Fatalf("hash of [1 2 3] = %d, want 123", got)
	}
	if got := r.Roll(4); got != 234 {
		t.Fatalf("hash of [2 3 4] = %d, want 234", got)
	}
}

// After every Roll, the rolling hash must equal a from-scratch hash of the
// same window. This is the core correctness property.
func TestRollingHashMatchesFromScratch(t *testing.T) {
	const windowSize = 48
	rng := rand.New(rand.NewSource(1)) // fixed seed: failures are reproducible
	data := make([]byte, 10_000)
	rng.Read(data)

	r := NewRollingHash(windowSize)
	for i, b := range data {
		got := r.Roll(b)

		start := i + 1 - windowSize
		if start < 0 {
			start = 0 // window not full yet: it holds everything seen so far
		}
		want := hashFromScratch(data[start:i+1], defaultBase)

		if got != want {
			t.Fatalf("at byte %d: rolling=%x, from scratch=%x", i, got, want)
		}
	}
}
