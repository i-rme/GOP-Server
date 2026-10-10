// Package cache keeps compiled scripts on disk, keyed by the hash of their generated source.
//
// The key changes with any edit to a script or to a file it requires, so an
// entry never goes stale and needs no freshness checks: a cached binary
// exists exactly when its source has been built before.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sync"
)

// binaryName is the file a build leaves in its entry directory
const binaryName = "app"

// Cache is a directory with one subdirectory per built source. It is safe for concurrent use.
type Cache struct {
	root string

	mu    sync.Mutex
	locks map[string]*sync.Mutex // per key, so a source is built once however many requests ask
}

// New returns a cache stored in root, which must exist
func New(root string) *Cache {
	return &Cache{root: root, locks: make(map[string]*sync.Mutex)}
}

// Key is the cache key of a source
func Key(source string) string {
	sum := sha256.Sum256([]byte(source))
	return hex.EncodeToString(sum[:])
}

// Get returns the binary for key, calling build to create it unless it exists;
// hit reports whether it did. build gets the entry's directory and the binary path,
// and must create the binary atomically. Concurrent calls for one key build it once.
func (c *Cache) Get(key string, build func(dir, binary string) error) (binary string, hit bool, err error) {
	binary = filepath.Join(c.root, key, binaryName)
	if exists(binary) {
		return binary, true, nil
	}

	unlock := c.lock(key)
	defer unlock()
	if exists(binary) { // built while this call waited
		return binary, false, nil
	}
	return binary, false, build(filepath.Join(c.root, key), binary)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// lock serializes builds of one key; call the returned function to unlock
func (c *Cache) lock(key string) (unlock func()) {
	c.mu.Lock()
	lock, ok := c.locks[key]
	if !ok {
		lock = new(sync.Mutex)
		c.locks[key] = lock
	}
	c.mu.Unlock()

	lock.Lock()
	return lock.Unlock
}
