package worker

import (
	"bytes"
	"io"
)

// maxOutput bounds what is kept of a process's output
const maxOutput = 16 << 20

// capped is a writer that keeps the first maxOutput bytes and drops the rest.
// It never fails a write, so a chatty process runs on until it exits or times out
// instead of dying on a closed pipe. The buffer is a field, not embedded, so that
// io.Copy cannot bypass the cap through bytes.Buffer's ReadFrom.
type capped struct {
	buf bytes.Buffer
}

func (c *capped) Write(p []byte) (int, error) {
	if room := maxOutput - c.buf.Len(); room > 0 {
		c.buf.Write(p[:min(len(p), room)])
	}
	return len(p), nil
}

// ReadFrom keeps what fits under the cap and drains the rest. With it, os/exec copies
// a pipe straight into the buffer instead of through a new 32 KB one per process.
func (c *capped) ReadFrom(r io.Reader) (int64, error) {
	kept, err := c.buf.ReadFrom(io.LimitReader(r, int64(maxOutput-c.buf.Len())))
	if err != nil {
		return kept, err
	}
	dropped, err := io.Copy(io.Discard, r)
	return kept + dropped, err
}

func (c *capped) String() string {
	return c.buf.String()
}
