// Package store is a content-addressable chunk store: each chunk is saved
// under the SHA-256 hash of its own bytes, so identical chunks are stored
// once and corruption is detectable on every read.
//
// Background: docs/concepts/00c-content-addressable-store.md
package store

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Hash is the SHA-256 digest of a chunk's bytes. It is also the chunk's name.
type Hash [sha256.Size]byte

// String returns the hash as 64 lowercase hex characters.
func (h Hash) String() string {
	return hex.EncodeToString(h[:])
}

// ErrCorrupt means a chunk's bytes on disk no longer hash to its name.
var ErrCorrupt = errors.New("store: chunk content does not match its hash")

// Store keeps each chunk in its own file at dir/chunks/<hh>/<hash>, where
// <hh> is the first two hex characters of the hash.
//
// Not safe for concurrent use: call it from one goroutine at a time.
type Store struct {
	dir string
}

// Open returns a Store rooted at dir, creating the directory if needed.
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(filepath.Join(dir, "chunks"), 0o755); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

func (s *Store) path(h Hash) string {
	name := h.String()
	return filepath.Join(s.dir, "chunks", name[:2], name)
}

// Put stores data and returns its hash. The bool is true if the chunk was
// newly written, false if an identical chunk was already stored (dedup hit).
func (s *Store) Put(data []byte) (Hash, bool, error) {
	var h Hash = sha256.Sum256(data)
	p := s.path(h)

	_, statErr := os.Stat(p)
	if statErr == nil {
		return h, false, nil
	}
	if !errors.Is(statErr, fs.ErrNotExist) {
		return h, false, statErr
	}

	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return h, false, err
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		return h, false, err
	}
	return h, true, nil
}

// Get returns the chunk named h after verifying its bytes still hash to h.
// A missing chunk returns an error matching fs.ErrNotExist.
func (s *Store) Get(h Hash) ([]byte, error) {
	data, err := os.ReadFile(s.path(h))
	if err != nil {
		return nil, err
	}
	if sha256.Sum256(data) != h {
		return nil, fmt.Errorf("%w: %s", ErrCorrupt, h)
	}
	return data, nil
}
