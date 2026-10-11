package chunker

import (
	"bytes"
	"errors"
	"io"
	"math/rand"
	"testing"
)

// splitAll runs the chunker to completion and returns every chunk.
func splitAll(t *testing.T, data []byte, cfg Config) [][]byte {
	t.Helper()
	c := New(bytes.NewReader(data), cfg)
	var chunks [][]byte
	for {
		chunk, err := c.Next()
		if errors.Is(err, io.EOF) {
			return chunks
		}
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		chunks = append(chunks, chunk)
	}
}

func randomBytes(seed int64, n int) []byte {
	data := make([]byte, n)
	rand.New(rand.NewSource(seed)).Read(data)
	return data
}

func TestEmptyInputHasNoChunks(t *testing.T) {
	if chunks := splitAll(t, nil, DefaultConfig); len(chunks) != 0 {
		t.Fatalf("got %d chunks, want 0", len(chunks))
	}
}

func TestSingleByteIsOneChunk(t *testing.T) {
	chunks := splitAll(t, []byte{42}, DefaultConfig)
	if len(chunks) != 1 || !bytes.Equal(chunks[0], []byte{42}) {
		t.Fatalf("got %v, want one chunk [42]", chunks)
	}
}

// All-identical bytes never trigger the hash rule (or trigger it at every
// position), so only MaxSize (or MinSize) can produce boundaries.
func TestIdenticalBytesAreCutByMinOrMax(t *testing.T) {
	data := bytes.Repeat([]byte{0}, 200*1024)
	chunks := splitAll(t, data, DefaultConfig)
	for i, chunk := range chunks[:len(chunks)-1] {
		if len(chunk) != DefaultConfig.MaxSize && len(chunk) != DefaultConfig.MinSize {
			t.Fatalf("chunk %d has size %d, want MinSize or MaxSize", i, len(chunk))
		}
	}
	t.Logf("200 KiB of zeros -> %d chunks, first size %d", len(chunks), len(chunks[0]))
}

// Joining the chunks back together must reproduce the input exactly, and
// every chunk except the last must respect MinSize and MaxSize.
func TestRoundTripAndSizeBounds(t *testing.T) {
	data := randomBytes(2, 8*1024*1024)
	chunks := splitAll(t, data, DefaultConfig)

	if joined := bytes.Join(chunks, nil); !bytes.Equal(joined, data) {
		t.Fatal("joined chunks differ from input")
	}

	smallest, largest := len(chunks[0]), len(chunks[0])
	for i, chunk := range chunks {
		if len(chunk) > DefaultConfig.MaxSize {
			t.Fatalf("chunk %d too large: %d", i, len(chunk))
		}
		if i < len(chunks)-1 && len(chunk) < DefaultConfig.MinSize {
			t.Fatalf("chunk %d too small: %d", i, len(chunk))
		}
		smallest = min(smallest, len(chunk))
		largest = max(largest, len(chunk))
	}
	t.Logf("8 MiB random: %d chunks, avg %d B, min %d B, max %d B",
		len(chunks), len(data)/len(chunks), smallest, largest)
}

// The reason content-defined chunking exists: inserting one byte at the
// start should change only the first chunk, not every chunk.
func TestInsertAtStartKeepsLaterChunks(t *testing.T) {
	original := randomBytes(3, 1024*1024)
	modified := append([]byte{'X'}, original...)

	seen := make(map[string]bool)
	for _, chunk := range splitAll(t, original, DefaultConfig) {
		seen[string(chunk)] = true
	}

	modChunks := splitAll(t, modified, DefaultConfig)
	shared := 0
	for _, chunk := range modChunks {
		if seen[string(chunk)] {
			shared++
		}
	}

	t.Logf("after 1-byte insert: %d of %d chunks unchanged", shared, len(modChunks))
	if shared < len(modChunks)-1 {
		t.Fatalf("only %d of %d chunks shared; want all but the first", shared, len(modChunks))
	}
}
