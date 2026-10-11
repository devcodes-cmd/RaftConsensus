package chunker

import (
	"bufio"
	"io"
)

// Config controls where chunk boundaries may fall.
type Config struct {
	MinSize    int  // no boundary is considered before this many bytes
	MaxSize    int  // a boundary is forced at this many bytes
	MaskBits   uint // cut when the top MaskBits bits of the hash are all zero
	WindowSize int  // number of trailing bytes the rolling hash looks at
}

// DefaultConfig gives an expected chunk size of about
// MinSize + 2^MaskBits = 2 KiB + 8 KiB = 10 KiB.
var DefaultConfig = Config{
	MinSize:    2 * 1024,
	MaxSize:    64 * 1024,
	MaskBits:   13,
	WindowSize: 48,
}

// Chunker reads a stream and hands out one content-defined chunk at a time,
// so arbitrarily large files never need to fit in memory.
type Chunker struct {
	cfg   Config
	src   *bufio.Reader
	hash  *RollingHash
	shift uint // 64 - MaskBits: moves the top MaskBits bits down to the bottom
}

// New returns a Chunker that reads from r.
func New(r io.Reader, cfg Config) *Chunker {
	return &Chunker{
		cfg:   cfg,
		src:   bufio.NewReader(r),
		hash:  NewRollingHash(cfg.WindowSize),
		shift: 64 - cfg.MaskBits,
	}
}

// Next returns the next chunk, or io.EOF once the input is exhausted.
// The rolling hash is not reset between chunks: after WindowSize bytes its
// value depends only on the last WindowSize bytes, and MinSize > WindowSize.
func (c *Chunker) Next() ([]byte, error) {
	chunk := make([]byte, 0, c.cfg.MaxSize)
	for {
		b, err := c.src.ReadByte()
		if err == io.EOF {
			if len(chunk) == 0 {
				return nil, io.EOF
			}
			return chunk, nil // final, possibly short, chunk
		}
		if err != nil {
			return nil, err
		}

		chunk = append(chunk, b)
		h := c.hash.Roll(b)

		if len(chunk) >= c.cfg.MaxSize {
			return chunk, nil
		}
		if len(chunk) >= c.cfg.MinSize && h>>c.shift == 0 {
			return chunk, nil
		}
	}
}
