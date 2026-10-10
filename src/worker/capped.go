package worker

import "bytes"

// maxOutput bounds what is kept of a process's output
const maxOutput = 16 << 20

// capped is a buffer that keeps the first maxOutput bytes and drops the rest.
// It never fails a write, so a chatty process runs on until it exits or times out
// instead of dying on a closed pipe.
type capped struct {
	bytes.Buffer
}

func (c *capped) Write(p []byte) (int, error) {
	if room := maxOutput - c.Len(); room > 0 {
		c.Buffer.Write(p[:min(len(p), room)])
	}
	return len(p), nil
}
