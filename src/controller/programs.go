package controller

import (
	"os"
	"slices"
	"sync"
	"time"

	"github.com/i-rme/gop-server/src/parser"
)

// settle is how old a file must be before its stamp is trusted. An edit made within
// the file system's timestamp granularity may leave the modification time unchanged.
const settle = 2 * time.Second

// programs remembers the program parsed from each script, with a stamp of every
// file it was read from. A request for a script whose files are unchanged reuses
// the program instead of reading and translating them again. The cache key is
// still the hash of the program, so this only saves work: it never picks a binary.
type programs struct {
	mu     sync.Mutex
	parsed map[string]parsed // by script path
}

type parsed struct {
	program *parser.Program
	stamps  []stamp
}

// stamp is what changes when a file is edited
type stamp struct {
	size, modified int64
}

// get returns the program of the script at path, calling parse unless every file
// it was read from last time is unchanged
func (p *programs) get(path string, parse func() (*parser.Program, error)) (*parser.Program, error) {
	p.mu.Lock()
	last, ok := p.parsed[path]
	p.mu.Unlock()
	if ok && slices.Equal(stamps(last.program.Files), last.stamps) {
		return last.program, nil
	}

	program, err := parse()
	if err != nil {
		return nil, err
	}
	if s := stamps(program.Files); settled(s) {
		p.mu.Lock()
		if p.parsed == nil {
			p.parsed = make(map[string]parsed)
		}
		p.parsed[path] = parsed{program, s}
		p.mu.Unlock()
	}
	return program, nil
}

// stamps returns the stamp of each file, or nil if one cannot be read
func stamps(files []string) []stamp {
	s := make([]stamp, len(files))
	for i, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			return nil
		}
		s[i] = stamp{info.Size(), info.ModTime().UnixNano()}
	}
	return s
}

// settled reports whether every stamp is old enough to be trusted
func settled(s []stamp) bool {
	if s == nil {
		return false
	}
	before := time.Now().Add(-settle).UnixNano()
	for _, st := range s {
		if st.modified > before {
			return false
		}
	}
	return true
}
