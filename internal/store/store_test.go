package store

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// openTemp opens a Store in a fresh directory that Go deletes after the test.
func openTemp(t *testing.T) *Store {
	t.Helper()
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return s
}

func TestPutThenGetRoundTrips(t *testing.T) {
	s := openTemp(t)
	data := []byte("hello")

	h, stored, err := s.Put(data)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if !stored {
		t.Fatal("first Put reported a dedup hit")
	}
	if h != sha256.Sum256(data) {
		t.Fatalf("hash = %s, want SHA-256 of the data", h)
	}

	got, err := s.Get(h)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("Get = %q, want %q", got, data)
	}
}

func TestPutSameDataTwiceStoresOnce(t *testing.T) {
	s := openTemp(t)
	data := []byte("same bytes")

	h1, _, _ := s.Put(data)
	h2, stored, err := s.Put(data)
	if err != nil {
		t.Fatalf("second Put: %v", err)
	}
	if stored {
		t.Fatal("second Put wrote the chunk again; want a dedup hit")
	}
	if h1 != h2 {
		t.Fatalf("same data gave different hashes: %s vs %s", h1, h2)
	}

	entries, err := os.ReadDir(filepath.Dir(s.path(h1)))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("found %d files, want exactly 1", len(entries))
	}
}

func TestGetMissingChunkIsNotExist(t *testing.T) {
	s := openTemp(t)
	var unknown Hash // all zeros: no real chunk hashes to this

	_, err := s.Get(unknown)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Get(unknown) error = %v, want fs.ErrNotExist", err)
	}
}

// Simulates bit rot or a buggy writer: the file's bytes change on disk.
func TestGetDetectsCorruption(t *testing.T) {
	s := openTemp(t)
	h, _, _ := s.Put([]byte("original"))

	if err := os.WriteFile(s.path(h), []byte("tampered"), 0o644); err != nil {
		t.Fatalf("tamper: %v", err)
	}

	_, err := s.Get(h)
	if !errors.Is(err, ErrCorrupt) {
		t.Fatalf("Get after tamper error = %v, want ErrCorrupt", err)
	}
}
