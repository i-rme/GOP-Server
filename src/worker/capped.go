package worker

import "bytes"

// maxOutput bounds what is kept of a process's output
const maxOutput = 16 << 20

// capped is a writer that keeps the first maxOutput bytes and drops the rest.
// It never fails a write, so a chatty process runs on until it exits or times out
// instead of dying on a closed pipe. The buffer is a field, not embedded, so that
// io.Copy cannot bypass Write through bytes.Buffer's ReadFrom.
type capped struct {
	buf bytes.Buffer
}

func (c *capped) Write(p []byte) (int, error) {
	if room := maxOutput - c.buf.Len(); room > 0 {
		c.buf.Write(p[:min(len(p), room)])
	}
	return len(p), nil
}

func (c *capped) String() string {
	return c.buf.String()
}
