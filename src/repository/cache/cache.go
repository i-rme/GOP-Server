// Package cache remembers the compiled binary of each script.
package cache

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Cache maps a script path to its compiled binary. It is safe for concurrent use.
type Cache struct {
	mu      sync.Mutex
	entries map[string]entry
}

type entry struct {
	binary string
	built  time.Time
}

// New returns an empty cache
func New() *Cache {
	return &Cache{entries: make(map[string]entry)}
}

// Put records the binary that `go run --work` built for source in workDir;
// name is the file go run was given, which names the binary
func (c *Cache) Put(source, workDir, name string) {
	binary := filepath.Join(workDir, "b001", "exe", strings.TrimSuffix(filepath.Base(name), filepath.Ext(name)))
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[filepath.Clean(source)] = entry{binary: binary, built: time.Now()}
}

// GetIfFresh returns the binary built for source, or "" when there is none
// or source changed since it was built
func (c *Cache) GetIfFresh(source string) string {
	info, err := os.Stat(source)
	if err != nil {
		return ""
	}
	c.mu.Lock()
	e := c.entries[filepath.Clean(source)]
	c.mu.Unlock()
	if e.built.After(info.ModTime()) {
		return e.binary
	}
	return ""
}
